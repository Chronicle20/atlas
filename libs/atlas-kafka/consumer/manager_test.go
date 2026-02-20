package consumer_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type MockReader struct {
	msg       kafka.Message
	read      bool
	committed []kafka.Message
	mu        sync.Mutex
}

func (r *MockReader) FetchMessage(ctx context.Context) (kafka.Message, error) {
	if !r.read {
		r.read = true
		return r.msg, nil
	}

	<-ctx.Done()
	return kafka.Message{}, context.Canceled
}

func (r *MockReader) CommitMessages(_ context.Context, msgs ...kafka.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.committed = append(r.committed, msgs...)
	return nil
}

func (r *MockReader) Committed() []kafka.Message {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]kafka.Message, len(r.committed))
	copy(result, r.committed)
	return result
}

func (r *MockReader) Close() error {
	return nil
}

func SimpleMockReader(msg kafka.Message) *MockReader {
	return &MockReader{msg: msg}
}

type MockSpan struct {
	trace.Span
	spanContext trace.SpanContext
}

func (ms *MockSpan) SpanContext() trace.SpanContext {
	return ms.spanContext
}

func (ms *MockSpan) IsRecording() bool {
	return true
}

func (ms *MockSpan) End(_ ...trace.SpanEndOption) {
}

func (ms *MockSpan) RecordError(_ error, _ ...trace.EventOption) {
}

type MockTracer struct {
	trace.Tracer
	StartedSpans []*MockSpan
}

func (mt *MockTracer) Start(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
	spanContext := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
		SpanID:     trace.SpanID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		TraceFlags: trace.FlagsSampled,
	})
	mockSpan := &MockSpan{spanContext: spanContext}
	return trace.ContextWithSpan(ctx, mockSpan), mockSpan
}

type MockTracerProvider struct {
	trace.TracerProvider
	tracer *MockTracer
}

func (m MockTracerProvider) Tracer(_ string, _ ...trace.TracerOption) trace.Tracer {
	if m.tracer == nil {
		m.tracer = &MockTracer{}
	}
	return m.tracer
}

type ChannelMockReader struct {
	msgCh     chan kafka.Message
	committed []kafka.Message
	mu        sync.Mutex
}

func (r *ChannelMockReader) FetchMessage(ctx context.Context) (kafka.Message, error) {
	select {
	case m := <-r.msgCh:
		return m, nil
	case <-ctx.Done():
		return kafka.Message{}, context.Canceled
	}
}

func (r *ChannelMockReader) CommitMessages(_ context.Context, msgs ...kafka.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.committed = append(r.committed, msgs...)
	return nil
}

func (r *ChannelMockReader) Committed() []kafka.Message {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]kafka.Message, len(r.committed))
	copy(result, r.committed)
	return result
}

func (r *ChannelMockReader) Close() error {
	return nil
}

func TestGracefulShutdown(t *testing.T) {
	consumer.ResetInstance()

	l, _ := test.NewNullLogger()
	wg := &sync.WaitGroup{}

	otel.SetTracerProvider(&MockTracerProvider{})

	msgCh := make(chan kafka.Message, 1)
	msgCh <- kafka.Message{Value: []byte("test")}

	rp := consumer.ConfigReaderProducer(func(config kafka.ReaderConfig) consumer.KafkaReader {
		return &ChannelMockReader{msgCh: msgCh}
	})

	var handlerCompleted atomic.Bool

	ctx, cancel := context.WithCancel(context.Background())

	cm := consumer.GetManager(rp)
	c := consumer.NewConfig([]string{""}, "test-consumer", "test-topic", "test-group")
	cm.AddConsumer(l, ctx, wg)(c)

	handlerStarted := make(chan struct{})
	_, _ = cm.RegisterHandler("test-topic", func(l logrus.FieldLogger, ctx context.Context, msg kafka.Message) (bool, error) {
		close(handlerStarted)
		time.Sleep(200 * time.Millisecond)
		handlerCompleted.Store(true)
		return true, nil
	})

	// Wait for handler to begin executing
	<-handlerStarted

	// Cancel context to trigger shutdown
	cancel()

	// Wait for consumer to finish — this should block until handler completes
	wg.Wait()

	if !handlerCompleted.Load() {
		t.Fatal("Expected handler to complete before shutdown finished")
	}
}

