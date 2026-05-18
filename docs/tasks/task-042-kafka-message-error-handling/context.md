# Task 042 — Implementation Context

## What this task is

Replace the silent `Debug`-level log on Kafka unmarshal failure with an `Error`-level structured log so operators can see schema drift in real time. The `Handler[M]` signature, the `(true, nil)` commit-and-skip contract, the validator-rejection path, and the `adapt[M]` factoring all stay byte-identical — this is the smallest possible diff that removes the silent drop.

## Key files

- **`libs/atlas-kafka/message/handler.go`** — the single production file modified. Add `const previewMax = 256`, add `fmt` to imports, replace the `Debugf` block inside `AdaptHandler[M]` with a `WithFields(...).WithError(err).Errorf(...)` block. Everything else is unchanged.
- **`libs/atlas-kafka/message/handler_test.go`** — new file, ~4 tests + 2 small helpers. Black-box (`package message_test`); uses a fresh `logrus.Logger` per test with `logrus.JSONFormatter` and a `bytes.Buffer` so log assertions parse real JSON.
- **`libs/atlas-kafka/consumer/debug_test.go`** — read-only reference for the prevailing test style in this lib: standard `testing`, `t.Fatalf`/`t.Errorf` (no testify), inline struct literals for `kafka.Message`.

## Key decisions (from design.md)

| Decision | Choice | Why this matters during execution |
| --- | --- | --- |
| Log level | `Error` | Anything below (Warn/Debug) is invisible at prod log levels; anything above (Fatal/Panic) kills the service on bad data. |
| Field set | `topic`, `partition`, `offset`, `payload_size`, `payload_preview`, `message_type` | Tests assert each by name. Do not rename. |
| Preview budget | `const previewMax = 256` (package-level) | Untyped const, not a config knob. |
| Preview formatting | `fmt.Sprintf("%q", preview)` | Safe for non-UTF8 / embedded newlines; tests assume the unquoted prefix length equals 256 for raw-ASCII payloads. |
| `message_type` source | `fmt.Sprintf("%T", *new(M))` | Standard idiom for naming a Go generic parameter type without an instance. Works for pointer and interface `M`. |
| Return value | `(true, nil)` unchanged | Required: changing it creates a poison pill or kills one-shot consumers. |
| Validator-rejection path | Unchanged | Validators filtering on tenant/type are *designed* to reject; logging them at Error would be deafening. |
| Test capture mechanism | `logrus.New()` + `bytes.Buffer` + `JSONFormatter` | Avoids `logrus/hooks/test` and any global state. |
| Test package | `message_test` (black-box) | We only need exported symbols (`PersistentConfig`, `OneTimeConfig`, `AdaptHandler`). |

## Dependencies / lib + service impact

- `libs/atlas-kafka/go.mod` already declares `github.com/sirupsen/logrus` and `github.com/segmentio/kafka-go` as direct deps. No `go.mod` change needed.
- **No service `go.mod` changes**, no `Dockerfile` changes, no k8s changes. The four-place lib list in every consumer's Dockerfile already includes `libs/atlas-kafka`. Workspace `go build` is sufficient verification; per-service `docker build` is **not** required for this task.
- Consumers pick up the new behavior automatically the next time they're rebuilt; no coordinated deploy needed because the on-wire protocol is unchanged.

## Out of scope (do not let scope creep in)

The design document §6 enumerates these; the plan must not reach for any of them:
1. DLQ topic / producer / replay tooling.
2. Prometheus or OpenTelemetry metrics for processing outcomes.
3. Widening `Handler[M]` to return `error`.
4. Producer-side hardening (retry, jitter, circuit breaker).
5. Schema-registry adoption / schema headers.
6. Changes to the validator-rejection log path (line 51–54 in current `handler.go`).
7. Logging `msg.Key`, `msg.Headers`, or `msg.Time` in the error entry.

Each of those is a future task. If a step in the plan starts to look like one of these, stop and re-check the design doc.

## Verification matrix

| PRD §10 item | Verified in |
| --- | --- |
| Error-level structured log on unmarshal failure | Task 1 test + Task 2 implementation |
| `(true, nil)` preserved on unmarshal failure | Task 1 assertion |
| No exported signature changes | Visual diff after Task 2 (the file replacement intentionally keeps every signature) |
| Four PRD test cases pass under `go test -race` | Tasks 1, 3, 4, 5; combined in Task 6 |
| `go vet ./...` clean in `libs/atlas-kafka` | Task 6 Step 2 |
| `go build ./...` clean lib + all consumers | Task 6 Step 3 (lib) + Task 7 Step 2 (workspace) |
| PR description calls out deferred follow-ups | Task 7 Step 4 |

## Project conventions worth re-reading before executing

- `CLAUDE.md` "Build & Verification" — confirms that for a lib-only change with no `go.mod`/`Dockerfile` edits, the workspace `go build ./...` is the right level of verification (per-service `docker build` is required only when `go.mod` or `Dockerfile` is touched).
- `CLAUDE.md` "Test Helper Pattern" — use builder-style setup, not `*_testhelpers.go` files. The two small helpers in the test file (`newCapturingLogger`, `decodeLogLines`) are local to the test file and fine.
- `CLAUDE.md` "Code Review Before PR" — invoke `superpowers:requesting-code-review` before opening the PR.

## Commit style

The repo follows Conventional Commits with a scope (see recent commits: `fix(pr-overlay): …`, `fix(atlas-wz-extractor): …`, `chore(images): …`). The plan uses `test(atlas-kafka): …` and `feat(atlas-kafka): …` for the new commits.
