package consumer_test

import (
	"context"
	"errors"
	"io"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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
	return kafka.Message{}, ctx.Err()
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
		return kafka.Message{}, ctx.Err()
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

// scriptedReader returns a sequence of (message, err) entries in order.
// Every call to FetchMessage consumes the next scripted entry. When entries
// are exhausted, FetchMessage blocks on ctx.Done() so the test can cancel.
// Close() counts invocations so tests can assert dead readers are closed.
type scriptedReader struct {
	mu        sync.Mutex
	script    []scriptedFetch
	closes    int
	committed []kafka.Message
}

type scriptedFetch struct {
	msg kafka.Message
	err error
}

func (r *scriptedReader) FetchMessage(ctx context.Context) (kafka.Message, error) {
	r.mu.Lock()
	if len(r.script) > 0 {
		next := r.script[0]
		// If this is the last entry and it's an error, leave it in place so
		// a retry loop re-pulling the same broken reader keeps seeing the
		// same error until the outer loop recreates the reader. If it's a
		// successful message, consume it so we don't redeliver — subsequent
		// calls will block on ctx like a normally-idle reader.
		if len(r.script) > 1 || next.err == nil {
			r.script = r.script[1:]
		}
		r.mu.Unlock()
		return next.msg, next.err
	}
	r.mu.Unlock()
	<-ctx.Done()
	return kafka.Message{}, ctx.Err()
}

func (r *scriptedReader) CommitMessages(_ context.Context, msgs ...kafka.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.committed = append(r.committed, msgs...)
	return nil
}

func (r *scriptedReader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closes++
	return nil
}

func (r *scriptedReader) Closes() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.closes
}

func (r *scriptedReader) Committed() []kafka.Message {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]kafka.Message, len(r.committed))
	copy(out, r.committed)
	return out
}

// readerFactory returns a ReaderProducer that hands out readers from the
// provided slice in order, one per call. Tests use this to observe reader
// recreation — the outer Consumer loop should request a fresh reader after
// each fetch-loop exit.
func readerFactory(t *testing.T, readers ...consumer.KafkaReader) consumer.ReaderProducer {
	t.Helper()
	idx := 0
	var mu sync.Mutex
	return func(_ kafka.ReaderConfig) consumer.KafkaReader {
		mu.Lock()
		defer mu.Unlock()
		if idx >= len(readers) {
			t.Fatalf("factory asked for reader #%d but only %d were provided", idx+1, len(readers))
		}
		r := readers[idx]
		idx++
		return r
	}
}

func TestRecreatesReaderOnEOF(t *testing.T) {
	consumer.ResetInstance()

	l, _ := test.NewNullLogger()
	wg := &sync.WaitGroup{}
	otel.SetTracerProvider(&MockTracerProvider{})

	// First reader: returns EOF immediately. Second reader: delivers one message
	// and then blocks on ctx.
	r1 := &scriptedReader{script: []scriptedFetch{{err: io.EOF}}}
	r2 := &scriptedReader{script: []scriptedFetch{{msg: kafka.Message{Value: []byte("after-recreate")}}}}

	rp := consumer.ConfigReaderProducer(readerFactory(t, r1, r2))

	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		wg.Wait()
	}()

	cm := consumer.GetManager(rp)
	c := consumer.NewConfig([]string{""}, "test-consumer", "eof-topic", "test-group")
	cm.AddConsumer(l, ctx, wg)(c)

	handlerDone := make(chan struct{})
	_, _ = cm.RegisterHandler("eof-topic", func(_ logrus.FieldLogger, _ context.Context, _ kafka.Message) (bool, error) {
		close(handlerDone)
		return true, nil
	})

	select {
	case <-handlerDone:
	case <-time.After(5 * time.Second):
		t.Fatal("handler was never invoked on recreated reader")
	}

	// r1 must have been closed once the outer loop noticed EOF.
	if r1.Closes() != 1 {
		t.Fatalf("expected r1 to be closed exactly once, got %d", r1.Closes())
	}

	// Observable state should reflect the recreate.
	snaps := cm.Consumers()
	if len(snaps) != 1 {
		t.Fatalf("expected 1 consumer, got %d", len(snaps))
	}
	s := snaps[0].Snapshot()
	if s.RecreateCount < 1 {
		t.Fatalf("expected recreateCount >= 1 after EOF, got %d", s.RecreateCount)
	}
}

