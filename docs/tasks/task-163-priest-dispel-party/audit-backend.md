# Backend Audit — task-163-priest-dispel-party

- **Scope:** Go files changed on branch `task-163-priest-dispel-party` vs `main`:
  - `services/atlas-channel/atlas.com/channel/skill/handler/dispel/dispel.go` (new)
  - `services/atlas-channel/atlas.com/channel/skill/handler/dispel/dispel_test.go` (new)
  - `services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations.go` (+1 import)
  - `libs/atlas-packet/model/skill_usage_info.go` (version-gated decode)
  - `libs/atlas-packet/model/skill_usage_info_test.go` (+2 tests, 4 tests migrated to tenant ctx)
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-08-07
- **Build/Test/Guards:** Reported clean by the requester (go build/vet/test -race, docker buildx bake atlas-channel, redis-key/goroutine/skill-job-id/buff-duration guards, lint.sh --check). Not re-run in this pass per instructions.
- **Overall:** PASS

## Package Classification

`skill/handler/dispel` is a socket-packet skill-handler package (registered via `channelhandler.Register` against `skill/handler/registry.go`), not a REST domain (`model.go`) nor a Kafka action-event sub-domain (`resource.go`). It carries none of the File-Responsibilities symbols (no `Processor`, `RestModel`, cross-service `requests.go` calls, `entity.go`, `builder.go`, `administrator.go`, `provider.go`). This matches the established shape of sibling packages `skill/handler/healdispel/healdispel.go` and `skill/handler/mysticdoor/mysticdoor.go`, which are cited directly in file-responsibilities.md's silence on handler packages — DOM/SUB checklists (built for REST/Kafka domains) do not have applicable rows here; the File Responsibilities Checklist is graded per-symbol below and has nothing to flag because no such symbols are defined in this package.

## File Responsibilities Checklist — `skill/handler/dispel`

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor in processor.go | N/A | No `Processor`/`ProcessorImpl` type declared in package; business logic lives in the registered `Apply` closure (`dispel.go:73-117`), matching sibling `healdispel.go`/`mysticdoor.go` shape. |
| FILE-02 | RestModel/Transform in rest.go | N/A | No `RestModel` declared; package has no REST surface. |
| FILE-03 | Cross-service requests in requests.go | N/A | No `requests.RootUrl`/`GetRequest`/`PostRequest` calls. `Apply` calls the in-service `buff.NewProcessor(l, ctx).CancelByTypes` (`dispel.go:64-66`), an internal atlas-channel package call, not a cross-service HTTP client — EXT-* checklist does not trigger. |
| FILE-04 | entity.go / Migration / TableName | N/A | No GORM entity in this package. |
| FILE-05 | Builder/Model/administrator/provider/state placement | N/A | None of these symbols are declared in the package. |
| FILE-06 | No package-named catch-all bundling ≥2 responsibilities | PASS | `dispel.go` contains only the registered handler function, three func-var seams (`selectPartyMembersFunc`, `propRollFunc`, `cancelByTypesFunc`), and a stat-type constant slice — zero of the File-Responsibilities categories, so it cannot bundle ≥2 of them. Same shape as `services/atlas-channel/atlas.com/channel/skill/handler/healdispel/healdispel.go` and `.../mysticdoor/mysticdoor.go` (both single-file handler packages), the explicit precedent the task cites. |

## DOM-21 — atlas-constants reuse

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-21 | No duplication of atlas-constants types | PASS | `dispellableStatTypes` (`dispel.go:33-40`) is built entirely from `charconst.TemporaryStatType*` constants already defined in `libs/atlas-constants/character/temporary_stat.go:6-24` (Curse/Darkness/Poison/Seal/Weaken/Slow all present there) — no new stat-type strings invented. `skill2.PriestDispel` (`dispel.go:19`) and `skill.PriestDispelId` (test file) come from `libs/atlas-constants/skill` — no new skill-id constant declared. The version gate in `libs/atlas-packet/model/skill_usage_info.go:40-41` uses `tenant.IsRegion`/`tenant.MajorAtLeast`, both pre-existing `libs/atlas-tenant` methods (`libs/atlas-tenant/tenant.go:88,93`) — no new region/version helper reinvented. |

## Decoder Change Blast Radius — `libs/atlas-packet/model/skill_usage_info.go`

