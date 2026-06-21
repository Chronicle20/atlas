# Backend Audit — task-103 guild dispatcher-family migration

- **Scope:** Changed Go packages in commit range `72cb42c6c..HEAD`
  - `libs/atlas-packet/guild/...` (clientbound, body funcs, serverbound, tests)
  - `services/atlas-channel/atlas.com/channel/kafka/consumer/{guild,invite}`, `socket/writer/guild_bbs.go`
  - `tools/packet-audit/cmd/run.go`
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-06-18
- **Build:** PASS (atlas-packet, atlas-channel, tools/packet-audit)
- **Vet:** PASS
- **Tests:** PASS (`go test -race ./guild/...` ok; channel writer ok; consumers no test files)
- **Overall:** PASS

## Build & Test Results

```
libs/atlas-packet:  go build ./guild/...  -> clean
                    go vet  ./guild/...   -> clean
                    go test -race ./guild/... -count=1
                      guild/clientbound   ok (1.049s)
                      guild/serverbound   ok (1.033s)
atlas-channel:      go build ./...        -> clean
                    go vet (guild,invite,writer) -> clean
                    go test (guild,invite,writer) -count=1
                      socket/writer       ok
                      consumer/guild      [no test files]
                      consumer/invite     [no test files]
tools/packet-audit: go build ./...        -> clean
```

Objective gate passes. Proceeded to checklist.

## Applicability Note

This change set is packet-library + channel-consumer code, not a CRUD domain
service. There is no `model.go`/`entity.go`/`rest.go`/`resource.go`/`administrator.go`
domain package introduced or modified, so DOM-01..05, DOM-08..09, DOM-10..19,
DOM-22..24, SUB-*, and SEC-* are N/A by structure. The checks that *do* apply
(immutability, logger type, functional composition, error handling, DOM-21
constant reuse, no TODO/stub, test quality) are evaluated against the
dispatcher-family focus areas.

## Checklist Results

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Encode/Decode accept `logrus.FieldLogger` (not `*logrus.Logger`) | PASS | Every Encode/Decode in `operation.go` (e.g. L37, L86, L462), `bbs.go` (L71, L188, L243) takes `logrus.FieldLogger`. Consumers `consumer.go` handlers take `logrus.FieldLogger` (L123, L178). |
| DOM-07 | Handlers/consumers pass `l` through, no `StandardLogger()` | PASS | `consumer/guild/consumer.go` threads `l` from `message.Handler` signature; no `logrus.StandardLogger()` anywhere in changed files. |
| DOM-21 | No reinvention of atlas-constants shared types | PASS | Changed files declare only wire-shape packet structs (`mode byte`, `guildId uint32`, …) and guild-protocol operation-key string consts (`operation_body.go:18-54`), which are guild-specific and correctly live in the guild library. No world/channel/map/item/inventory type redeclared. |
| Immutability | Discrete structs are immutable (private fields + ctor) | PASS | All 36 clientbound structs in `operation.go` have lowercase fields and a `New*` constructor (e.g. L21-30 `RequestAgreement`, L81-83 `RequestName`, L1075-1083 `SetSkillResponse`). `bbs.go` L51-64, L168-181, L236-240. No exported mutable fields on the dispatcher structs. |
| Mode-via-config | Mode byte resolved from config, never literal | PASS | Every GuildOperation body func uses `atlas_packet.WithResolvedCode("operations", KEY, func(mode byte)…)` — `operation_body.go:58-276`. Constructors receive the resolved `mode` as first arg; no struct-literal mode in `operation.go`. |
| BBS literal-mode exception | BBS fixed-const mode is the documented, justified exception | PASS (noted) | `bbs_body.go:10-21` and `bbs.go:14-24` document version-stable modes 6/7/8 with no tenant `operations` table; passed as package consts `GuildBBSMode*`, not config-resolved. Intentional and documented — not a violation. |
| Encode/Decode symmetry | Round-trip correctness | PASS | Decode mirrors Encode field-for-field in every struct (spot-verified `EmblemChange` L462-484, `Invite` trailing-int gate L758-796, `ShowTitles` count-loop L976-1006, `SetSkillResponse` conditional message L1088-1107). `pt.RoundTrip` asserts zero unconsumed bytes across all `pt.Variants`. |
| Error-handling (focus) | Unmapped error codes logged + dropped, never sent as wrong mode | PASS | `consumer/guild/consumer.go:144-160` `guildErrorBodies` maps each code to exactly ONE fixed-key body func. `announceGuildError` L166-170: on miss, `l.WithField("error_code", errCode).Warn("unmapped guild error code; dropping")` and returns a no-op `func(_ session.Model) error { return nil }`. The dynamic `GuildErrorBody(errCode)` / `GuildErrorBody2(key, target)` footgun is fully removed (grep for a key-taking body func returns empty). |
| Call-site migration | Old footgun call sites migrated to fixed-key body funcs | PASS | `invite/consumer.go:181` now calls `guildpkt.GuildInviteDeniedBody(targetName)` (was `GuildErrorBody2(...)`). `socket/writer/guild_bbs.go:19,58,75` now call `guildbody.GuildBBS*Body(...)` (was `guildpkt.NewBBSThread*(...).Encode`). |
| Functional composition | Curried Operator/Announce pattern preserved | PASS | All `announce*` helpers in `consumer/guild/consumer.go` return curried `model.Operator[session.Model]` (e.g. L194-202, L348-356) consistent with `patterns-functional.md`. |
| Test quality | Real byte fixtures, Builder/ctor inputs, table-driven, no test-only helpers | PASS | `operation_test.go`: golden-byte asserts — mode-only `want := []byte{mode}` (L257-264), target-bearing `[]byte{mode,0x03,0x00,'B','o','b'}` (L313), per-version v83/v95 mode shift asserted (L304-305, L333-342); plus per-version round-trips over `pt.Variants`. No `*_testhelpers.go` (find returns none). No mocks (`grep` returns no mock usages). Uses ctors `New*`, not test-only constructors. |
| No TODO/stub/501 | No new stubs landed | PASS | No `TODO`/`FIXME`/`panic`/`501` introduced in changed files. (Pre-existing `// TODO` at `invite/consumer.go:154` predates this range — commit `2c43c0e9e`, not in `72cb42c6c..HEAD`, not in the task-103 diff.) |

## Investigated and Cleared (not violations)

- **`info.go:70 w.WriteByte(0x1A)` literal** — `info.go` has **no diff** in this
  range (`git diff 72cb42c6c..HEAD -- …/info.go` empty). It is the separate
  GUILDDATA::Decode path (not an OnGuildResult dispatcher arm), explicitly
  documented as out of dispatcher scope in `operation_body.go:278-285`. Untouched,
  not a regression.
- **`invite/consumer.go:154 // TODO`** — introduced by an unrelated prior commit
  (`2c43c0e9e`, buddy invitation), outside this range. Task-103 only changed
  `consumer.go:181`. Not landed by this task.

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- None introduced by this task. (Pre-existing `// TODO` at
  `invite/consumer.go:154` is outside scope; flagging for awareness only.)

### Verdict
PASS. The discrete-per-mode migration is clean: immutable structs with
constructor-injected (config-resolved) mode bytes, the AP-4 caller-picks-the-mode
footgun (`GuildErrorBody`/`GuildErrorBody2`) is fully eliminated with a
log-and-drop guard for unmapped codes, the BBS fixed-const exception is
documented and justified, and tests are real per-version golden-byte fixtures
with no mocks or test-only constructors. Build/vet/race-test gate is green on all
three modules.
