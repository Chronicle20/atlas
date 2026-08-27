# Review — Task 22: `CreateMapleLife`

Commit reviewed: `8eda3ee6b` (single commit, as instructed).
Brief: `.superpowers/sdd/plan/task-22-brief.md` (plan text + CONTROLLER AMENDMENT read in full).
Report: `.superpowers/sdd/plan/task-22-report.md`.

## 1. HP/MP arithmetic (amendment's core requirement) — PASS

Code: `services/atlas-character-factory/atlas.com/character-factory/factory/maple_life.go:245-261`.

```go
stats := e.Stats
if e.SpSkillId != 0 && in.SP > 0 && effectX != 0 {
    contribution := uint16(29 * int(effectX))
    switch e.SpSkillId {
    case uint32(skill.WarriorImprovedMaxHpIncreaseId):
        stats.Hp += contribution
    case uint32(skill.MagicianImprovedMaxMpIncreaseId):
        stats.Mp += contribution
    }
}
```

- Ordinal 0 Warrior: `skill.WarriorImprovedMaxHpIncreaseId = 1000001` (`libs/atlas-constants/skill/constants.go:2933`), only `stats.Hp` is touched — MP branch is a distinct `case`, never both. Confirmed at `maple_life.go:256-257`.
- Ordinal 1 Magician: `skill.MagicianImprovedMaxMpIncreaseId = 2000001` (`constants.go:3023`), only `stats.Mp` touched. `maple_life.go:258-259`.
- Ordinals 2-4 carry no `SpSkillId` in configuration (`maplelife.ClassEntry.SpSkillId` is `omitempty`, comment at `configuration/tenant/maplelife/rest.go:60-63` states "ABSENT (zero) means this class offers no SP step"), so the `switch` never matches either case for them — no bolted-on ordinal check, the `SpSkillId` gate does it. Verified in the fixture: ordinal 2 has no `SpSkillId` field set (`maple_life_test.go:70-78`), and `TestCreateMapleLifeSagaPayload_NoSkillClass` asserts `Hp==250`/`Mp==50` exactly.
- `nSP=0` falls out of the arithmetic via the `in.SP > 0` guard on the whole block (`maple_life.go:253`), and `effectX` is only ever populated when `in.SP > 0` (`maple_life.go:184-190`) — so even without the guard the multiply is `29*0`. Confirmed by `TestCreateMapleLifeSagaPayload_SPZero`: `Hp==600` exactly.
- Uses `effect.X`, never `Y` — `SkillEffectRestModel` (`data/skill_requests.go:17-19`) carries only `X int16 json:"x"`; there is no `Y` field anywhere in the factory's client at all, so `Y` cannot leak in even by accident.

**Test honesty verified, not just read.** I mutated `29` to `30` in `maple_life.go` and re-ran `go test ./factory/... -run TestCreateMapleLifeSagaPayload`:

```
--- FAIL: TestCreateMapleLifeSagaPayload (0.00s)
    maple_life_test.go:400: Hp: expected 1180, got 1200
--- FAIL: TestCreateMapleLifeSagaPayload_MagicianMP (0.00s)
    maple_life_test.go:474: Mp: expected 474, got 480
```

