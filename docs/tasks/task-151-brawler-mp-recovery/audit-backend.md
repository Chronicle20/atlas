# Backend Audit — task-151-brawler-mp-recovery (Go diff)

- **Service Path:** `services/atlas-channel/atlas.com/channel`
- **Scope:** Go files changed on branch `task-151-brawler-mp-recovery` (merge-base `cdfb71aa3`..`c72d9c18f`)
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-07-27
- **Build:** PASS
- **Tests:** all packages `ok` (module-wide `go test ./... -count=1`), no failures
- **Overall:** PASS

## Scope

| File | Nature |
|---|---|
| `data/skill/effect/model.go` | +6 lines: `Y() int16` getter added to pre-existing `Model` |
| `data/skill/effect/model_test.go` | +9 lines: `TestModelY` |
| `skill/handler/mprecovery/formula.go` | new file: pure `Amounts(maxHp, x, y) (int16, int16)` |
| `skill/handler/mprecovery/formula_test.go` | new file: table-driven test for `Amounts` |
| `skill/handler/mprecovery/mprecovery.go` | new file: `Apply` handler registered for skill 5101005, package-var seams (`loadCaster`/`changeHP`/`changeMP`) over `character.Processor` |
| `skill/handler/mprecovery/mprecovery_test.go` | new file: behavioral tests for `Apply` |
| `skill/handler/registrations/registrations.go` | +1 line: blank-import wiring `mprecovery` into `init()` |

## Build & Test Results

```
$ cd services/atlas-channel/atlas.com/channel && go build ./...
(clean, no output)

$ go test ./... -count=1
ok  	atlas-channel/skill/handler/mprecovery	0.011s
ok  	atlas-channel/data/skill/effect	...   (aggregated — see full run; no FAIL lines anywhere in the module)
... all other packages ok / [no test files]
```
`go vet ./skill/handler/... ./data/skill/effect/...` — clean, no output.

## Package Classification (Phase 2)

- `skill/handler/mprecovery`: **no `model.go`, no `resource.go`** → not a Domain package, not a Sub-Domain (action-event/resource.go) package. It is a **support package** implementing the established "per-skill handler" pattern (`Handler` type defined in `skill/handler/registry.go`), structurally identical to sibling packages `mysticdoor`, `heal`, `healdispel`, `hide`, `resurrection` (all single-file, package-var-seam handlers registered via `init()` and dispatched from `skill/handler/common.go:152`). This is a distinct, pre-existing architectural lane from the REST/DDD domain-package pattern that `file-responsibilities.md` and the DOM-* checklist govern — it declares none of the governed symbols (`Processor`, `RestModel`, entity, builder, administrator, provider, state enum), so there is nothing for those checks to misplace.
- `data/skill/effect`: has `model.go` + `rest.go` → classified Domain package by the letter of Phase 2, but the change here is a single 6-line accessor mirroring the pre-existing `X()` getter (`model.go:150-153`) on an already-established, REST/WZ-sourced (not DB-persisted) model. The package's structural gaps against the full DOM checklist (no `builder.go`, no `entity.go`, no `administrator.go`/`provider.go` — it is not GORM-backed) predate this branch by a wide margin and are not introduced or touched by this diff; re-auditing the whole package's pre-existing shape is out of scope for a diff-scoped review of a one-getter addition. The getter itself is evaluated below.

## File Responsibilities Checklist — `skill/handler/mprecovery`

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor in `processor.go` | N/A | Package declares no `Processor`/`ProcessorImpl` type. `mprecovery.go` only *calls* `character.NewProcessor(l, ctx)` (an existing, separately-governed package) — `mprecovery.go:24,33,38`. |
| FILE-02 | RestModel/Transform in `rest.go` | N/A | Package declares no `RestModel`. |
| FILE-03 | Cross-service request funcs in `requests.go` | N/A | Package makes no `requests.GetRequest[T]`/`PostRequest[T]` calls; all calls go through `character.Processor` methods. |
| FILE-04 | Entity in `entity.go` | N/A | No entity/GORM code in package. |
| FILE-05 | Builder/Model/administrator/provider/state placement | N/A | None of these symbols exist in the package. |
| FILE-06 | No package-named catch-all bundling ≥2 responsibilities | PASS | `mprecovery.go` (99 lines) holds only: `init()` registration (`mprecovery.go:18-20`), three package-var seams delegating to `character.Processor` (`mprecovery.go:23-39`), and the `Apply` handler closure (`mprecovery.go:49-99`). Zero of the FILE-01..05 tracked responsibilities are present, so none are bundled. Same one-file shape as sibling handler packages `mysticdoor`, `heal`, `hide`, `healdispel`, `resurrection` — but graded here on its own contents, not on that precedent: it independently carries none of the governed symbols. |

## DOM-* / SUB-* Applicability

