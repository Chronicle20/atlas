package consumer_test

import (
	"context"
	"errors"
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
	"sync"
	"testing"
)

type MockReader struct {
	msg  kafka.Message
	read bool
}

func (r MockReader) ReadMessage(ctx context.Context) (kafka.Message, error) {
	if !r.read {
		r.read = true
		return r.msg, nil
	}

	select {
	case <-ctx.Done():
	}

	return kafka.Message{}, context.Canceled
}

func (r MockReader) Close() error {
	return nil
}

func SimpleMockReader(msg kafka.Message) MockReader {
	return MockReader{msg: msg}
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

func (ms *MockSpan) End(options ...trace.SpanEndOption) {
}

func (ms *MockSpan) RecordError(err error, options ...trace.EventOption) {
	// You can record the error or count calls here
}

type MockTracer struct {
	trace.Tracer
	StartedSpans []*MockSpan
}

func (mt *MockTracer) Start(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
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

func (m MockTracerProvider) Tracer(name string, options ...trace.TracerOption) trace.Tracer {
	if m.tracer == nil {
		m.tracer = &MockTracer{}
	}
	return m.tracer
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
		t.Fatalf(err.Error())
	}

}

func TestTenantPropagation(t *testing.T) {
	consumer.ResetInstance()

	l, _ := test.NewNullLogger()
	wg := &sync.WaitGroup{}

	it, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf(err.Error())
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
		t.Fatalf(err.Error())
	}
}