func TestContextCancelDoesNotRecreate(t *testing.T) {
	consumer.ResetInstance()

	l, _ := test.NewNullLogger()
	wg := &sync.WaitGroup{}
	otel.SetTracerProvider(&MockTracerProvider{})

	// One reader is provided. If the outer loop misbehaves and asks for a
	// second reader after ctx-cancel, readerFactory will Fatal the test.
	r1 := &scriptedReader{}
	rp := consumer.ConfigReaderProducer(readerFactory(t, r1))

	ctx, cancel := context.WithCancel(context.Background())

	cm := consumer.GetManager(rp)
	c := consumer.NewConfig([]string{""}, "test-consumer", "cancel-topic", "test-group")
	cm.AddConsumer(l, ctx, wg)(c)

	// Give the consumer a moment to enter its fetch loop.
	time.Sleep(50 * time.Millisecond)
	cancel()
	wg.Wait()

	if r1.Closes() != 1 {
		t.Fatalf("expected reader closed exactly once on ctx-cancel, got %d", r1.Closes())
	}
}

func TestRetryExhaustionRecreatesReader(t *testing.T) {
	consumer.ResetInstance()

	l, _ := test.NewNullLogger()
	wg := &sync.WaitGroup{}
	otel.SetTracerProvider(&MockTracerProvider{})

	// r1: returns a transient error on every fetch. Inner retry (3 attempts)
	// exhausts, outer loop closes r1 and requests r2.
	transient := errors.New("transient broker failure")
	r1 := &scriptedReader{script: []scriptedFetch{
		{err: transient},
		{err: transient},
		{err: transient},
	}}
	r2 := &scriptedReader{script: []scriptedFetch{{msg: kafka.Message{Value: []byte("recovered")}}}}

	rp := consumer.ConfigReaderProducer(readerFactory(t, r1, r2))

	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		wg.Wait()
	}()

	cm := consumer.GetManager(rp)
	c := consumer.NewConfig([]string{""}, "test-consumer", "retry-topic", "test-group")
	cm.AddConsumer(l, ctx, wg)(c)

	handlerDone := make(chan struct{})
	_, _ = cm.RegisterHandler("retry-topic", func(_ logrus.FieldLogger, _ context.Context, _ kafka.Message) (bool, error) {
		close(handlerDone)
		return true, nil
	})

	select {
	case <-handlerDone:
	case <-time.After(10 * time.Second):
		t.Fatal("handler was never invoked after retry exhaustion recreate")
	}

	if r1.Closes() != 1 {
		t.Fatalf("expected r1 closed once after retry exhaustion, got %d", r1.Closes())
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

func TestFetchTimeoutTicksWithoutRecreate(t *testing.T) {
	consumer.ResetInstance()

	l, _ := test.NewNullLogger()
	wg := &sync.WaitGroup{}
	otel.SetTracerProvider(&MockTracerProvider{})

	// Empty scriptedReader always blocks on ctx — every FetchMessage call
	// returns DeadlineExceeded when the per-call deadline fires.
	r1 := &scriptedReader{}
	rp := consumer.ConfigReaderProducer(readerFactory(t, r1))

	ctx, cancel := context.WithCancel(context.Background())
	defer wg.Wait()

	cm := consumer.GetManager(rp)
	c := consumer.NewConfig([]string{""}, "tick-consumer", "tick-topic", "test-group")
	cm.AddConsumer(l, ctx, wg)(
		c,
		consumer.SetFetchTimeout(50*time.Millisecond),
		consumer.SetMaxConsecutiveTimeouts(3),
	)

	// Wait long enough for at least one deadline to fire (50ms) but well
	// short of three (150ms).
	time.Sleep(75 * time.Millisecond)

	snaps := cm.Consumers()
	if len(snaps) != 1 {
		t.Fatalf("expected 1 consumer, got %d", len(snaps))
	}
	s := snaps[0].Snapshot()

	if s.ConsecutiveTimeouts < 1 {
		t.Fatalf("expected ConsecutiveTimeouts >= 1 after a tick, got %d", s.ConsecutiveTimeouts)
	}
	if s.RecreateCount != 0 {
		t.Fatalf("expected RecreateCount == 0 after a single tick, got %d", s.RecreateCount)
	}
	if s.LastError != "" {
		t.Fatalf("expected LastError empty (idle is not an error), got %q", s.LastError)
	}
	if s.LastTimeoutAt.IsZero() {
		t.Fatal("expected LastTimeoutAt to be set after a tick")
	}

	cancel()
	wg.Wait()

	if r1.Closes() != 1 {
		t.Fatalf("expected reader closed exactly once on ctx-cancel, got %d", r1.Closes())
	}
}

func TestFetchTimeoutEscalatesAfterMaxToWedge(t *testing.T) {
	consumer.ResetInstance()

	// Capture the logger hook so we can verify the Warn-level wedge log
	// fired with the sentinel message. We can't observe LastError in a
	// snapshot because both onReaderCreated (on the new reader) and
	// recordFetch (on the new reader's first message) clear it before the
	// test's handler-invocation sync point — by design (PRD §4.5).
	l, hook := test.NewNullLogger()
	wg := &sync.WaitGroup{}
	otel.SetTracerProvider(&MockTracerProvider{})

	// r1: empty — every FetchMessage hits the deadline. After 3 ticks
	// runFetchLoop returns errFetchWedged, the outer loop closes r1 and
	// requests r2.
	// r2: delivers one message. Handler invocation is the signal that the
	// recreate path completed.
	r1 := &scriptedReader{}
	r2 := &scriptedReader{script: []scriptedFetch{{msg: kafka.Message{Value: []byte("after-wedge")}}}}
	rp := consumer.ConfigReaderProducer(readerFactory(t, r1, r2))

	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		wg.Wait()
	}()

	// Goroutine-leak guard (risks R2): capture before the consumer starts.
	goroutinesBefore := runtime.NumGoroutine()

	cm := consumer.GetManager(rp)
	c := consumer.NewConfig([]string{""}, "wedge-consumer", "wedge-topic", "test-group")
	cm.AddConsumer(l, ctx, wg)(
		c,
		consumer.SetFetchTimeout(50*time.Millisecond),
		consumer.SetMaxConsecutiveTimeouts(3),
	)

	handlerDone := make(chan struct{})
	_, _ = cm.RegisterHandler("wedge-topic", func(_ logrus.FieldLogger, _ context.Context, _ kafka.Message) (bool, error) {
		close(handlerDone)
		return true, nil
	})

	select {
	case <-handlerDone:
	case <-time.After(2 * time.Second):
		t.Fatal("handler was never invoked on recreated reader after wedge")
	}

	if r1.Closes() != 1 {
		t.Fatalf("expected r1 closed exactly once after wedge, got %d", r1.Closes())
	}

	snaps := cm.Consumers()
	if len(snaps) != 1 {
		t.Fatalf("expected 1 consumer, got %d", len(snaps))
	}
	s := snaps[0].Snapshot()

	if s.RecreateCount < 1 {
		t.Fatalf("expected RecreateCount >= 1 after wedge recreate, got %d", s.RecreateCount)
	}
	// Counter must be reset by onReaderCreated for the new reader.
	if s.ConsecutiveTimeouts != 0 {
		t.Fatalf("expected ConsecutiveTimeouts reset to 0 on new reader, got %d", s.ConsecutiveTimeouts)
	}

	// Verify the Warn log fired with the wedge message (PRD §4.2). This
	// is the durable signal that the sentinel was recorded; lastError
	// would have been cleared by the time the test observes the snapshot.
	foundWedgeWarn := false
	for _, e := range hook.AllEntries() {
		if e.Level == logrus.WarnLevel &&
			strings.Contains(e.Message, "FetchMessage wedged") &&
			strings.Contains(e.Message, "wedge-topic") &&
			strings.Contains(e.Message, "test-group") {
			foundWedgeWarn = true
			break
		}
	}
	if !foundWedgeWarn {
		t.Fatalf("expected one Warn log containing 'FetchMessage wedged' with topic+group, got entries: %v", hook.AllEntries())
	}

	// Goroutine-leak guard: settle then compare. If FetchMessage on r1 did
	// not honor ctx cancellation, leaked goroutines accumulate here.
	time.Sleep(50 * time.Millisecond)
	goroutinesAfter := runtime.NumGoroutine()
	if delta := goroutinesAfter - goroutinesBefore; delta > 5 {
		t.Fatalf("goroutine leak suspected: before=%d after=%d delta=%d (>5)",
			goroutinesBefore, goroutinesAfter, delta)
	}
}

