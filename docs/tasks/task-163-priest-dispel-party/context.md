# Task-163 Context — Priest Dispel Party Debuff Cure

Companion to `plan.md`. Key files, decisions, and dependencies for implementers.

## Worktree / Module

- Worktree: `.worktrees/task-163-priest-dispel-party/`, branch `task-163-priest-dispel-party`.
- Only changed module: `services/atlas-channel/atlas.com/channel` (module name `atlas-channel`). Run all `go` commands from that directory.
- `services/atlas-buffs/` must end the task with zero diffs — its `CANCEL_BY_TYPES` support is already complete.

## Key Files

| File | Role |
|---|---|
| `services/atlas-channel/atlas.com/channel/kafka/message/buff/kafka.go` | Channel-side mirror of the buff command envelope. Gains `CommandTypeCancelByTypes` + `CancelByTypesCommandBody`. |
| `services/atlas-channel/atlas.com/channel/character/buff/producer.go` | Kafka providers. Gains `CancelByTypesCommandProvider` (mirror of `CancelCommandProvider`). |
| `services/atlas-channel/atlas.com/channel/character/buff/processor.go` | Buff processor. Interface + impl gain `CancelByTypes(f, types) model.Operator[uint32]` (curried, typed→string conversion once). No mock exists for this processor. |
| `services/atlas-channel/atlas.com/channel/skill/handler/dispel/` | New per-skill handler subpackage (registration, seams, prop roll, summary log). |
| `services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations.go` | Blank-import list that drives handler `init()` in production. |
| `services/atlas-channel/atlas.com/channel/skill/handler/recipients.go` | `SelectPartyMembersInMap` (line 116) — map-wide bitmap selector, already filters offline/other-map/no-session/dead and decodes the MSB-first bitmap. Consumed, not modified. |
| `services/atlas-channel/atlas.com/channel/skill/handler/common.go` | UNTOUCHED. Mob half of Dispel (`applyToMobs`, `isCrashOrDispel`, magic-reflect skip) + the per-skill dispatcher (line 117) that invokes the new handler. Its `propRollFunc` (line 45) is the semantics the dispel package mirrors. |
| `services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go` | Wire contract source of truth (`CancelByTypesCommandBody{ Types []string }`, type `CANCEL_BY_TYPES`). Read-only reference — cross-module import is impossible; the producer test asserts literal JSON field names. |

## Precedents to Follow

- **Handler shape + seams + tests:** `skill/handler/mysticdoor/mysticdoor.go` + `mysticdoor_test.go` — package-level func-var seams, `t.Cleanup` restore, `Apply(l)(ctx)(wp, f, characterId, info, e)` signature.
- **Per-recipient error handling:** `skill/handler/heal/heal.go` — log and continue, never abort the cast.
- **Producer/processor shape:** `character/buff/producer.go` `CancelCommandProvider` and `processor.go` `Apply` (curried operator).
- **Registry test:** `skill/handler/registry_test.go`.

## Decisions Locked in Design (do not re-litigate)

1. **Approach A**: per-skill handler subpackage + shared `CancelByTypes` on the buff processor. Not inline in `UseSkill`, not an atlas-buffs `DISPEL` command.
2. **Cure set is exactly six**: CURSE, DARKNESS, POISON, SEAL, WEAKEN, SLOW (`libs/atlas-constants/character/temporary_stat.go`). ZOMBIFY/SEDUCE/CONFUSE excluded (task-156 cure-all semantics).
3. **Map-wide recipients, no rectangle**: caster + `SelectPartyMembersInMap` bitmap selection. The WZ lt/rb rect governs only the mob half.
4. **Prop roll per recipient** (caster included), mirrored `propRollFunc` (not shared — parent seam is unexported; heal precedent). `e.Prop()` is pre-normalized 0.0–1.0 (`data/skill/effect/model.go:138`) — no /100.
5. **Handler always returns nil** — failures logged, cast never aborts.
6. **Stat set stays handler-local**; the processor method takes the set as a parameter so task-156 can pass its wider set.

## Dependencies / Coordination

- **atlas-buffs (existing, verified):** consumer `handleCancelByTypes` (`kafka/consumer/character/consumer.go:81`) → `CancelByStatTypes` (`character/processor.go:103`) → cancels intersecting buffs → `EXPIRED` status events → atlas-channel buff status consumer → client buff-cancel packets. End-to-end path verified in acceptance, not reimplemented.
- **task-156 (gm-hide-heal-dispel, in design):** plans the same `CancelByTypes` producer. task-163 builds it; whichever lands second rebases and consumes the shared method.
- **No mob→character disease infliction exists yet** (PRD open question 1): live Dispel casts find nothing to cancel until that lands. Acceptance seeds a debuff via a direct buff `APPLY` command with a debuff stat type.
- **Client bitmap open question** (PRD open question 2): if a client version sends an empty bitmap, the selector returns nil and only the caster is cured — accepted behavior, verify during implementation testing.

## Verification Gates (CLAUDE.md + design §7)

1. `go test -race ./...`, `go vet ./...`, `go build ./...` clean in `services/atlas-channel/atlas.com/channel`.
2. `docker buildx bake atlas-channel` from the worktree root (mandatory — `go build` won't catch shared-Dockerfile gaps).
3. `tools/redis-key-guard.sh` clean from the worktree root.
4. `git diff --stat main...HEAD -- services/atlas-buffs/ ...skill/handler/common.go` EMPTY.
5. Code review (`superpowers:requesting-code-review`) before any PR.

## Gotchas

- Test files that override the unexported seams must be internal (`package dispel`), not `package dispel_test`.
- `SkillUsageInfo.AffectedPartyMemberBitmap()` has a pointer receiver; calling it on an addressable value is fine (heal does the same).
- The bitmap is MSB-first by party slot (slot i → bit 5-i) — but the dispel handler never decodes it; it passes the raw byte to the selector.
- `effect.Model{}` zero value has Prop 0 → the roll always fails; tests needing passes must build the effect via `effect.Extract(effect.RestModel{Prop: 1.0})`.
- Do not add `*_testhelpers.go` files (project rule); the test harness lives inside the `_test.go` file.
