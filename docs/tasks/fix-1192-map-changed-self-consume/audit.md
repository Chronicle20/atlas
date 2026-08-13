# Backend Audit — atlas-maps (fix-1192-map-changed-self-consume)

- **Service Path:** services/atlas-maps
- **Scope:** Single commit `4ddfcb0bb` on branch `fix-1192-map-changed-self-consume` (5 files: `character/warp/processor.go`, `character/warp/processor_test.go`, `kafka/consumer/character/consumer.go`, `docs/domain.md`, `docs/kafka.md`)
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-08-09
- **Build:** PASS
- **Tests:** all passed (0 failed) — `go test ./... -count=1` clean across every package
- **Overall:** PASS

## Build & Test Results

```
$ go build ./...          # exit 0, no output
$ go test ./... -count=1
ok  atlas-maps/character
ok  atlas-maps/character/location
ok  atlas-maps/character/warp
ok  atlas-maps/data/map/info
ok  atlas-maps/data/map/monster
ok  atlas-maps/kafka/consumer/character
ok  atlas-maps/kafka/consumer/data
ok  atlas-maps/kafka/consumer/mist
ok  atlas-maps/kafka/consumer/session
ok  atlas-maps/kafka/message/character
ok  atlas-maps/kafka/message/map
ok  atlas-maps/map
ok  atlas-maps/map/character
ok  atlas-maps/map/monster
ok  atlas-maps/map/timer
ok  atlas-maps/mist
ok  atlas-maps/monster
ok  atlas-maps/reactor
ok  atlas-maps/rest
ok  atlas-maps/tasks
ok  atlas-maps/visit
(all other packages: [no test files])
```

Also ran, scoped to the two changed Go packages:
- `gofmt -l character/warp/processor.go character/warp/processor_test.go kafka/consumer/character/consumer.go` → no output (clean).
- `go vet ./character/warp/... ./kafka/consumer/character/...` → clean.
- `tools/goroutine-guard.sh` (repo root) → no new violations attributable to this diff; no bare `go` statement appears in either changed file (`grep -n '^\s*go (func|[A-Za-z_])' character/warp/processor.go kafka/consumer/character/consumer.go` → 0 matches).

## Package Classification (Phase 2)

| Package | Classification | Rationale |
|---|---|---|
| `character/warp` | Support package (no `model.go`, no `resource.go`) | Holds only `processor.go` + `processor_test.go` — a pure coordination processor over `location`, `_map`, `timer`, `info`. |
| `kafka/consumer/character` | Support package (Kafka consumer registration, no `model.go`/`resource.go`) | Standard `InitConsumers`/`InitHandlers` + per-event handler funcs in `consumer.go`, pre-existing layout, unmodified by this diff except the deletion. |

Neither touched package has a `model.go`, so the DOM-* checklist does not apply as a full domain audit; the File Responsibilities Checklist (FILE-*) still applies to both per the audit charter, and is run below.

## File Responsibilities Checklist Results

### `character/warp` (files: `processor.go`, `processor_test.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | `Processor` interface + `ProcessorImpl` methods in `processor.go` | PASS | `character/warp/processor.go:34` (`type Processor interface`), `:50` (`type ProcessorImpl struct`), `:60` (`func NewProcessor(`), `:81` (`func (p *ProcessorImpl) ChangeMap(`), `:116` (`func (p *ProcessorImpl) applyMapTimer(`) — all in the single `processor.go`, no split needed at this size. |
| FILE-06 | No package-named catch-all file | PASS | Package contains only `processor.go` (production) + `processor_test.go` (test); no `warp.go` bundling multiple responsibilities. |
| N/A | `rest.go` / `requests.go` / `entity.go` / `builder.go` / `administrator.go` / `provider.go` | N/A | Package has no REST model, no cross-service HTTP calls, no DB entity, no builder, no writes, no queries of its own — it purely composes `location.Processor`, `mapTransitioner` (`_map.Processor`), `timer.Processor`, `info.Processor`. Nothing to misplace. |

