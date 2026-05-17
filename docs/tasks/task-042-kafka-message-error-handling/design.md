# Kafka Message Error Handling — Design Document

Task: task-042-kafka-message-error-handling
Status: Draft
Created: 2026-05-17

---

## 1. Scope and Framing

The PRD is small and prescriptive: one file change in `libs/atlas-kafka/message/handler.go`, plus a new `handler_test.go`. The design questions are therefore not "what subsystems do we add?" but "what shape does the diff take so it is correct, cheap to maintain, and does not paint future work into a corner?"

Three design surfaces matter:

1. **What we log, and how** — field set, formatting, truncation strategy, and the `%T` trick for the generic type name.
2. **How we make it testable** — capturing logrus output deterministically without dragging in a new dependency or coupling the test to a global logger.
3. **What we deliberately do *not* change** — the `Handler[M]` signature, the `(true, nil)` return contract, the validator-rejection log path, the `adapt[M]` split.

Each of those has at least one viable alternative. We pick one per section, with the rejected options written down so a future task can revisit without re-deriving the reasoning.

## 2. Current State

`libs/atlas-kafka/message/handler.go:42-60`:

```go
func AdaptHandler[M any](config Config[M]) handler.Handler {
    h := func(l logrus.FieldLogger, ctx context.Context, msg kafka.Message) (bool, error) {
        tem := model.Map[kafka.Message, M](adapt[M])(model.FixedProvider(msg))
        m, err := tem()
        if err != nil {
            l.WithError(err).Debugf("Unable to unmarshal message.")
            return true, nil
        }
        process := config.validator(l, ctx, m)
        if !process {
            return true, nil
        }
        config.handler(l, ctx, m)
        return config.persistent, nil
    }
    return h
}
```

Single Debug log, no structured fields, no payload context. `kafka.Message` carries `Topic`, `Partition`, `Offset`, `Value`, `Key`, `Time`, `Headers`. `model.Map` is just a functorial wrapper around `adapt[M]`; the error returned to `AdaptHandler` is the raw `json.Unmarshal` error.

`handler.Handler` (from `libs/atlas-kafka/handler`) is the consumer-facing type:
```go
type Handler func(l logrus.FieldLogger, ctx context.Context, msg kafka.Message) (bool, error)
```

`(bool, error)` semantics in the consumer loop: `bool` is the "persistent" flag (continue listening / one-shot done); a non-nil `error` is the signal to NOT commit the offset. Returning `(true, nil)` after a drop is therefore a deliberate "commit-and-skip" decision, not an accident.

## 3. Design Decision Log

### 3.1 Logging level and message

**Decision:** `l.WithFields(...).WithError(err).Errorf("Failed to unmarshal Kafka message; offset will be committed and the message dropped.")`

**Alternatives considered:**
- `Warn` level. Rejected: this is a silent data-loss event by definition; on-call should see it. `Warn` traditionally signals "self-healing condition," which this isn't.
- `Panic`/`Fatal`. Rejected: kills the whole service for one bad message, exactly the failure mode we are *trying* to avoid.
- A separate message string per call site. Rejected: there is only one call site.

**Why the message text matters:** the literal sentence "offset will be committed and the message dropped" is search-grep-able and unambiguous for an on-call responder. The structured fields tell the *what*; the message tells the *what we did about it*. Both are useful.

### 3.2 Field set

**Decision:** `logrus.Fields{ "topic", "partition", "offset", "payload_size", "payload_preview", "message_type" }`.

Field-by-field:

| Field | Source | Type | Rationale |
| --- | --- | --- | --- |
| `topic` | `msg.Topic` | string | Lets log filtering scope by stream. |
| `partition` | `msg.Partition` | int | Together with offset, uniquely identifies the dropped message in the cluster. |
| `offset` | `msg.Offset` | int64 | Same. Logged as a number; do not stringify. |
| `payload_size` | `len(msg.Value)` | int | Catches "envelope shape" drift even when preview is truncated. |
| `payload_preview` | first N bytes of `msg.Value`, `%q`-quoted | string | Human-debuggable shape. Quoting handles non-UTF8 and embedded newlines/quotes safely. |
| `message_type` | `fmt.Sprintf("%T", *new(M))` | string | Disambiguates which consumer's adapter logged. `*new(M)` works for any `M` including pointers and is a well-known idiom for getting the type name of a type parameter without an instance. |

