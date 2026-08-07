# Meso Explosion — Implementation Context

Companion to `plan.md`. Key files, resolved decisions, and dependencies for executors.

## Task summary

Skill 4211006 (Chief Bandit Meso Explosion) sends a CLOSE_RANGE_ATTACK variant with three version-invariant wire deltas (design §2.1): per-mob 1-byte damage-line count replacing the 2-byte delay, a trailing `{int32 dropId, byte hitMask}` list after charX/charY, and a trailing int16 delay. Server work: decode it, validate the listed drops against the field, destroy them via the existing drop CONSUME route, apply client-trusted damage through the existing pipeline.

## Key files

### libs/atlas-packet (module: `libs/atlas-packet`)
- `model/damage_info.go` — gains `mesoExplosion bool` + `NewMesoExplosionDamageInfo()`; Decode/Encode branch count-byte vs delay. Mob CRC unchanged.
- `model/attack_info.go` — variant detection right after `skillId` decode (`skill.Id(m.skillId) == skill.ChiefBanditMesoExplosionId`); meso-mode `DamageInfo` construction in the entry loop; variant tail after `characterY`; new `ExplodedMesoDrop` type + accessors (`ExplodedMesoDrops() []uint32`, `ExplodedMesoDropEntries()`, `MesoDelay()`) and builders (`SetExplodedMesoDrops`, `SetMesoDelay`). **No new version gates.**
- `model/attack_info_test.go`, `model/damage_info_test.go` (new), `character/serverbound/attack_request_test.go` — fixtures. `character/clientbound/attack_test.go#TestAttackMeleeWithMesoExplosionRoundTrip` already exists and covers FR-11; do not duplicate.
- Test harness: `libs/atlas-packet/test` (`pt.Variants` = GMS v28/83/87/95, JMS v185, GMS v84/86; `pt.RoundTrip` fails on unconsumed bytes; `pt.Encode` for byte-equality).