### `kafka/consumer/character` (file: `consumer.go`, diff-only)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Handler funcs not masquerading as a Processor | PASS | `handleStatusEventMapChangedFunc` deleted outright (`kafka/consumer/character/consumer.go` diff, removed lines formerly ~138–171); remaining handler funcs (`handleStatusEventCreatedFunc` L82, `handleStatusEventLoginFunc` L94, `handleStatusEventLogoutFunc` L109, `handleStatusEventChannelChangedFunc` L141, `handleStatusEventDeletedFunc` L160) are pre-existing, untouched pattern — Kafka consumer registration file, not a domain `processor.go`. |
| FILE-06 | No new catch-all introduced | PASS | Diff only removes a handler function and its registration call, and adds an explanatory comment (`consumer.go:52-57`); no responsibility was added or relocated into this file. |

## Domain Checklist — DI / Dependency Injection (focus area 1)

| Check | Status | Evidence |
|---|---|---|
| `warp.NewProcessor` signature preserved | PASS | `character/warp/processor.go:60` — `func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor`, unchanged signature from before the diff (only the body grew two fields). |
| New deps constructed via their own `NewProcessor(l, ctx, ...)` | PASS | `character/warp/processor.go:68` `tp: timer.NewProcessor(l, ctx, pp)` matches `timer.Processor`'s own constructor at `map/timer/processor.go:37` (`func NewProcessor(l logrus.FieldLogger, ctx context.Context, p producer.Provider) Processor`); `character/warp/processor.go:69` `ip: info.NewProcessor(l, ctx)` matches `data/map/info/processor.go:24` (`func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor`). Both constructed the same way `_map.NewProcessor(l, ctx, pp, db)` already was on the preceding line (`processor.go:67`) — consistent composition style. |
| Test seam kept in sync | PASS | `newProcessorWithDeps` (`character/warp/processor.go:77`) extended with `tp timer.Processor, ip info.Processor` params; every call site in `processor_test.go` updated (`processor_test.go:131,179,210`). No `*_testhelpers.go` file created — the seam lives inline in `processor.go` per the doc comment at `processor.go:75-76` ("It is not exported and not a `*_testhelpers.go` file"), matching CLAUDE.md's "Test Helper Pattern" rule. |
| `FieldLogger` vs `*logrus.Logger` (DOM-06 style) | PASS | `character/warp/processor.go:60` takes `l logrus.FieldLogger`; unchanged, not downgraded to a concrete `*logrus.Logger`. |

## Error-Handling Parity (focus area 2)

| Check | Status | Evidence |
|---|---|---|
| `applyMapTimer` failures logged, not silently swallowed, not aborting `ChangeMap` | PASS | `character/warp/processor.go:116-129`: `CancelIfTracked` return value intentionally ignored (bool, matches existing idiom `_ = tp.ForceReturnIfTracked(...)` at `kafka/consumer/character/consumer.go:155`); `GetById` error path logs `p.l.WithError(err).Debugf(...)` at `:121` and returns (no timer registered, transition already completed); `Register` error path logs `p.l.WithError(err).Warnf(...)` at `:128`. `ChangeMap` itself always returns `nil` after `applyMapTimer` (`processor.go:102-104`) — parity with the pre-existing emit-failure (`processor.go:91-96`, `Errorf`) and transition-failure (`processor.go:98-100`, `Warnf`) handling immediately above it in the same function: none of the three post-persistence side effects can fail the call. |
| Log-level parity with the deleted consumer code | PASS | The deleted `handleStatusEventMapChangedFunc` used `Debugf` for the map-info-lookup miss and `Warnf` for the `Register` failure (pre-image at `kafka/consumer/character/consumer.go` diff, removed lines). `applyMapTimer` preserves the identical levels verbatim — this is a lift-and-shift, not a re-decision of severity. |
| Doc comment accurately describes the new failure contract | PASS | `character/warp/processor.go:41` ("...emit/transition/timer failures are logged and the call still succeeds") updated in lockstep with the code; `docs/domain.md` (Warp Processor section) says "emit and transition failures are logged and the call still succeeds" without explicitly naming timer failures as a third item — a minor prose omission, not a code defect (non-blocking, see Summary). |

## Test-Helper Convention (focus area 4)

