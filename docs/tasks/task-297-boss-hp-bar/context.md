# Boss HP Bar (task-297) — Implementation Context

Companion to `plan.md`. Everything here was read out of the tree at plan time; each claim
carries its `file:line`.

## Scope in one line

atlas-channel only. The packet, the body helper, the dispatcher spec and the atlas-data
fields all already exist — this task adds the three emit sites and the data plumbing.

Module root for every build/test: `services/atlas-channel/atlas.com/channel` (module
`atlas-channel`).

## What already exists (do not rebuild)

| Thing | Where | State |
|---|---|---|
| `EffectBossHp` struct + Encode/Decode | `libs/atlas-packet/field/clientbound/effect.go:126-166` | byte-verified on gms_v48/61/72/79/83/84/87/95 and jms_v185 (`effect_test.go:10-49`) |
| `FieldEffectBossHpBody` | `libs/atlas-packet/field/field_effect_body.go:57` | already routes through `atlas_packet.WithResolvedCode("operations", "BOSS_HP", …)` — NFR-2 needs no work |
| `BOSS_HP` dispatcher entry | `docs/packets/dispatchers/field_effect.yaml` | present |
| `tag_color` / `tag_background_color` on atlas-data | `services/atlas-data/atlas.com/data/monster/rest.go:37-38`, parsed at `reader.go:85-89` (`hpTagColor` / `hpTagBgcolor`) | served today |
| `MonsterId` on every monster status event | `services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka.go:214`; populated by `statusEventProvider` in `services/atlas-monsters/atlas.com/monsters/monster/producer.go` for `KILLED` and `DESTROYED` alike | verified |

## Key files, and what each one is for

| File | Role in this task |
|---|---|
| `services/atlas-channel/atlas.com/channel/data/monster/{rest.go,model.go}` | the narrow projection to widen; today only `Boss` + `FixedDamage` (`rest.go:8-12`) |
| `.../data/monster/processor.go:26` | uncached `GetById`; gains the read-through cache |
| `.../data/skill/cache.go` | the cache being ported, verbatim except for the metrics calls |
| `.../monster/live_mirror.go:23-35`, `:73-83` | `LiveEntry` + `LiveEntryFromModel`; gains `MaxHp` |
| `.../kafka/consumer/monster/consumer.go:257-305` | `handleStatusEventDamaged` — damage hook |
| `.../kafka/consumer/monster/consumer.go:195-219`, `:307-324` | `handleStatusEventDestroyed` / `handleStatusEventKilled` — death hooks |
| `.../kafka/consumer/map/consumer.go:772-790` | `spawnMonsterForSession` — field-entry hook |
| `.../monster/bosshp/` | new package; the single home for the FR-1 rule |

## Decisions carried in from the design, and why

- **`KILLED`, not just `DESTROYED`** (design F1). A boss killed by damage emits
  `DAMAGED` then `KILLED` and is removed from the atlas-monsters registry
  (`services/atlas-monsters/atlas.com/monsters/monster/producer.go`
  `killedStatusEventProvider`). Hooking only `DESTROYED`, as FR-8 literally reads, would
  mean the gauge never empties on a kill. Both are hooked.
- **Max HP comes from the live mirror, not a re-fetch** (design F2/D4). At kill time the
  monster is gone from atlas-monsters, so `monster.NewProcessor(...).GetById` cannot
  serve the denominator. `LiveEntry` gains `MaxHp`, seeded once from `LiveEntryFromModel`
  — no per-damage mirror write on the hot path.
- **Cache in `data/monster`, not on the mirror** (design D2). Sourcing tag colours onto
  the mirror would need an atlas-data lookup on every `CREATED`, i.e. per ordinary-mob
  spawn — strictly worse. The `e.Body.Boss` pre-filter still keeps ordinary mobs off the
  damage path entirely.
- **Metrics not ported.** `data/skill/cache.go` calls `recordCache` from
  `data/skill/metrics.go`; the monster port drops those calls. Observability is a
  separate concern and NFR-5 asks only for error logging.
- **Field-entry hook inside `spawnMonsterForSession`** (design D7). Placing it there makes
  FR-12 (Spawn before gauge), FR-13 (one per monster, enumeration order) and FR-14 (no
  lookup on an empty field) structural rather than conventional.

## Deliberate deviations introduced by this plan

- **Task 7 routes `spawnMonsterForSession`'s two existing announces through the
  pre-existing `doorAnnounce` seam** (`kafka/consumer/map/consumer.go:825`). Behaviour is
  identical — that var *is* `session.Announce(l)(ctx)(wp)(name)(enc)(s)` — and it is
  already the file's general announce seam (it carries ContiMove, Jukebox and
  FieldObstacle traffic, not only doors). Without it, FR-12's ordering claim has no
  deterministic assertion, because sessions constructed in that test package carry a nil
  connection and cannot take a real announce.
- **FR-5 has no wire-level unit assertion.** The `MonsterHealth` broadcast runs through
  `session.Announce` inside `_map.ForSessionsInMap` with no seam. Task 5 asserts the two
  observable proxies instead (the shared `monsterGetByIdFn` fetch stays at exactly one
  call; the handler completes with the gauge recorded), and the real "both bars coexist"
  proof is the live smoke, AC-13.

## Task sizing

Eight tasks, none over 5 files, all in one service — no F4 warning is expected and no task
is deliberately oversized. Tasks 5 and 6 both edit
`kafka/consumer/monster/consumer.go`; they are split because damage and death are
independent acceptance surfaces with separate review interest, and the edits do not
overlap (Task 5 touches `handleStatusEventDamaged` and adds the shared seam; Task 6
touches the two death handlers and consumes it). Run them in order.

## Open item carried forward, not resolved

`template_gms_12_1.json` and `template_gms_92_1.json` carry no `CField::OnFieldEffect`
writer at all (design OQ1) — the whole field-effect family is absent on those two
versions, not just `BOSS_HP`. Adding it is a per-version bring-up with its own IDB
derivation and `packet-audit` verification, so Task 8 writes the finding up rather than
widening scope. On those tenants the feature degrades silently and safely:
`session.Announce` fails at `writerProducer(writerName)`
(`services/atlas-channel/atlas.com/channel/session/processor.go:265-270`) before any
encoding, and every call site logs and continues (NFR-3).
