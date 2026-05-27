# Kafka Message Error Handling — Surface Silent Drops — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-05-17
---

## 1. Overview

`libs/atlas-kafka/message/handler.go:42` adapts a typed `Config[M]` into the generic consumer handler. When the raw Kafka message body fails JSON unmarshaling (line 45–48), the current code logs at **Debug** level and returns `(true, nil)` — the consumer commits the offset and the message disappears silently. Because every Atlas Go service consumes Kafka through this single adapter, every malformed event in production is dropped with no operator-visible signal.

Default log levels in Atlas services are typically Info or above, so the Debug log is effectively invisible. Combined with the lack of metrics, this means schema mismatches between producers and consumers (e.g., after a partial deploy or an envelope drift like the one called out in the 2026-05-17 transport audit) cause silent message loss until something downstream visibly breaks.

This task closes the silent-drop gap with the minimum viable change: upgrade the log level to **Error**, enrich the log payload with the topic / partition / offset / payload-size / payload-preview, and keep `(true, nil)` so we do not create a poison-pill retry loop. The `Handler[M]` signature is intentionally **not** changed in this task — that is a larger refactor we will revisit after we have observability data on real-world drop frequency.

## 2. Goals

Primary goals:
- Make every malformed Kafka message that hits the consumer adapter loudly visible in service logs.
- Preserve current at-most-once behavior on unmarshal failure (commit the offset, do not poison-pill the partition).
- Add unit tests in `libs/atlas-kafka/message` that pin the new behavior, including log assertions.

Non-goals:
- Changing the `Handler[M]` signature to return an error. (Deferred — would require touching ~55 services and is better done with metrics in place to justify the cost.)
- Adding a DLQ topic, DLQ producer, replay tooling, or per-topic DLQ configuration. (Deferred — flagged for a future task.)
- Adding Prometheus / OpenTelemetry metrics for processing outcomes. (Deferred — log-only this round.)
- Hardening the Kafka producer (retry config, jitter, circuit breaker) — flagged in the 2026-05-17 audit but out of scope here.
- Validator-rejection logging changes (line 51–54): validators reject *valid* messages by design; not a silent-drop bug.

## 3. User Stories

- As a service operator, I want every malformed Kafka message to produce an Error-level log line with topic / partition / offset, so I can detect and diagnose schema mismatches without waiting for a downstream symptom.
- As a backend engineer rolling out an envelope change, I want any cross-version mismatch to be visible in logs immediately, so I can roll back before user-facing damage accumulates.
- As an on-call responder, I want the log payload to contain enough context (a payload preview) to identify the offending producer without having to attach a Kafka console consumer to the topic.

## 4. Functional Requirements

### 4.1 Log behavior change

In `libs/atlas-kafka/message/handler.go` inside `AdaptHandler[M]`:
- Replace `l.WithError(err).Debugf("Unable to unmarshal message.")` with an Error-level log that includes structured fields:
  - `topic` (from `msg.Topic`)
  - `partition` (from `msg.Partition`)
  - `offset` (from `msg.Offset`)
  - `payload_size` (`len(msg.Value)`)
  - `payload_preview` — the first N bytes of `msg.Value` rendered as a string, where N is a constant (suggested: 256). Must be safe for non-UTF8 bytes (e.g., use `string(msg.Value[:min(N, len(msg.Value))])` after the same bounds check `kafka-go` already enforces, or `fmt.Sprintf("%q", ...)` for safer quoting).
  - `message_type` — the Go type name of `M` for the consumer (e.g., via `fmt.Sprintf("%T", *new(M))`), so the log line says which event shape failed to deserialize.
- Continue to return `(true, nil)` so the consumer commits the offset and proceeds.

### 4.2 Behavior preservation

- Validator-rejection path (line 51–54) is unchanged.
- Persistent / one-time semantics (line 57) are unchanged.
- Successful handler invocation path is unchanged.
- Existing `Validator[M]`, `Handler[M]`, `Config[M]`, `PersistentConfig`, `OneTimeConfig`, `AdaptHandler` exported surface is unchanged. No consumer code in any service needs to change.

### 4.3 Tests

Add unit tests in `libs/atlas-kafka/message/handler_test.go`:
- Test 1: valid message → handler invoked once, no error log emitted, return value matches `Config.persistent`.
- Test 2: invalid JSON message → handler NOT invoked, Error-level log emitted with all required structured fields populated, return value is `(true, nil)`.
- Test 3: oversized payload (e.g., 10 KB) → log emitted with `payload_preview` truncated to the configured N bytes and `payload_size` matching the full length.
- Test 4: validator returns false → handler NOT invoked, no Error log emitted, return value is `(true, nil)`.

Use a `logrus.New()` instance with a captured `Hook` or buffered `Output` to assert log entries. Do not introduce a new logging dependency.

## 5. API Surface

No public Go API changes. The exported symbols in `libs/atlas-kafka/message` retain their current signatures and behavior contracts. The only observable diff is log volume on the Error channel when malformed messages occur.

## 6. Data Model

None.

## 7. Service Impact

- **`libs/atlas-kafka`** — single file change in `message/handler.go` plus a new test file.
- **All Atlas Go services** — no code change. The library is consumed via `go.work` replace; the next build of each service picks up the new behavior automatically. Log volume on the Error channel may increase if malformed messages are already happening unnoticed; this is the intended outcome.
- **Dockerfile / k8s** — no changes. The four-place lib list documented in the project README is already correct for `libs/atlas-kafka` across all consumers.

## 8. Non-Functional Requirements

### Observability
- The new log line must be parseable by the standard ECS JSON formatter Atlas uses. Use `logrus.Fields` for structured fields, not interpolated message strings.

### Performance
- The unmarshal-error path is already cold (only fires on bad messages). The added formatting cost is negligible.

### Security
- `payload_preview` is the first N bytes of the message body. Atlas Kafka messages contain game state, not raw credentials, but operators should treat the preview as potentially sensitive. The 256-byte cap limits accidental exposure. Do not log the full payload.

### Backward compatibility
- No producer or consumer needs to change. Existing log scrapers that filtered on the old Debug message string will no longer match — this is acceptable since the message was effectively invisible at production log levels anyway.

### Rollout
- Single bundled PR for `libs/atlas-kafka` only. After merge, every service rebuilt with the new lib version picks up the behavior. No coordinated deploy required because the on-wire protocol is unchanged.

## 9. Open Questions

- **Payload preview byte budget** — 256 bytes is a suggested default; design phase may revisit if envelope sizes argue for more or less.
- **Future DLQ work** — should be a separate task that builds on this one. Once we have Error logs and a baseline frequency, we can size DLQ retention.
- **Future handler-signature widening** — should be a separate task, gated on metrics being available so we can quantify the cost/benefit.

## 10. Acceptance Criteria

- [ ] `libs/atlas-kafka/message/handler.go` logs unmarshal failures at Error level with the structured fields listed in §4.1.
- [ ] `libs/atlas-kafka/message/handler.go` still returns `(true, nil)` on unmarshal failure (offset is committed, no poison pill).
- [ ] No exported symbols in `libs/atlas-kafka/message` have changed signatures.
- [ ] New unit tests cover the four cases listed in §4.3 and pass under `go test -race ./...` in `libs/atlas-kafka`.
- [ ] `go vet ./...` passes in `libs/atlas-kafka`.
- [ ] `go build ./...` passes in `libs/atlas-kafka` and in every service that consumes it (sanity check; no service code is changed).
- [ ] PR description explicitly notes the deferred follow-ups (handler error return, metrics, DLQ) so they remain visible as future work.
