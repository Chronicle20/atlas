
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