### services/atlas-channel (module: `atlas-channel` at `services/atlas-channel/atlas.com/channel`)
- `socket/handler/character_attack_common.go` — `processAttack`: validation gate after `se` load / before the cost block (`handler.Lookup` gate at line ~303); consume emission post-broadcast next to the projectile emission; TODO at line 407 removed.
- `socket/handler/character_attack_meso_explosion.go` (new) — pure `validateMesoExplosion(dropIds, fieldDrops, maxCount) (offendingId, ok)`.
- `data/skill/effect/model.go` — `attackCount` field exists (line 54) but had no accessor; add `AttackCount() uint32`.
- `drop/` — `Model.Meso() > 0` is the meso predicate (same as pickup consumer `kafka/consumer/drop/consumer.go:179`); `Processor.InMapModelProvider(f)` is the one-REST-fetch field drop source; `NewModelBuilder()` for tests (`SetId/SetMeso/SetItem/.../MustBuild`).
- `kafka/message/drop/kafka.go` — gains `CommandTypeConsume`, `ConsumeCommandBody{DropId}`, and `TransactionId uuid.UUID` on `Command[E]` (atlas-drops' envelope at `services/atlas-drops/atlas.com/drops/kafka/message/drop/kafka.go:66-75` already has it; zero-UUID from the old reservation provider stays wire-compatible).
- `drop/producer.go` / `drop/processor.go` — `ConsumeAllCommandProvider(txId, f, dropIds)` builds N `producer.RawMessage` (key = `producer.CreateKey(int(dropId))`) through `producer.MessageProvider(model.FixedProvider(raws))`; `Processor.ConsumeAll` emits once via `producer.ProviderImpl`.

### Untouched by design
- `services/atlas-drops` — `ConsumeAndEmit` already removes + emits CONSUMED; a meso-only guard there would break `services/atlas-reactors` (`item_reactor.go:122` consumes item drops). Attack-side validation owns the meso-only restriction.
- `socket/writer/character_attack_melee.go` — already sets `isMesoExplosion` (line 19); clientbound encoder already writes per-mob counts from `len(di.Damages())`.
- `kafka/consumer/drop/consumer.go:134-149` — CONSUMED already broadcasts `DropDestroyWriter` with `DropDestroyTypeExplode`.
- `processDamageInfoEntry` — variable-length `di.Damages()` flows through unchanged.

## Resolved decisions

1. **Variant flag inside the existing codec** (design §4.1-A) — not a separate codec, not handler-side decode.
2. **Validation before the cost block** (design §4.2-A) — FR-6 zero-side-effects; the HP/MP deduction at `character_attack_common.go:303-310` is a side effect.
3. **One buffered emission, N messages** (design §4.3-A) — single produce call, one message per drop keyed by dropId, one shared `uuid.New()` transaction id.
4. **Rejection does NOT destroy the session** (design §6) — a stale drop id occurs legitimately (pet/player picks a meso mid-flight); warn log only, return nil.
5. **No new `packet-audit:verify` markers** (plan-time resolution of design §5.1 ambiguity): melee cells are pinned to the registry-primary fname `CUserLocal::TryDoingNormalAttack` (evidence records are one-per-cell; audit reports reference only that address); the meso senders are registered as `fname_alts` in all five registries but are absent from the IDA exports, so a second marker would fail `matrix --check` as an orphan (its `ida=` matches neither evidence nor report). Variant evidence lives as documentation comments in `attack_request_test.go` + the jms audit MD note.
6. **jms_v185 serverbound variant implemented, not proven** (design §2.3) — sender `0xa3aab1` SCY-virtualized; compensating evidence: identical deltas across four verified versions, verified jms dispatch + drop collection + clientbound meso symmetry (`0xa53999` region). **gms_v92**: no IDB, follows the `GMS >= 87` family branch, documented unverified.
7. **Nibble trap** (design §2.2) — the mask low nibble is `nMaxAttackCount & 0xF` (wraps at 16); only the per-mob count byte sizes damage arrays. Fixtures set hits=0 with non-empty per-mob counts so nibble-misuse fails loudly.
8. **`attackCount` = max detonatable drops** — WZ-verified 10/12/14/16/18/20 by level bracket (design §3); atlas-data already parses and serves it end-to-end.

## IDA evidence (design §2, verified 2026-07-10)

| Version | Sender | Address |
|---|---|---|
| gms_v83 | `CUserLocal::DoActiveSkill_MesoExplosion` | `0x96b3fb` |
| gms_v84 | meso sender (IDB label wrong: says TryDoingMeleeAttack; real melee sender is `0x989692`) | `0x9aa379` |
| gms_v87 | `CUserLocal::DoActiveSkill_MesoExplosion` | `0x9eee04` |
| gms_v95 | `CUserLocal::DoActiveSkill_MesoExplosion` (typed IDB) | `0x942200` |
| jms_v185 | `sub_A3AAB1` — encode tail SCY-virtualized | `0xa3aab1` |

Full verified write order: design §2.1.

## Dependencies between tasks

- Task 2 (AttackInfo) needs Task 1 (`NewMesoExplosionDamageInfo`).
- Task 3 (wrapper fixture) needs Tasks 1–2.
- Task 6 (wiring) needs Task 2 (`ExplodedMesoDrops()`), Task 4 (`validateMesoExplosion`, `AttackCount()`), Task 5 (`ConsumeAll`).
- Tasks 4 and 5 are independent of each other and of Tasks 1–3 (channel-side only).
- Task 7 (audit artifacts) needs Task 3's test to exist (the MD note cites it).
- Task 8 (verification) last.

## Verification gates (PRD AC-6)

- `go test -race ./...`, `go vet ./...`, `go build ./...` in `libs/atlas-packet` and `services/atlas-channel/atlas.com/channel`.
- `tools/redis-key-guard.sh` from worktree root (no `GOWORK=off` prefix).
- `docker buildx bake atlas-channel` from worktree root.
- `go run ./tools/packet-audit matrix --check`: no NEW problems (pre-existing 🟥 conflicts keep exit 1 until task-085 Phase 5 — the bar is zero new lines mentioning our packet).
- `git diff main --stat -- services/atlas-drops` must be empty.
- Code review (`superpowers:requesting-code-review`) before PR.

## Gotchas for executors

- `pt.Variants` includes GMS v28 and v86 — the meso tests run there too; the variant has no gates, so they pass; do not add gates to exclude them.
- Existing tests decode into the same object (`ai.Encode`/`ai.Decode`), which doubles `damageInfo` — harmless because `RoundTrip` only checks unconsumed bytes; the new tests decode into a **fresh** object so accessor assertions are meaningful.
- `drop2` is the conventional alias for `atlas-channel/kafka/message/drop`; the `producer` import in `drop/producer.go` is `libs/atlas-kafka/producer`, while in `drop/processor.go` it is the service-local `atlas-channel/kafka/producer` wrapper. Both files gain a `uuid` import.
- The handler package imports `skill3` = `libs/atlas-constants/skill` and does not yet import `atlas-channel/drop` — add it in Task 6.
- The `// TODO apply Pick Pocket` line stays (explicit non-goal).
- Run guard scripts from the worktree root; never `go work sync`.
