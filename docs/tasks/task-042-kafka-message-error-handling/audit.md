# Backend Audit — task-042-kafka-message-error-handling

- **Scope:** `libs/atlas-kafka/message/handler.go` (modified) and `libs/atlas-kafka/message/handler_test.go` (created).
- **Type:** Library-only change. Full DOM-* service checklist N/A; applied logging, error-handling, test-quality, and concurrency-relevant subset.
- **Date:** 2026-05-17
- **Build:** PASS (`go build ./...` clean from `libs/atlas-kafka`).
- **Vet:** PASS (`go vet ./...` clean).
- **Tests:** PASS (`go test -race -count=1 ./...` clean across all `atlas-kafka` packages, including the new `message` tests).
- **Overall:** PASS_WITH_NITS

## Build & Test Results

```
ok   github.com/Chronicle20/atlas/libs/atlas-kafka/consumer        4.928s
ok   github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup   1.608s
?    github.com/Chronicle20/atlas/libs/atlas-kafka/handler         [no test files]
ok   github.com/Chronicle20/atlas/libs/atlas-kafka/message         1.480s
ok   github.com/Chronicle20/atlas/libs/atlas-kafka/producer        1.314s
ok   github.com/Chronicle20/atlas/libs/atlas-kafka/retry           2.232s
?    github.com/Chronicle20/atlas/libs/atlas-kafka/topic           [no test files]
```

## Strengths

- **Actionable error log.** `handler.go:57-64` emits a single Error entry with `topic`, `partition`, `offset`, `payload_size`, `payload_preview`, `message_type`, and the underlying error via `WithError`. The message body explicitly states "offset will be committed and the message dropped." That is precisely the "3 AM page" content an operator needs: which topic/partition/offset to investigate, how big the payload was, what type was being decoded, and what the system is going to do next. No ambiguity.
- **Bounded preview, well-justified.** `const previewMax = 256` (`handler.go:14-17`) has a comment explaining both the size envelope rationale and the sensitive-data exposure rationale. The slice `preview = preview[:previewMax]` is a length check before slice; safe on nil/short payloads.
- **`%q` for binary safety.** `fmt.Sprintf("%q", preview)` (`handler.go:62`) renders non-UTF8 / control-byte payloads as Go-quoted escapes — protects log ingestion pipelines from raw binary, keeps the line single-token, and preserves enough fidelity to diagnose the bad bytes. The right call for this use case.
- **No concurrency regression.** Logrus's `WithFields` and `WithError` return new `*Entry` values rather than mutating the shared logger (`logrus.Logger.WithFields` → `entry.WithFields`), so the closure in `AdaptHandler` has no shared mutable state. The closure also touches no captured mutable variables outside the per-call `msg` and `tem` locals. Safe for concurrent `AdaptHandler` invocations on multiple partitions.
- **Behavior-focused tests.** All four tests (`handler_test.go:50, 120, 163, 218`) assert externally observable behavior: handler-called counts, persistent return value, presence/absence of error entries, and decoded JSON field values. No mock spying on internal state, no testing of implementation details.
- **Truncation boundary is exactly exercised.** `handler_test.go:163-216` uses a 10 KB body of `'A'` (no escape expansion under `%q`) and asserts `len(unquoted) == 256`. The `'A'` choice makes the assertion exact rather than approximate — this is the right way to test a boundary.
- **JSON line decoder is robust.** `decodeLogLines` (`handler_test.go:34-48`) splits on newlines, trims, skips empty lines, and `t.Fatalf`s on a JSON decode error. Will not silently swallow a malformed log line.
- **Out-of-scope items documented up-front.** `context.md:33-44` enumerates the seven deferred follow-ups (DLQ, metrics, `Handler[M]` returning `error`, producer hardening, schema registry, validator-rejection log changes, logging headers/keys/time). `context.md:56` makes "PR description calls out deferred follow-ups" a verification-matrix item, so the PR author is contractually reminded.

## Issues by Severity

### Blocking
None.

### Non-blocking / Nits