// alternatingReader returns DeadlineExceeded on odd-numbered FetchMessage
// calls (1st, 3rd, 5th, ...) and a scripted message on even-numbered calls.
// Used to exercise the timeout-success-timeout-success cycle that should
// keep consecutiveTimeouts pinned at 0 across many iterations.
type alternatingReader struct {
	mu        sync.Mutex
	calls     int
	committed []kafka.Message
	closes    int
}

func (r *alternatingReader) FetchMessage(ctx context.Context) (kafka.Message, error) {
	r.mu.Lock()
	r.calls++
	n := r.calls
	r.mu.Unlock()

	if n%2 == 1 {
		<-ctx.Done()
		return kafka.Message{}, ctx.Err()
	}
	return kafka.Message{Value: []byte("ok")}, nil
}

func (r *alternatingReader) CommitMessages(_ context.Context, msgs ...kafka.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.committed = append(r.committed, msgs...)
	return nil
}

func (r *alternatingReader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closes++
	return nil
}

func (r *alternatingReader) Closes() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.closes
}

func (r *alternatingReader) Committed() []kafka.Message {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]kafka.Message, len(r.committed))
	copy(out, r.committed)
	return out
}

func TestFetchTimeoutResetsOnSuccessfulFetch(t *testing.T) {
	consumer.ResetInstance()

	l, _ := test.NewNullLogger()
	wg := &sync.WaitGroup{}
	otel.SetTracerProvider(&MockTracerProvider{})

	r := &alternatingReader{}
	rp := consumer.ConfigReaderProducer(readerFactory(t, r))

	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		wg.Wait()
	}()

	cm := consumer.GetManager(rp)
	c := consumer.NewConfig([]string{""}, "reset-consumer", "reset-topic", "test-group")
	cm.AddConsumer(l, ctx, wg)(
		c,
		consumer.SetFetchTimeout(50*time.Millisecond),
		consumer.SetMaxConsecutiveTimeouts(3),
	)

	handlerInvocations := atomic.Int32{}
	gotThree := make(chan struct{})
	_, _ = cm.RegisterHandler("reset-topic", func(_ logrus.FieldLogger, _ context.Context, _ kafka.Message) (bool, error) {
		if handlerInvocations.Add(1) == 3 {
			close(gotThree)
		}
		return true, nil
	})

	// 3 successes interleaved with 3 timeouts ≈ 200ms wall-clock.
	select {
	case <-gotThree:
	case <-time.After(2 * time.Second):
		t.Fatalf("expected 3 handler invocations, got %d", handlerInvocations.Load())
	}

	snaps := cm.Consumers()
	s := snaps[0].Snapshot()

	if s.RecreateCount != 0 {
		t.Fatalf("expected RecreateCount == 0 (counter resets between successes), got %d", s.RecreateCount)
	}
	if s.ConsecutiveTimeouts != 0 {
		t.Fatalf("expected ConsecutiveTimeouts == 0 after a success, got %d", s.ConsecutiveTimeouts)
	}
	if r.Closes() != 0 {
		t.Fatalf("expected reader to remain open across timeout/success cycles, got Closes=%d", r.Closes())
	}
}

