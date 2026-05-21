
---
title: Kafka Pattern
description: Event-driven messaging design with producers, consumers, and AndEmit pattern.
---

# Kafka Pattern

Uses Kafka for all inter-service communication.

## Producer Initialization
```go
func ProviderImpl(l logrus.FieldLogger) func(ctx context.Context) func(token string) producer.MessageProducer {
  return func(ctx context.Context) func(token string) producer.MessageProducer {
    sd := producer.SpanHeaderDecorator(ctx)
    td := producer.TenantHeaderDecorator(ctx)
    return func(token string) producer.MessageProducer {
      return producer.Produce(l)(producer.WriterProvider(topic.EnvProvider(l)(token)))(sd, td)
    }
  }
}
```


## Message Buffer Pattern
Accumulate messages and emit atomically.
```go
func (p *ProcessorImpl) OperationAndEmit(params...) error {
  return message.Emit(p.p)(func(mb *message.Buffer) error {

    return p.Operation(mb)(params...)
  })
}
```

## Consumer Pattern (Curried Config)
- Curried builder for consumers
- Attach header parsers for span + tenant
- Decode → handle → call processor

## Producer Stubbing in Tests
Any test package that exercises an emit path (`*AndEmit()` or `message.Emit(...)`) MUST stub the producer. The default writer factory retries failed sends 10× with exponential backoff (~42s per message) when `BOOTSTRAP_SERVERS` is unset, which compounds catastrophically across a test suite.

Install the no-op writer in `TestMain`:
```go
import "github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"

func TestMain(m *testing.M) {
    producertest.InstallNoop()
    os.Exit(m.Run())
}
```

For per-test injection (when the processor exposes `WithProducer(...)`), pass a no-op `producer.Provider` directly. See [Stubbing the Kafka Producer in Tests](testing-guide.md#stubbing-the-kafka-producer-in-tests).