1. **`fmt.Sprintf("%T", *new(M))` quirk for pointer / interface type parameters.** At `handler.go:63`, `*new(M)` evaluates to the zero value of `M`. For value-typed events (the only kind used in practice — typed structs) this prints e.g. `message.fakeEvent`. For an interface-typed `M` or a pointer-typed `M`, this prints `<nil>` and the operator loses the type tag. No current caller instantiates `AdaptHandler` with a pointer or interface type, so this is theoretical. Optional fix: use `reflect.TypeOf((*M)(nil)).Elem().String()` which always returns the static type name. Leaving as-is is acceptable.
2. **`message_type` field is the only field that could collide with logrus reserved keys depending on formatter configuration.** Logrus reserves `msg`, `level`, `time`, and the field configured as `error` (default `error`). The chosen keys (`topic`, `partition`, `offset`, `payload_size`, `payload_preview`, `message_type`) are all safe. No action needed — just noting that the choice avoided the foot-guns.
3. **`logrus` is imported transitively by tests but the import shape is fine.** `handler_test.go:13` imports `github.com/sirupsen/logrus` directly. The test file is in `package message_test` (black-box), which is the right call for verifying public API behavior.
4. **`fmt` import** is present at `handler.go:6` — confirmed it was added (was absent before this diff). The `goimports`-style ordering (stdlib block then third-party block) is preserved.

## Targeted Evaluation Checklist (from review request)

| Concern | Result | Evidence |
| --- | --- | --- |
| Logging discipline (level + fields + message) | PASS | `handler.go:57-64`. Error level matches "operational regression that needs investigation," structured fields are all queryable, message text states the take-action and the consequence. |
| Bounded resource usage (truncation correctness) | PASS | `handler.go:53-56` length-checks before slicing; `previewMax = 256` documented at `handler.go:14-17`. Exact-boundary test at `handler_test.go:208-212`. |
| `%q` for binary / non-UTF8 payloads | PASS | `handler.go:62`. `%q` escapes invalid UTF-8 to `\xNN` per the `strconv.Quote` rules — safe for downstream log ingestion. |
| Tests verify behavior, not implementation | PASS | All four tests assert public observables (`handler_test.go:50, 120, 163, 218`). |
| JSON line decoder is robust | PASS | `handler_test.go:34-48` — trim, skip empty, fatal on decode error. |
| Truncation fixture sized to exact boundary | PASS | `handler_test.go:173` 10 KB of `'A'` bytes; `'A'` has no `%q` expansion → exact `len(unquoted) == 256` (line 210-212). |
| Concurrency safety (`AdaptHandler[M]` closure) | PASS | `logrus.Entry` is the receiver of `WithFields`/`WithError`; both return a fresh `*Entry`. No shared mutable state in the closure. |
| DOM-21 (no reinvented `atlas-constants` types) | PASS | `previewMax` is a local logging concern with no shared equivalent in `libs/atlas-constants/`. Not an item-id, classification, inventory type, or world/channel/character/map width. |
| Test helper pattern (no `*_testhelpers.go`) | PASS | Two small helpers (`newCapturingLogger`, `decodeLogLines`) live inside `handler_test.go` itself, package `message_test`. CLAUDE.md forbids only `*_testhelpers.go` files; in-file helpers are explicitly fine per `context.md:61`. |
| `fmt` import added correctly | PASS | `handler.go:6`. |
| PR-deferred work flagged | DEFERRED-TO-PR | The seven deferred items are documented at `context.md:33-44`. `context.md:56` makes the PR description's call-out an explicit verification gate. Audit cannot verify the PR body (it does not exist yet), but the plan/context already mandate it. Flag for the PR author. |

## Summary

### Blocking (must fix)
None.

### Non-Blocking (consider)
- Optional: replace `fmt.Sprintf("%T", *new(M))` with `reflect.TypeOf((*M)(nil)).Elem().String()` to keep the type tag stable if any future caller instantiates `AdaptHandler` with a pointer or interface type parameter. Current callers all use struct types so this is purely defensive.
- PR author: ensure the PR description explicitly lists the seven deferred follow-ups from `context.md:33-44` (per the plan's verification matrix at `context.md:56`).
