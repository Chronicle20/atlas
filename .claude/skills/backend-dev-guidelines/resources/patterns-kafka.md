
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

---

## Audit verification — DOM-30

Rule defined in [audit-checklist.md](audit-checklist.md). This section is the
verification procedure. Triggers when a changed package has `producer.go`, or
calls `AndEmit` / `message.Emit` / `producer.ProviderImpl`.

**How to verify.**

1. Grep the changed non-test files for `producer.ProviderImpl(` and
   `producer.Produce(` call sites outside `producer.go` itself.
2. For each, read the enclosing function. The emission belongs inside a
   `message.Emit(p.p)(func(mb *message.Buffer) error { ... })` wrapper — the
   `*AndEmit` shape in [Message Buffer Pattern](#message-buffer-pattern) — so
   that the operation's DB write and its events commit or fail together.

**Pass criteria.** Operations that mutate state emit through `AndEmit` +
`message.Buffer`.

**Documented exception.** A direct producer call on a *post-failure* branch —
one reached only after the operation's transaction has already failed, where
there is no live buffer to attach to — is not a finding. The buffer requirement
exists to keep a write atomic with its side effects; on that branch there is no
write to be atomic with. This is the task-137 ruling (`atlas-notes` /
`saga-orchestrator` CREATE_FAILED notifications). Cite the branch condition as
evidence; a direct producer call on a *success* path is a FAIL.