| Check | Status | Evidence |
|---|---|---|
| No new `*_testhelpers.go` file | PASS | `git show --stat HEAD` lists only `processor_test.go` as the touched test file; no new file matching `*_testhelpers.go` was added. |
| New test doubles use the Builder pattern where a domain model needs constructing | PASS | `processor_test.go:172-178` and `:207-209` build `info.Model` fixtures via `info.NewBuilder().SetId(...).SetTimeLimit(...).SetForcedReturnMapId(...).Build()`, using `info`'s own builder (`data/map/info/model.go:31-36`) rather than a struct literal or an ad hoc constructor. |
| Fakes (`recordingTimer`, `stubMapInfo`, `capturingProducer`, `noopTransitioner`) are plain interface-satisfying structs, not helper-file constructors | PASS | All four types are declared directly in `processor_test.go` (`:28-91`, `capturingProducer`/`noopTransitioner` pre-existing, `recordingTimer`/`stubMapInfo` new in this diff) implementing `timer.Processor` / `info.Processor` / `mapTransitioner` / `kafkaproducer.Provider` respectively — the standard fake-dependency pattern for this package, not a violation of the Builder-for-test-setup rule (that rule targets constructing domain **models**, which is done via `info.NewBuilder()` here). |

## Kafka Producer Stub Check (DOM-24-style)

| Check | Status | Evidence |
|---|---|---|
| Tests exercising `message.Emit` are stubbed | PASS | `TestChangeMap_PersistsAndEmitsMapChanged` (`processor_test.go:117`) and the two new tests (`:164`, `:199`) all pass `cp.Provider()` — a fully in-memory fake (`capturingProducer`, `processor_test.go:28-47`) that never touches a real Kafka writer/broker. No real `producer.ProviderImpl` or `producertest.InstallNoop()` is needed here because the fake never reaches `libs/atlas-kafka/producer`'s retry/backoff path — this is the "per-test injection of a no-op Provider via a builder-style seam" shape the guideline endorses (DOM-24 option (c)), not the shared `producertest` package, but it satisfies the same goal (no live producer touched) and pre-dates this diff. |

## Contract-Deletion Risk Check (focus area 5)

| Check | Status | Evidence |
|---|---|---|
| Was atlas-maps the only in-repo registrant of the deleted `handleStatusEventMapChangedFunc`? | PASS (within atlas-maps) | `grep -rn "MapChangedStatusProvider"` inside `services/atlas-maps/atlas.com/maps` returns exactly one call site: `character/warp/processor.go:92-93`. `grep -rn "EnvEventTopicCharacterStatus"` inside the same module shows the only consumer-side registration for that topic is `kafka/consumer/character/consumer.go:32,42`, and `InitHandlers` (`consumer.go:39-74`) after this diff registers exactly `CREATED, LOGIN, LOGOUT, CHANNEL_CHANGED, DELETED` for that topic — `MAP_CHANGED` is confirmed absent from atlas-maps' own dispatch table. |
| Does any OTHER service silently lose a real downstream contract? | **UNVERIFIED — out of this audit's scope** | This audit was scoped to `services/atlas-maps` only. Whether atlas-transports or the "~18 other services" the commit message asserts actually consume `EVENT_TOPIC_CHARACTER_STATUS` / `MAP_CHANGED` cannot be confirmed by reading atlas-maps' own source. The commit message's claim is plausible (MAP_CHANGED is a broadly-relevant status event) but is an **unverified assertion**, not evidence, per this project's "Verification Over Memory" rule. This is not scored as a blocking finding because the diff under audit does not touch or claim ownership of any consumer outside atlas-maps, and the instructions for this audit explicitly permit deferring cross-service grepping as out of scope — but it should not be read as "confirmed safe for all consumers," only as "confirmed atlas-maps no longer double-fires its own registries." A follow-up cross-service grep for `EVENT_TOPIC_CHARACTER_STATUS` consumers (especially in atlas-transports, per the commit message's example) is recommended before merge if that hasn't already been done outside this audit. |

## Summary

### Blocking (must fix)

None.

### Non-Blocking (should fix)

- `docs/domain.md` (Warp Processor section, "ChangeMap: ... Returns an error only when the durable Set fails; emit and transition failures are logged and the call still succeeds.") does not explicitly name timer-application failures as a third logged-and-swallowed side effect, even though `applyMapTimer` (`character/warp/processor.go:116-129`) is exactly that. The code's own doc comment at `processor.go:41` already says "emit/transition/timer failures" — `docs/domain.md` should be updated to match for consistency.
- Cross-service consumption of `EVENT_TOPIC_CHARACTER_STATUS` / `MAP_CHANGED` outside atlas-maps was not verified as part of this audit (scoped to `services/atlas-maps` only). Recommend a repo-wide grep for `EVENT_TOPIC_CHARACTER_STATUS` consumer registrations before treating the commit message's "~18 other services consume it" claim as confirmed.