func TestSpanPropagation(t *testing.T) {
	consumer.ResetInstance()

	l, _ := test.NewNullLogger()
	wg := &sync.WaitGroup{}

	otel.SetTracerProvider(&MockTracerProvider{})
	otel.SetTextMapPropagator(propagation.TraceContext{})

	ictx, ispan := otel.GetTracerProvider().Tracer("atlas-kafka").Start(context.Background(), "test-span")

	msg := kafka.Message{Value: []byte("this is a test")}
	msg, err := model.Map(producer.DecorateHeaders(producer.SpanHeaderDecorator(ictx)))(model.FixedProvider(msg))()
	if err != nil {
		t.Fatalf("Unable to prepare headers for test.")
	}

	rp := consumer.ConfigReaderProducer(func(config kafka.ReaderConfig) consumer.KafkaReader {
		return SimpleMockReader(msg)
	})

	errChan := make(chan error)

	cm := consumer.GetManager(rp)
	c := consumer.NewConfig([]string{""}, "test-consumer", "test-topic", "test-group")
	cm.AddConsumer(l, context.Background(), wg)(c, consumer.SetHeaderParsers(consumer.SpanHeaderParser))
	_, _ = cm.RegisterHandler("test-topic", func(l logrus.FieldLogger, ctx context.Context, msg kafka.Message) (bool, error) {
		span := trace.SpanFromContext(ctx)
		if !span.SpanContext().TraceID().IsValid() {
			errChan <- errors.New("invalid trace id")
		}
		if span.SpanContext().TraceID() != ispan.SpanContext().TraceID() {
			errChan <- errors.New("invalid trace id")
		}

		errChan <- nil
		return true, nil
	})

	err = <-errChan
	if err != nil {
		t.Fatal(err.Error())
	}

}

func TestTenantPropagation(t *testing.T) {
	consumer.ResetInstance()

	l, _ := test.NewNullLogger()
	wg := &sync.WaitGroup{}

	it, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err.Error())
	}
	ictx := tenant.WithContext(context.Background(), it)

	msg := kafka.Message{Value: []byte("this is a test")}
	msg, err = model.Map(producer.DecorateHeaders(producer.TenantHeaderDecorator(ictx)))(model.FixedProvider(msg))()
	if err != nil {
		t.Fatalf("Unable to prepare headers for test.")
	}

	rp := consumer.ConfigReaderProducer(func(config kafka.ReaderConfig) consumer.KafkaReader {
		return SimpleMockReader(msg)
	})

	errChan := make(chan error)

	cm := consumer.GetManager(rp)
	c := consumer.NewConfig([]string{""}, "test-consumer", "test-topic", "test-group")
	cm.AddConsumer(l, context.Background(), wg)(c, consumer.SetHeaderParsers(consumer.TenantHeaderParser))
	_, _ = cm.RegisterHandler("test-topic", func(l logrus.FieldLogger, ctx context.Context, msg kafka.Message) (bool, error) {
		ot, err := tenant.FromContext(ctx)()
		if err != nil {
			errChan <- err
		}
		if !it.Is(ot) {
			errChan <- errors.New("tenant does not match")
		}

		errChan <- nil
		return true, nil
	})

	err = <-errChan
	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestCommitAfterHandlerCompletes(t *testing.T) {
	consumer.ResetInstance()

	l, _ := test.NewNullLogger()
	wg := &sync.WaitGroup{}
	otel.SetTracerProvider(&MockTracerProvider{})

	reader := &ChannelMockReader{msgCh: make(chan kafka.Message, 1)}
	reader.msgCh <- kafka.Message{Value: []byte("commit-test")}

	rp := consumer.ConfigReaderProducer(func(config kafka.ReaderConfig) consumer.KafkaReader {
		return reader
	})

	ctx, cancel := context.WithCancel(context.Background())

	cm := consumer.GetManager(rp)
	c := consumer.NewConfig([]string{""}, "test-consumer", "test-topic", "test-group")
	cm.AddConsumer(l, ctx, wg)(c)

	handlerDone := make(chan struct{})
	_, _ = cm.RegisterHandler("test-topic", func(l logrus.FieldLogger, ctx context.Context, msg kafka.Message) (bool, error) {
		// Verify no commits have happened yet while handler is running
		if len(reader.Committed()) != 0 {
			t.Error("Expected no commits while handler is still running")
		}
		close(handlerDone)
		return true, nil
	})

	<-handlerDone
	// Give the consumer loop time to commit after handler returns
	time.Sleep(50 * time.Millisecond)

	committed := reader.Committed()
	if len(committed) != 1 {
		t.Fatalf("Expected 1 committed message, got %d", len(committed))
	}
	if string(committed[0].Value) != "commit-test" {
		t.Fatalf("Expected committed message value 'commit-test', got '%s'", string(committed[0].Value))
	}

	cancel()
	wg.Wait()
}

func TestHandlerErrorPreventsCommit(t *testing.T) {
	consumer.ResetInstance()

	l, _ := test.NewNullLogger()
	wg := &sync.WaitGroup{}
	otel.SetTracerProvider(&MockTracerProvider{})

	reader := &ChannelMockReader{msgCh: make(chan kafka.Message, 1)}
	reader.msgCh <- kafka.Message{Value: []byte("error-test")}

	rp := consumer.ConfigReaderProducer(func(config kafka.ReaderConfig) consumer.KafkaReader {
		return reader
	})

	ctx, cancel := context.WithCancel(context.Background())

	cm := consumer.GetManager(rp)
	c := consumer.NewConfig([]string{""}, "test-consumer", "test-topic", "test-group")
	cm.AddConsumer(l, ctx, wg)(c)

	handlerDone := make(chan struct{})
	_, _ = cm.RegisterHandler("test-topic", func(l logrus.FieldLogger, ctx context.Context, msg kafka.Message) (bool, error) {
		defer close(handlerDone)
		return true, errors.New("handler failed")
	})

	<-handlerDone
	time.Sleep(50 * time.Millisecond)

	committed := reader.Committed()
	if len(committed) != 0 {
		t.Fatalf("Expected 0 committed messages when handler errors, got %d", len(committed))
	}

	cancel()
	wg.Wait()
}