| Check | Status | Evidence |
|----|--------|----------|
| Gate scoped correctly, not just to Dispel | PASS | The `isAntiRepeatBuffSkill` predicate at `skill_usage_info.go:259` is shared by every skill in that list, not just Dispel; the fix at `skill_usage_info.go:40-41` is applied at the shared gate site, so it corrects the over-read for all such skills on gms_48/gms_61, not just 2311001. |
| Caller supplies tenant in ctx (no `tenant.MustFromContext` panic risk) | PASS | Sole production call site `services/atlas-channel/atlas.com/channel/socket/handler/character_skill_use.go:47` (`sui.Decode(l, ctx)(r, readerOptions)`) runs inside the socket per-request handler, which already calls `tenant.MustFromContext(ctx)` itself at `character_skill_use.go:107`, confirming ctx is tenant-bearing on this path. |
| Regression coverage for the gate boundary | PASS | `TestDecodeDispelGms61NoCastXY` (`skill_usage_info_test.go:20-58`, no-gate case) and `TestDecodeDispelGms83HasCastXY` (`skill_usage_info_test.go:64-96`, gated case) both assert full byte-consumption (`reader.Available() > 0` check) pinning the exact boundary at gms_72. |
| Pre-existing tests not silently broken by adding tenant requirement | PASS | Four pre-existing tests (`TestDecodeBishopResurrectionReadsPartyBitmap`, `TestDecodeBuccaneerTimeLeapReadsPartyBitmap`, `TestSkillUsageInfoDecodeSpiritJavelinItemId`, `TestDecodeCorsairBattleshipV92Prefix`) were updated from `context.Background()` to `pt.CreateContext("GMS", 83/92, 1)` (diff hunks at `skill_usage_info_test.go:127,158,183,242`) rather than left to panic; `pt.CreateContext` is a pre-existing helper (`libs/atlas-packet/test/context.go:43`, added in task-157, not a new test-helper file). |
| JMS branch left conservative, documented | PASS | Comment at `skill_usage_info.go:29-33` states jms_185 could not be IDA-verified and the gate defaults to reading castX/castY (preserving current behavior) for `t.IsRegion("JMS")` regardless of major version — an explicit, honest unverified-boundary call, not a guess presented as fact. |

## Handler Correctness Spot Checks — `dispel.go`

| Check | Status | Evidence |
|----|--------|----------|
| No bare `go` statements | PASS | `grep -n '^\s*go '` over `dispel.go` returns no matches. |
| No `os.Getenv` | PASS | No matches in `dispel.go`. |
| Per-recipient failure isolation (doesn't abort the loop) | PASS | `dispel.go:97-100` logs and `continue`s on a `cancelByTypesFunc` error rather than returning, so one recipient's failure doesn't block others. |
| Seam signatures match the delegated production calls | PASS | `cancelByTypesFunc` (`dispel.go:64-66`) delegates to `buff.NewProcessor(l, ctx).CancelByTypes(f, characterId, types)`, matching `buff.ProcessorImpl.CancelByTypes` signature at `services/atlas-channel/atlas.com/channel/character/buff/processor.go:82`. |
| Registration uses Identity, not wire id | PASS | `channelhandler.Register(skill2.PriestDispel, Apply)` at `dispel.go:14` — comment at `dispel.go:9-13` documents the Identity-keyed registry / wire-id resolution split (task-187 pattern). |
| Blank import wired into registrations | PASS | `services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations.go` diff adds `_ "atlas-channel/skill/handler/dispel"` with a task-163 comment, alongside sibling `heal`/`healdispel`/`hide` imports in the same alphabetized block. |

## Test Quality — `dispel_test.go`

| Check | Status | Evidence |
|----|--------|----------|
| Coverage breadth | PASS | 9 test functions cover: registration lookup, in-order cure of caster+members, selector receiving cast args, empty-selector caster-only path, per-recipient prop roll, error-continues-loop, zero-prop cures nobody, prop-roll boundary values, and summary log field values (`dispel_test.go` grep of `func Test`). |
| Seams restored via `t.Cleanup` | PASS | Every override of `selectPartyMembersFunc`/`propRollFunc`/`cancelByTypesFunc` is paired with a `t.Cleanup` restore, e.g. `dispel_test.go:55-59,67-68`, matching the `healdispel`/`mysticdoor` precedent. |
| Table-driven pattern (DOM-20) | N/A (advisory) | DOM-20 is a domain-package (`model.go`) checklist item; this package has none. Most tests here are scenario-style rather than `tests := []struct{...}{}` table-driven — not a violation given the package's non-domain classification, noted only for completeness. |

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- None. `TestPropRollBoundaries` (`dispel_test.go:272-284`) and most scenario tests use sequential `if` assertions rather than a `tests := []struct{...}{}` table; this is stylistic only (no DOM-20 applicability here) and matches sibling handler-package test style, not called out as blocking.