**Alternatives considered:**
- Logging `msg.Key`. Rejected for v1: most Atlas topics key on tenant or entity id which is not particularly diagnostic for a malformed-payload incident, and keys can themselves be binary. Easy to add later if operators ask.
- Logging `msg.Headers`. Rejected: headers can be large, and Atlas does not put schema versions in them today. Worth revisiting if/when we adopt schema headers.
- Logging `msg.Time`. Rejected: redundant with the logrus entry timestamp under any sane formatter.
- Computing a SHA hash of the payload as an identifier. Rejected: cute, but `topic+partition+offset` already uniquely identifies the message in the cluster, so a hash is pure overhead.

### 3.3 Payload preview byte budget

**Decision:** `const previewMax = 256` declared as a package-level untyped const in `handler.go`.

256 bytes catches enough of any JSON envelope to see `{"type":"...","data":{...}` and the first few fields, which is what an on-call responder needs to identify the producer. It is small enough that an at-rest log line stays well under typical ECS / Loki single-line caps even when JSON-escaped (worst case ~4× expansion via `%q` on non-printable bytes → ~1KB, which is fine).

**Alternatives considered:**
- 128. Rejected: too aggressive; a typical tenant-tagged envelope already eats ~100 bytes before the payload.
- 1024. Rejected: bigger than needed for the diagnostic question; raises sensitive-data exposure and per-incident log volume if a noisy producer goes bad.
- Configurable via env var. Rejected: premature configurability for a constant we have no operational reason to vary per service.

### 3.4 Truncation strategy

**Decision:**

```go
preview := msg.Value
if len(preview) > previewMax {
    preview = preview[:previewMax]
}
previewStr := fmt.Sprintf("%q", preview)
```

`%q` produces a Go-syntax double-quoted string, escaping non-printable bytes safely. This means the preview is always a single line, JSON-loggable without further escaping, and copy-pasteable.

**Alternatives considered:**
- `string(msg.Value[:N])`. Rejected: corrupts logs on non-UTF8 bytes and may embed real newlines that confuse line-oriented log shippers.
- `base64.StdEncoding.EncodeToString`. Rejected: makes human inspection harder for the 99% case where the payload is JSON, which is the entire point of the preview.
- Adding an explicit `truncated bool` field. Rejected: derivable from `payload_size > previewMax`. Not worth a separate field.

### 3.5 Logger field plumbing

**Decision:** Use `l.WithFields(logrus.Fields{...}).WithError(err).Errorf(msg)`.

The handler already receives a `logrus.FieldLogger`, which is the interface that supports `WithFields` and `WithError`. The Atlas project's ECS formatter consumes these as structured fields rather than embedding them in the message text. No new logger dependency, no global state, no formatter changes.

**Alternatives considered:**
- Build a helper `logUnmarshalFailure(l, msg, err)` in the same package. Considered. Slight code-quality win but the call site is a single block; inlining keeps the diff and the test surface minimal. We can extract later if a second call site appears.
- Use `logrus.NewEntry` and craft a fully custom entry. Rejected: redundant with what `WithFields` already does.

### 3.6 Return semantics on unmarshal failure

**Decision:** Continue to return `(true, nil)`.

This is the explicit "commit-and-skip / at-most-once" decision the PRD locks in. Returning `(false, nil)` would terminate one-shot consumers prematurely on a bad message — wrong. Returning `(_, err)` would prevent the offset commit and create a poison-pill loop on persistent consumers — wrong, and the exact failure we are avoiding. The PRD says signature stays put; this design honors that.

**Future-state note (deferred, not in scope):** When we eventually widen `Handler[M]` to return an error, the right shape is likely two error channels — a *transport* error that signals "do not commit" and a *payload* error that signals "commit-and-skip-with-diagnostic." Today both are conflated into the boolean. We will know which way to split it after the Error logs give us a few weeks of real-world data.

### 3.7 Validator-rejection path

**Decision:** Unchanged. Line 51–54 stays untouched.

Validator returning `false` means the message is well-formed JSON but the consumer chose not to process it (tenant filter, type discriminator, etc.). That is a designed feature, not a fault. Logging it at Error level would create deafening noise on any topic with a non-trivial filter. The PRD explicitly carves this out and the design agrees.

### 3.8 `adapt[M]` factoring

**Decision:** Keep `adapt[M]` as a separate function (lines 62–69 stay).

It is used through `model.Map`, which is the functorial pattern the rest of the lib follows. Inlining the `json.Unmarshal` into `AdaptHandler` would save four lines but break consistency with `libs/atlas-model` style and lose a useful seam for future per-message-type adapters (e.g., protobuf, avro).

## 4. Test Design

### 4.1 Capturing logrus output