`skill/handler/mprecovery` has no `model.go` (not Domain) and no `resource.go` (not Sub-Domain per the action-event/REST-resource definition), so the DOM-01..28 and SUB-01..04 checklists as scoped ("every domain with `model.go`" / "action-event packages") do not formally attach. The items below are the ones with a real analog in this diff, evaluated on their merits rather than skipped by classification:

| ID (analog) | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 analog | Handler/constructor takes `logrus.FieldLogger`, not `*logrus.Logger` | PASS | `func Apply(l logrus.FieldLogger) func(ctx context.Context) func(...)` — `mprecovery.go:49`. |
| DOM-12 | No `os.Getenv()` in handler | PASS | `grep os.Getenv` on `mprecovery/*.go` → zero matches. |
| DOM-13 | No cross-domain orchestration bypassing the domain's own processor | PASS | Only collaborator is `character.NewProcessor(l, ctx)` (`GetById`, `ChangeHP`, `ChangeMP`) — `mprecovery.go:24,33,38`. No other domain's processor is touched. |
| DOM-14 | Handler doesn't call providers directly | PASS | No `provider.go`-style function is called anywhere in the package; only `character.Processor` interface methods. |
| DOM-15 | No direct entity creation/DB writes in handler | PASS | `grep 'db.Create\|db.Save\|db.Delete'` on `mprecovery/*.go` → zero matches; all state changes are Kafka commands emitted via `character.Processor.ChangeHP/ChangeMP` (`mprecovery.go:32-39`), which itself already routes through `producer.ProviderImpl` (`character/processor.go:277,285`) — not reimplemented here. |
| DOM-17 analog | Errors surfaced, not swallowed | PASS | `loadCaster` error returns immediately (`mprecovery.go:71-74`); `changeHP` error returns before `changeMP` is attempted (`mprecovery.go:83-86`); `changeMP` error returned (`mprecovery.go:87-91`). Verified by `TestMPRecoveryCasterLoadError`, `TestMPRecoveryChangeHPError`, `TestMPRecoveryChangeMPError` (`mprecovery_test.go:126-165`). |
| DOM-20 | Table-driven tests | **WARN (non-blocking / Minor)** | `formula_test.go:11-42` is correctly table-driven (`tests := []struct{...}` + `t.Run`). `mprecovery_test.go` is NOT table-driven: `TestMPRecoveryRegistered`, `TestMPRecoveryHappyPath`, `TestMPRecoveryCasterLoadError`, `TestMPRecoveryChangeHPError`, `TestMPRecoveryChangeMPError`, `TestMPRecoveryBadDataSkips`, `TestMPRecoveryZeroMpGainSkipsChangeMP` (`mprecovery_test.go:97-194`) are each a discrete `func Test...(t *testing.T)` with no shared table/`t.Run` structure. This mirrors the pre-existing sibling `mysticdoor_test.go` shape, but per this audit's "grade the file, not the convention" rule that precedent doesn't exempt it — it is a genuine (Minor, non-blocking) deviation from the DOM-20 pattern. Each scenario exercises a materially different seam wiring (different stubbed error returns/orderings), so a literal single `[]struct` table would need per-case function fields rather than pure data — the guideline's table-driven intent is only partially mechanical here, which is why this is graded non-blocking rather than Important. |
| DOM-24 | Kafka producer stubbed in tests that emit | PASS (N/A path, verified) | `mprecovery_test.go` never calls the real `character.Processor.ChangeHP/ChangeMP` (which would hit `producer.ProviderImpl` — `character/processor.go:277,285`); the three package-level seams `loadCaster`/`changeHP`/`changeMP` are fully replaced with in-test fakes before `Apply` runs (`mprecovery_test.go:72-90`), and restored via `t.Cleanup` (`mprecovery_test.go:73-75`) — this `t.Cleanup` reverts *test fakes* back to the package's *own* seam vars (not a `producertest`/`producer.ResetInstance` singleton reset), so the DOM-24(e) anti-pattern ("don't `t.Cleanup(producer.ResetInstance)`") does not apply here — no real producer is ever installed or reset. `grep 'AndEmit(\|message.Emit(\|producer.Produce('` on `mprecovery/*_test.go` → zero matches. |
| DOM-25 | Client wire values config-resolved | N/A | `x`/`y` (`effect.Model.X()/Y()`) are skill-balance formula inputs sourced from WZ data via the existing REST/`Extract` path (`effect/rest.go:107-108`, `162-163`), consumed server-side to compute `ChangeHP`/`ChangeMP` amounts (semantic HP/MP deltas) — not a client dispatcher mode byte, sub-op code, or notice/fail-reason code fed through a client-side lookup switch. Not the class of value DOM-25 governs (see `anti-patterns.md` "Hardcoding client-interpreted wire values"). |
| DOM-26 | Goroutines via `routine.Go` | PASS | `grep -nE '^\s*go (func\|[A-Za-z_])'` on all seven changed files (excluding none, none are `_test.go`-exempt-needing since no bare `go` found anywhere) → zero matches. |
| DOM-21 | No duplication of `libs/atlas-constants` types | PASS | Skill id sourced from `skill2.BrawlerMPRecoveryId` (`libs/atlas-constants/skill/constants.go:3199`, `Id(5101005)`), used directly at `mprecovery.go:19` and in the test at `mprecovery_test.go:39`; no local re-declaration of a skill-id type or constant. `field.Model`/`world.Id`/`channel.Id`/`_map.Id` in the test all come from `libs/atlas-constants` (`mprecovery_test.go:13-17`), not service-local reinventions. |