Both HP and MP assertions failed as expected, then reverted the mutation and confirmed a clean `go build`. The assertions (`600+29*4*5`, `300+29*2*3`) are computed from the fixture inputs (`effectX` table `mapleLifeWarriorEffectX = [4,8,12,...]` = `4×level`, matching the amendment's `x = 4×nSP`/`x = 2×nSP` shape) rather than a hard-coded total that happens to match — this is a genuinely derived test, not weak coverage.

## 2. The `x` sourcing path — PASS, with one coverage gap flagged (non-blocking)

Report claims the PREFERRED path (widen the factory's own client via `data/skills?ids=`, no WZ hard-coding). Verified against atlas-data's actual resource:

- `services/atlas-data/atlas.com/data/skill/resource.go:23-25` — `/data/skills` (`search_skills`) is a real, already-existing endpoint (`GET data/skills?ids=`), returning `skill.RestModel`.
- `services/atlas-data/atlas.com/data/skill/rest.go:8-16` — `RestModel.Effects []effect.RestModel`.
- `services/atlas-data/atlas.com/data/skill/effect/rest.go:37` — `X int16 json:"x"` is a real field on the effect resource, populated by `Transform` from the domain model (`effect/rest.go:143: X: m.X()`), not a stub.
- Factory's own widened client (`data/skill_requests.go:11-19,22-29`) mirrors the JSON shape exactly: `SkillEffectRestModel{X int16 json:"x"}`, `SkillRestModel.Effects []SkillEffectRestModel`.
- `data/processor.go:60-73` (`GetSkillsByIds`) correctly maps `rm.Effects[i].X` → `SkillInfo.EffectX[i]`, and `EffectXAt` (`data/processor.go:27-35`) correctly treats index `level-1` as level `level`, matching atlas-character's own `GetEffect`'s `Effects()[level-1]` convention (`services/atlas-character/atlas.com/character/data/skill/processor.go:34-44`).

This is a real, existing endpoint carrying real per-level `x` — not an invented or absent one. The escalation to WZ constants was correctly not taken.

**Coverage gap (non-blocking, worth fixing):** the `data` package has zero test files (`find . -path './data/*_test.go'` returns nothing; `go test ./...` confirms `atlas-character-factory/data [no test files]`). The JSON-decode step from `SkillRestModel.Effects[].X` into `SkillInfo.EffectX` — the exact step the brief warned "if `x` silently decodes to zero, every SP investment is worth nothing and every test above still passes" — is untested. All of `TestCreateMapleLife*`'s HP/MP assertions go through `dmock.ProcessorMock`, whose `GetSkillsByIds` (`data/mock/processor.go:16-27`) returns the fixture `data.SkillInfo` values directly, bypassing `SkillRestModel`/JSON entirely. If the JSON tag on `SkillEffectRestModel.X` were ever wrong (e.g. `json:"X"` capitalized, or a typo), or the `Effects` field name mismatched atlas-data's wire shape, no test in this diff would catch it. This mirrors a pre-existing convention in this package (the earlier `Id`/`Name`/`MaxLevel` fields were never unit-tested for decode either), and the JSON shape itself is directly modeled on atlas-character's already-working `effect.RestModel`, so the practical risk is low — but it is a real, undetected gap on the highest-stakes new field this task adds. Recommend a follow-up: a small `data` package test (or an integration test) that decodes a literal `data/skills` JSON:API response fixture and asserts `SkillInfo.EffectX` is non-zero.

## 3. Unbriefed cross-service change (`preset/rest.go` in both repos) — PASS, additive and consistent

`services/atlas-character-factory/.../configuration/tenant/characters/preset/rest.go` and `services/atlas-configurations/.../tenants/characters/preset/rest.go` both gain:

```go
AP uint16 `json:"ap"`
SP string `json:"sp"`
```

- `diff` of the two files after the commit is empty — confirmed byte-identical, so the mirror pair stays in sync (this is the established convention for these two files; the two configurations repos ship the same `RestModel` struct twice by design, per the report and no other file in this diff family disputes it).
- The change is purely additive: new fields with Go zero values (`0`, `""`) and JSON tags that were absent before. Any existing preset document (stored config, or in-flight consumer) that doesn't populate `ap`/`sp` decodes to the zero value — reproducing the exact pre-existing behaviour (`AP: 0`, `SP: ""` on `CharacterCreatePayload`, same as before this commit when those fields simply weren't set). The comment at `rest.go:28-32` makes this explicit and correct.
- `buildPresetCharacterCreationSaga` now sets `AP: a.AP, SP: a.SP` on `CharacterCreatePayload` (`factory/processor.go:395-396`) for *every* caller, including the existing admin `CreateFromPreset` path — not just Maple Life. Verified this doesn't regress `CreateFromPreset`: for an admin preset lacking `ap`/`sp` in its stored JSON, `a.AP`/`a.SP` are the zero value, so `CharacterCreatePayload.AP=0`/`SP=""`, matching what `CharacterCreatePayload.AP`/`SP` already defaulted to before Task 17 added the fields (Task 17 added the payload fields with no producer; this task is the first producer). No existing `CreateFromPreset` test asserts a *different* AP/SP value that this would break — `go test ./...` (module-local) shows all pre-existing tests still green, and I additionally re-ran the full suite from a clean tree.
- No other consumer of `preset.RestModel`/`preset.Attributes` was found to break: `grep -rn "preset.Attributes{" services/atlas-character-factory` shows only `CreateFromPreset`'s decode path and this task's `toPreset`, both of which build the full literal rather than relying on struct-tag positional ordering, so the field insertion is safe.

This is a real, intentional four-file widening exactly matching the escalation guard's named scope ("the two configurations mirrors, the factory mirror, the saga builder") — not a scope creep. Treating it as blocking would be wrong; it is exactly what the amendment asked for.

## 4. The `resolveMapleLifePreset` testability seam — sound, PASS

`CreateMapleLife` (`maple_life.go:53-72`) calls `resolveMapleLifePreset` then `buildPresetCharacterCreationSaga` then `saga.NewProcessor(...).Create(sg)` — three calls, in that order, no branching. The saga-payload tests (`TestCreateMapleLifeSagaPayload*`) call the exact same first two functions directly and inspect the returned `saga.Saga` struct before it would be handed to the (no-op, per `producertest.InstallNoop()`) Kafka producer. This is not a bypass: the only step skipped is the actual Kafka emit, which is orthogonal to payload correctness and is exercised by the `TestCreateMapleLife` table's "no error; a saga is created" happy-path assertion (which does call `CreateMapleLife` itself, round-tripping through `saga.NewProcessor(...).Create`). Both functions are unexported and used by production code with no alternate code path — there is no way for `CreateMapleLife` to diverge from what the tests observe. Ruling: sound.

## 5. Error-case table and `TestCreateMapleLifeNeverConsultsCreationTemplates` — PASS, all present

Swept the brief's table against `maple_life_test.go`'s `TestCreateMapleLife` table:

| Error | Covered? | Location |
|---|---|---|
| `ErrClassOrdinalUnknown` | yes (unknown ordinal + wrong-gender-same-ordinal) | `maple_life_test.go:169-189` |
| `ErrSPInvalid` | yes (no-skill class + above pool) | `:190-212` |
| `ErrLookInvalid` | yes (face/hair/hairColor/skin/gender) | `:213-267` |
| `ErrNameDuplicate` | yes | `:268-274` |
| `ErrMapleLifeNotConfigured` | yes (empty `tenant.RestModel{}`) | `:282-289` |
| `ErrAtlasDataUnreachable` | yes (`SkillsErr` set) | `:290-296` |
| `*NameInvalidError` | yes (`Reason: "blocked"`, asserted via `errors.As`) | `:275-281`, `:305-313` |

`TestCreateMapleLifeNeverConsultsCreationTemplates` (`:509-525`) publishes a config with an empty `Characters.Templates` (asserted as a fixture precondition at `:515-517`) and a populated `MapleLife` block, then asserts `CreateMapleLife` succeeds — a real behavioural test of §11 A5, not a comment. Nothing in `resolveMapleLifePreset` touches `tc.Characters` at all (confirmed by reading `maple_life.go` — only `tc.MapleLife` is read), so this is structurally guaranteed, and the test pins it.

All seven items from the brief's table are present. No gaps found.

## Other checks

- `go build ./... && go test ./...` from `services/atlas-character-factory/atlas.com/character-factory` — all green, matching the report.
- `services/atlas-character-factory/atlas.com/character-factory/data` package has no test files at all (pre-existing convention, not introduced by this task, but relevant to §2 above).
- Reused `validGender`/`validOption`/`ErrNameDuplicate`/`ErrPresetValidation`/`ErrAtlasDataUnreachable`/`NameInvalidError` rather than duplicating (`factory/processor.go:29-42,477-499`) — confirmed by direct read, not just the report's claim.
- Ordinal→class mapping stays data-driven: the HP/MP `switch` keys off `e.SpSkillId` (configuration data), not `e.Ordinal`, consistent with the repo's own "ordinal→class mapping is DATA, not code" convention documented at `configuration/tenant/maplelife/rest.go:43-45`.
- Not re-raised (per reviewer instructions): malformed-SP legacy rows, `se.Y()`/`se.X()` AP-path asymmetry, socket corpus count.

## Verdict rationale

No blocking defects found. One non-blocking coverage gap (§2: the `data` package's JSON-decode step for the new `EffectX` field is untested, so a wire-shape mismatch would silently zero every SP investment and no test in this diff would catch it) is worth a follow-up but does not, on the evidence gathered, indicate an actual decode defect — the JSON tags and field names were directly cross-checked against atlas-data's real resource and match.
