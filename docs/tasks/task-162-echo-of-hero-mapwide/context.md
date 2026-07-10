# task-162 Echo of Hero Map-Wide — Execution Context

## What this task does

The four X005 Echo of Hero skills (`1005`, `10001005`, `20001005`, `20011005`) currently buff only the caster. This task routes their casts to a map-wide recipient set: every live-session character in the caster's field, caster included exactly once, excluding dead characters (HP 0) and hidden GMs (any buff with `SourceId == 9101004`). Server-side routing only — no packet, data, or buff-service changes.

## Key files

| File | Why it matters |
|---|---|
| `services/atlas-channel/atlas.com/channel/skill/handler/common.go` | `UseSkill` owns the generic buff step (lines 107–111 pre-change). Gains `isEchoOfHero`, `applyBuffToRecipients` (routing), `applyToMap` (fan-out + `echo_of_hero_apply_summary` debug log). |
| `services/atlas-channel/atlas.com/channel/skill/handler/recipients.go` | Party selectors + seams live here. Gains `loadBuffsFunc` seam, `MapWideSelectionStats`, `SelectMapWideRecipients`. Reuses `PartyRecipient`/Builder, `inMapCharacterIdsFunc`, `loadPartyMemberFunc`. |
| `services/atlas-channel/atlas.com/channel/character/buff/processor.go` | `GetByCharacterId` (line 18) backs the hidden-GM check; `Apply` (line 19) is the unchanged per-recipient buff operator. |
| `services/atlas-channel/atlas.com/channel/character/buff/model.go` | `Model.SourceId()` (line 31); `NewBuff(...)` constructor used by tests to fabricate hide buffs. |
| `libs/atlas-constants/skill/constants.go` | All five ids exist: `BeginnerEchoOfHeroId=1005` (l.2908), `SuperGmHideId=9101004` (l.3247), `NoblesseEchoOfHeroId=10001005` (l.3262), `LegendEchoOfHeroId=20001005` (l.3378), `EvanEchoOfHeroId=20011005` (l.3420). DOM-21: define nothing new. |
| `libs/atlas-packet/model/skill_usage_info.go` | MUST NOT change (FR-4, IDA-verified in design §1). X005 stays out of `isPartyBuff`. |

## Locked decisions (design.md §2)

- **D1** Routing is a branch inside the generic buff step, not a registry handler (registry dispatches *after* the buff step → would double-apply the caster) and not a mount-style pre-buff short-circuit (would duplicate effect gating).
- **D2** `SelectMapWideRecipients` returns `[]PartyRecipient` — caster enumerated by the session sweep like everyone else, applied exactly once.
- **D3** Hidden-GM detection = per-recipient buff fetch, skip on any `SourceId() == int32(skill.SuperGmHideId)`. Vacuously true until task-156 lands; implement anyway (FR-2.2).
- **D4** Uniform skip-and-continue on any per-recipient failure (character fetch, buff fetch, or apply error). Never abort the cast.
- **D6** Collect ids via the existing mutex-guarded `inMapCharacterIdsFunc` sweep, then fetch/filter **sequentially**, ids sorted ascending (deterministic tests/logs).

## Plan deviations from design sketches (intentional, behavior-identical)

- Routing extracted to `applyBuffToRecipients(l, ctx, f, characterId, info, applyBuffFunc)` instead of inline in `UseSkill`; `applyToMap` takes plain args (matching `applyToMobs`) instead of the curried sketch. This is what lets tests inject a recorded operator without emitting Kafka.
- `MapWideSelectionStats` carries `inMap` (not the design parenthetical's `applied`) — successful applies are countable only in `applyToMap`'s operator loop; the summary log gets both.

## Dependencies & cross-task notes

- **task-156** (`gm-hide-heal-dispel`): defines hide as a `DARK_SIGHT` buff with `SourceId == 9101004`. Not a build dependency — the check compiles and tests against stubbed buffs today and becomes effective when task-156 lands.
- **No new module deps** — `go.mod` untouched, so `docker buildx bake atlas-channel` is not required (re-verify in Task 3 Step 6).
- Existing party-buff behavior (bitmap MSB-first, `SelectPartyMembersInMap`) is frozen (FR-1.1); `recipients_test.go` is the regression guard.

## Test conventions in this package

- Seam-variable override + `t.Cleanup` restore (`installPartySeams` precedent, `recipients_test.go:30`). New `installMapWideSeams` follows it; no `*_testhelpers.go` files.
- Shared fixtures already in package scope: `testLogger()`, `mkField()` (`common_apply_to_mobs_test.go:104`), `mkMemberChar()`, `recipientIds()`, `eqIds()`, `testCasterId=100`/`testMemberA=101`/`testMemberB=102`, `threePersonParty`, `mkPartyMember`.
- `character.NewModelBuilder().SetId(...).SetHp(...).SetMaxHp(...).MustBuild()` builds character models; `buff.NewBuff(sourceId, level, duration, changes, createdAt, expiresAt)` builds buff models.

## Verification gate (Task 3)

`go test -race ./...`, `go vet ./...`, `go build ./...` in `services/atlas-channel/atlas.com/channel`; `tools/redis-key-guard.sh` from worktree root (never with a global `GOWORK=off` prefix); `git diff --name-only main...HEAD -- libs/ services/atlas-data services/atlas-buffs` must be empty. Code review (`superpowers:requesting-code-review`) is mandatory before any PR.