func TestHandlerPanicPreventsCommit(t *testing.T) {
	consumer.ResetInstance()

	l, _ := test.NewNullLogger()
	wg := &sync.WaitGroup{}
	otel.SetTracerProvider(&MockTracerProvider{})

	reader := &ChannelMockReader{msgCh: make(chan kafka.Message, 2)}
	reader.msgCh <- kafka.Message{Value: []byte("panic-test")}
	reader.msgCh <- kafka.Message{Value: []byte("after-panic")}

	rp := consumer.ConfigReaderProducer(func(config kafka.ReaderConfig) consumer.KafkaReader {
		return reader
	})

	ctx, cancel := context.WithCancel(context.Background())

	cm := consumer.GetManager(rp)
	c := consumer.NewConfig([]string{""}, "test-consumer", "test-topic", "test-group")
	cm.AddConsumer(l, ctx, wg)(c)

	callCount := atomic.Int32{}
	secondHandlerDone := make(chan struct{})
	_, _ = cm.RegisterHandler("test-topic", func(l logrus.FieldLogger, ctx context.Context, msg kafka.Message) (bool, error) {
		count := callCount.Add(1)
		if count == 1 {
			panic("test panic")
		}
		// Second call — consumer survived the panic
		close(secondHandlerDone)
		return true, nil
	})

	<-secondHandlerDone
	time.Sleep(50 * time.Millisecond)

	committed := reader.Committed()
	// First message (panic) should NOT be committed, second message should be committed
	if len(committed) != 1 {
		t.Fatalf("Expected 1 committed message (only the non-panicking one), got %d", len(committed))
	}
	if string(committed[0].Value) != "after-panic" {
		t.Fatalf("Expected committed message to be 'after-panic', got '%s'", string(committed[0].Value))
	}

	cancel()
	wg.Wait()
}

func TestRegisterHandlerUnknownTopicReturnsError(t *testing.T) {
	consumer.ResetInstance()

	cm := consumer.GetManager()
	_, err := cm.RegisterHandler("nonexistent-topic", func(l logrus.FieldLogger, ctx context.Context, msg kafka.Message) (bool, error) {
		return true, nil
	})
	if err == nil {
		t.Fatal("Expected error when registering handler for unknown topic, got nil")
	}
}

func TestMultipleHandlersAllCompleteBeforeCommit(t *testing.T) {
	consumer.ResetInstance()

	l, _ := test.NewNullLogger()
	wg := &sync.WaitGroup{}
	otel.SetTracerProvider(&MockTracerProvider{})

	reader := &ChannelMockReader{msgCh: make(chan kafka.Message, 1)}
	reader.msgCh <- kafka.Message{Value: []byte("multi-handler-test")}

	rp := consumer.ConfigReaderProducer(func(config kafka.ReaderConfig) consumer.KafkaReader {
		return reader
	})

	ctx, cancel := context.WithCancel(context.Background())

	cm := consumer.GetManager(rp)
	c := consumer.NewConfig([]string{""}, "test-consumer", "test-topic", "test-group")
	cm.AddConsumer(l, ctx, wg)(c)

	handler1Done := atomic.Bool{}
	handler2Done := atomic.Bool{}
	allDone := make(chan struct{})

	_, _ = cm.RegisterHandler("test-topic", func(l logrus.FieldLogger, ctx context.Context, msg kafka.Message) (bool, error) {
		time.Sleep(100 * time.Millisecond)
		handler1Done.Store(true)
		return true, nil
	})

	_, _ = cm.RegisterHandler("test-topic", func(l logrus.FieldLogger, ctx context.Context, msg kafka.Message) (bool, error) {
		time.Sleep(50 * time.Millisecond)
		handler2Done.Store(true)
		// Signal that both should be done soon
		go func() {
			time.Sleep(100 * time.Millisecond)
			close(allDone)
		}()
		return true, nil
	})

	<-allDone
	time.Sleep(50 * time.Millisecond)

	if !handler1Done.Load() || !handler2Done.Load() {
		t.Fatal("Expected both handlers to complete")
	}

	committed := reader.Committed()
	if len(committed) != 1 {
		t.Fatalf("Expected 1 committed message after both handlers complete, got %d", len(committed))
	}

	cancel()
	wg.Wait()
}