**Decision:** In each test, construct a fresh `logger := logrus.New()`, set `logger.SetOutput(&buf)` with `buf := &bytes.Buffer{}`, set `logger.SetFormatter(&logrus.JSONFormatter{})`, set `logger.SetLevel(logrus.DebugLevel)`, then pass `logger` (which satisfies `logrus.FieldLogger`) into the adapted handler.

Assertions read `buf.String()`, parse each line as JSON, and inspect:
- `level` field
- `msg` field
- structured field values (`topic`, `partition`, `offset`, `payload_size`, `payload_preview`, `message_type`)

**Why JSON formatter:** Makes field-by-field assertions straightforward without regex hacks. The production formatter is also structured, so this stays close to production behavior.

**Alternatives considered:**
- `logrus/hooks/test`. Rejected: introduces a hook abstraction the rest of the codebase does not use, and we get cleaner assertions reading parsed JSON than walking `Entry.Data`.
- Inspecting `logrus.StandardLogger()`. Rejected: global state, race-unsafe, contaminates other tests.

### 4.2 Test cases

Map directly to PRD §4.3. Sketch (final wording lives in the test file):

1. **Happy path** — a typed event marshalled via `json.Marshal`, fed in as `msg.Value`, with a `PersistentConfig` handler that increments a counter. Assert: counter == 1, buf contains no `level=error`, return value is `(true, nil)`.
2. **Malformed JSON** — `msg.Value = []byte("{not json")`. Handler counter must stay at 0. buf must contain exactly one `level=error` entry, and that entry's fields must include `topic`, `partition` (matching the input msg), `offset`, `payload_size = 9`, `payload_preview` containing `"{not json"`, `message_type` matching `fmt.Sprintf("%T", *new(M))`. Return value `(true, nil)`.
3. **Oversized payload** — `msg.Value = bytes.Repeat([]byte("A"), 10000)` (no closing brace, intentionally invalid). Assertions: `payload_size == 10000`, `payload_preview` length is bounded (post-`%q` quoting, the *unquoted* prefix is ≤ 256 bytes), preview content starts with `AAAA…`.
4. **Validator rejects** — valid JSON, `OneTimeConfig` with a validator returning `false`. Assert: handler counter stays 0, buf contains no `level=error`, return value `(true, nil)`.

Each test runs against its own `logrus.Logger` and a fresh buffer, so cases are independent and parallelizable.

### 4.3 Test plumbing

We need a concrete `M` for the generic. Define a tiny `type fakeEvent struct { ID int `json:"id"` }` inside the test file. No exported types are added.

`kafka.Message` is a plain struct; build it inline with literal field values. No mocking framework needed.

## 5. Risks and Mitigations

| Risk | Mitigation |
| --- | --- |
| Log-volume spike if a producer is actively broken. | Acceptable and intended — this is the signal we have been missing. We are not adding rate-limiting in v1; if a downstream incident shows we need it, that becomes a follow-up. |
| Sensitive bytes in preview. | 256-byte cap; `%q` quoting; PRD §8 explicitly accepts this risk for Atlas's threat model (game state, not credentials). |
| `*new(M)` panicking for some `M`. | It does not — `new(M)` always returns a valid `*M` even for interface or pointer `M`; `%T` against a nil pointer formats as `*pkg.Type`, which is exactly what we want. Test case 2 exercises this. |
| Adding a test that depends on log line ordering across goroutines. | Tests are synchronous: one call into `AdaptHandler`'s returned function per case. No goroutines. |
| Drift between this lib and consumers via `go.work`. | No signature change → no consumer rebuild required for correctness. Per CLAUDE.md, a downstream `docker build` is still required because lib bytes change; the plan phase will spell out which services need rebuilds. |

## 6. Out of Scope (Restated for Plan Phase)

The plan in phase 3 should **not** include:
- DLQ topic / producer / replay tooling.
- Metrics (Prometheus or OTel) for processing outcomes.
- Handler signature change to return `error`.
- Producer hardening (retry, jitter, circuit breaker).
- Schema-registry adoption.
- Validator-rejection logging changes.

Each of those is its own task. The current task is exactly the minimum diff to stop silent drops.

## 7. Diff Shape Estimate

- `libs/atlas-kafka/message/handler.go` — ~10 added lines (preview computation + WithFields/WithError call), 1 removed line (the Debugf). One new top-level `const previewMax = 256`.
- `libs/atlas-kafka/message/handler_test.go` — new file, ~150 lines covering four cases plus a small `fakeEvent` type and a JSON-line parser helper.
- No changes anywhere else in the repo. No changes to any service's `go.mod` / `Dockerfile` / k8s manifest.