## Domain Checklist — `data/skill/effect` (diff-scoped)

Only the changed surface (`Y()` getter + its test) is graded; the package's pre-existing structural shape (no `builder.go`/`entity.go`/`administrator.go`/`provider.go` — it is a REST/WZ-sourced read model, not GORM-backed) predates this branch and is not part of this diff.

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| — | Getter correctly returns private field, immutable-model convention preserved | PASS | `func (m Model) Y() int16 { return m.y }` — `model.go:159-161`; `y int16` is a private struct field (`model.go` struct def, pre-existing), populated from `rest.go:163` (`y: rm.Y`, pre-existing) — the getter surfaces already-flowing data, adds no new mutable state. |
| DOM-20 | Table-driven tests | WARN (non-blocking / Minor, pre-existing file style) | `TestModelY` (`model_test.go:39-44`) is a discrete function, not table-driven — but this matches every other single-field getter test already in the file (`TestModelBulletCount`, `TestModelBulletConsume` — `model_test.go:7-19`, both pre-existing). Graded per this audit's own rule (precedent doesn't exempt), so recorded as a genuine but Minor, non-blocking deviation, not newly introduced by this diff's *pattern* (it follows the file's established shape) but present in the new code nonetheless. |

## Sub-Domain Checklist

Not applicable — no package in this diff has a `resource.go` (no new REST/action-event endpoint was added).

## External HTTP Client Checklist

Not applicable — `mprecovery` makes no direct `requests.GetRequest[T]`/`requests.PostRequest[T]` calls. It calls `character.Processor` (`GetById`/`ChangeHP`/`ChangeMP`), an already-existing, separately-governed processor; auditing that processor's own EXT-* compliance is out of scope for this diff (it is unchanged by this branch).

## Security Review

Not applicable — `atlas-channel`'s skill-handler subsystem is not an auth/token service; SEC-01..04 do not apply to this diff.

## Registration Wiring Verification

- `registrations.go:10` blank-imports `atlas-channel/skill/handler/mprecovery`, triggering `mprecovery.go:18-20`'s `init()` → `channelhandler.Register(skill2.BrawlerMPRecoveryId, Apply)`.
- Confirmed live wiring: `socket/handler/character_attack_common.go:539` gates the generic HP/MP-consume block on `handler.Lookup(skill3.Id(ai.SkillId()))` (double-deduct guard), and `skill/handler/common.go:152-156` is the actual per-skill dispatch call site: `if h, ok := Lookup(skill2.Id(info.SkillId())); ok { if err := h(l)(ctx)(wp, f, characterId, info, e); err != nil { ... } }`. Both are pre-existing dispatch machinery shared by every sibling handler (`heal`, `mysticdoor`, etc.) — unmodified by this diff, confirmed only to establish the new handler is not orphaned.
- Note (informational, not a finding against this diff): `common.go:154` logs but does not propagate a per-skill handler's error further up the call chain (`common.go:158` still `return nil`) — this is pre-existing dispatcher behavior applying uniformly to every registered handler, not something introduced or alterable by `mprecovery.go`.

## Summary

### Blocking (must fix)
None.

### Non-Blocking (should fix)
- DOM-20: `mprecovery_test.go` (`mprecovery_test.go:97-194`) uses discrete `Test...` functions rather than a `tests := []struct{...}` + `t.Run` table. Minor — each case has materially different seam-stub wiring, so full tabularization would add indirection without much readability gain, but a partial table (e.g. table the pure input/error-flag combinations, keep the seam recorder shared) would align with the guideline.
- DOM-20: `data/skill/effect/model_test.go:39-44` (`TestModelY`) is likewise not table-driven, consistent with the file's pre-existing per-getter test style (`model_test.go:7-19`).

No Critical or Important findings. Build clean, tests clean, no structural (File Responsibilities) violations, no client-wire-value hardcoding, no bare goroutines, no `os.Getenv` in handler code, no direct DB/provider access from the handler, no duplicated `atlas-constants` types, error paths correctly short-circuit the HP-before-MP emit ordering, and Kafka emission is fully seam-isolated in tests (no unstubbed producer path).

**Verdict: Ready.** (0 Critical, 0 Important, 2 Minor/non-blocking.)
