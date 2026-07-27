# Resurrection Skill — Implementation Context

Companion to `plan.md`. Key files, decisions, and dependencies an implementer needs before starting. Paths are relative to the task worktree root (`.worktrees/task-111-resurrection-skill`).

## What this builds

An active-skill handler for Bishop Resurrection (`2321006`) and the GM/SuperGM variants (`9001005`/`9101005`) in **atlas-channel**. It revives dead characters in place at full HP: per dead recipient, set HP to full, then chase-warp the recipient to its own death coordinates on the same map. The death-stance client fires its native `OnRevive` on the resulting `SET_FIELD`. No new packet is introduced; this reuses the task-093 Mystic Door warp primitive.

## Modules touched

- `libs/atlas-constants` — one missing skill constant + var + registry entry (Task 1).
- `services/atlas-channel/atlas.com/channel` — everything else.

Both have a `go.mod`; both get the full verification gate. `atlas-channel` gets the docker bake.

## Key reference files (read before editing)

| File | Why it matters |
|---|---|
| `services/atlas-channel/atlas.com/channel/skill/handler/heal/heal.go` | Handler lifecycle precedent (caster load, selector, effect broadcast). Resurrection's effect-broadcast block mirrors `heal.go:156-164`. |
| `services/atlas-channel/atlas.com/channel/skill/handler/mysticdoor/mysticdoor.go` + `mysticdoor_test.go` | **The testability pattern to copy** — `var` function seams + table-style seam-swap tests. Resurrection's `Apply` follows this, not heal (whose `Apply` calls processors directly and is less testable). |
| `services/atlas-channel/atlas.com/channel/skill/handler/recipients.go` | Shared party selector. `selectPartyMembers` (line 127) hard-codes the living-only skip at line 174 — generalized in Task 5b. Bitmap is MSB-first by party slot (line 157). Missing-rectangle → nil fallback (line 100). |
| `services/atlas-channel/atlas.com/channel/skill/handler/recipients_test.go` | Existing seam-stub test idiom: `installPartySeams`, `mkPartyMember`, `mkMemberChar`, `recipientIds`, `eqIds`, `testLogger`. Reuse these; the plan's fixtures may need adapting to them. |
| `services/atlas-channel/atlas.com/channel/skill/handler/common.go:70-126` | `UseSkill` dispatch. Confirms MP consume (73-78), cooldown (93-95), and the generic stat-up buff apply (107-111) all run **before** the per-skill `Lookup` dispatch (117). Resurrection inherits MP/cooldown/range from here. |
| `services/atlas-channel/atlas.com/channel/skill/handler/registry.go` | `Handler` type signature + `Register`/`Lookup`. Resurrection's `Apply` must match `Handler` exactly. |
| `services/atlas-channel/atlas.com/channel/character/producer.go:57-69` | `ChangeHPCommandProvider` — the exact template for `SetHPCommandProvider`. |
| `services/atlas-channel/atlas.com/channel/character/processor.go:41,271-273` | `ChangeHP` interface entry + impl — template for `SetHP`. |
| `services/atlas-channel/atlas.com/channel/kafka/message/character/kafka.go:16,55-63` | `CommandChangeHP` + `ChangeHPCommandBody` — template for `CommandSetHP` + `SetHPCommandBody`. |
| `services/atlas-channel/atlas.com/channel/portal/processor.go:50` | `WarpToPosition(f, characterId, targetMapId, x, y)` — the chase-warp primitive (task-093). Used unchanged. |
| `services/atlas-channel/atlas.com/channel/socket/handler/effects.go:19,31` | `AnnounceSkillUse(l)(ctx)(wp)(skillId, characterLevel, skillLevel)` and `AnnounceForeignSkillUse(l)(ctx)(wp)(characterId, skillId, characterLevel, skillLevel)`. |

## Cross-service facts verified (atlas-character)

- `services/atlas-character/atlas.com/character/kafka/message/character/kafka.go:29,170-173` — atlas-character already defines `CommandSetHP = "SET_HP"` and `SetHPBody{ ChannelId channel.Id; Amount uint16 }`. The channel-side `SetHPCommandBody` must use **identical JSON tags** (`channelId`, `amount`) so the existing consumer deserializes it.
- `services/atlas-character/atlas.com/character/character/processor.go:1166-1185` — `SetHP` **clamps `Amount` to effective MaxHP** (falls back to base MaxHP if stats unavailable). This is why the handler can send `0xFFFF` (`math.MaxUint16`) and get a correct full-HP fill without fetching per-recipient effective stats. It emits `DIED` only when clamped to 0 — irrelevant here (`0xFFFF` never clamps to 0 for a living-capable character).
- The channel `Command[E]` wrapper has **no** `TransactionId` field; atlas-character's does and defaults it to the zero UUID on deserialize. `CHANGE_HP` already works this way, so `SET_HP` needs no transaction plumbing.

## Key decisions (from design.md)

- **D1 — new absolute `SET_HP` producer** rather than relative `ChangeHP` (`int16` can't express "fill to max") or a respawn-style saga (adds orchestration the fresh `GetById` HP read makes unnecessary). The channel side only adds the *producing* half; atlas-character already consumes it.
- **D2 — generalize the shared selector** with a `wantDead bool` predicate; existing callers pass `false` (behavior preserved). Add `SelectDeadInRangePartyMembers` (Bishop) and `SelectDeadInRangeMapPlayers` (GM, party-agnostic). The GM selector reuses the existing `inMapCharacterIdsFunc` seam (its own mutex guards the concurrent `ForSessionsInMap` callback) and iterates serially, so the new selector itself has no concurrency.
- **D3 — in-place same-map warp** to each recipient's captured death `X`/`Y`, via `WarpToPosition` unchanged. `revive`-byte packet variant (`warp_to_map.go:109` hard-codes `0`) is **not** built up front — it's the OQ-2 live-gated fallback.

## Decisions made at plan time (beyond design.md)

- **`GmResurrectionId` (9001005) does not exist** — confirmed by grep (only Bishop `2321006` and SuperGM `9101005` are defined; GM ids run `9001000`–`9001004`). PRD FR-1 was wrong. Task 1 adds it.
- **Add `SetX`/`SetY` to the character `modelBuilder`** (Task 5a) — the `Model` has `x`/`y` fields and `X()`/`Y()` getters and `Build()` copies them, but there's no setter. The map-selector tests need characters at specific coordinates. This is the sanctioned Builder extension (no `*_testhelpers.go`).
- **No `warnIfMissingRectangle` in the resurrection handler** — heal's version is unexported/package-private, and the dead selectors already no-op on a zero rectangle, so a missing rectangle is a clean zero-recipient cast without a separate guard.
- **Caster-load failure returns `nil` without broadcasting** (matches `heal.go:82-85`) — the effect packet needs the caster's level. Design §8's "still broadcast" rule applies to the *empty recipient set* case (caster loaded fine), which the handler honors.

## Out of scope (do not implement)

- Post-revive invincibility (v83 `232.img.xml` has no such property); `character_damage.go` stays untouched.
- Experience / death-penalty interaction.
- The `revive`-byte packet variant + portal/maps plumbing (OQ-2) unless live testing requires it.
- Reviving pets/summons; self-revive.

## Testing approach

Unit tests only, function-seam style (Mystic Door pattern). Three test surfaces:
1. `libs/atlas-constants/skill/resurrection_test.go` — constant + registry assertions.
2. `services/atlas-channel/atlas.com/channel/character/producer_test.go` — `SetHPCommandProvider` payload via JSON round-trip.
3. `skill/handler/recipients_test.go` (extend) + `skill/handler/resurrection/{recipients,resurrection}_test.go` (new) — dead selectors, variant dispatch, handler ordering/no-op/isolation.

## Verification gate (CLAUDE.md)

`go test -race`, `go vet`, `go build` clean in `libs/atlas-constants` and `atlas-channel`; `tools/redis-key-guard.sh` (run with `GOWORK=off`) clean; `docker buildx bake atlas-channel`. Then live OQ-1/OQ-5 on the running environment.

## Live verification gates (post-code)

OQ-1 (dead chase-warp fires `OnRevive` on v83), OQ-2 (revive-byte fallback only if OQ-1 unreliable), OQ-3 (same-map flicker), OQ-4 (tracked X/Y == death position), OQ-5 (v87/v95/JMS parity + per-version GM id/WZ existence). See plan.md "Live verification gates".
