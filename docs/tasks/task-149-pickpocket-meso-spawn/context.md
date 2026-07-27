# Pick Pocket Meso Spawn (task-149) — Execution Context

Companion to `plan.md`. Key files, locked decisions, and gotchas for implementers.

## Key Files

| File | Role |
|---|---|
| `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go` | All proc logic lands here: `shouldProc` (renamed from `mpEaterShouldProc`), whitelist, meso math, `pickPocketState`/`pickPocketResolveState`, `pickPocketTryProc`, hook widening, `processAttack` wiring. The `// TODO apply Pick Pocket` line (408 pre-change) must be gone at the end. |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_pick_pocket_test.go` | New test file (package `handler`) — all Pick Pocket unit + handler-level tests. |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_mp_eater_test.go` | Existing; `TestMpEaterShouldProc` renamed to `TestShouldProc`. |
| `services/atlas-channel/atlas.com/channel/kafka/message/drop/kafka.go` | Gains `CommandTypeSpawn` + `SpawnCommandBody` (mirror of atlas-drops' `CommandSpawnBody` minus `EquipmentData`). |
| `services/atlas-channel/atlas.com/channel/drop/producer.go` / `processor.go` | Gain `SpawnMesoCommandProvider` / `Processor.SpawnMeso`, patterned on `RequestReservationCommandProvider` / `RequestReservation`. |
| `services/atlas-channel/atlas.com/channel/drop/producer_test.go` | New provider test (package `drop_test`). |
| `services/atlas-channel/atlas.com/channel/character/buff/` | Read-only consumer: `Processor.GetByCharacterId`, `Model.Level()/Expired()/Changes()`, `stat.Model.Type()/Amount()`. |
| `services/atlas-channel/atlas.com/channel/data/skill/processor.go` | Read-only consumer: `GetEffect(uniqueId uint32, level byte) (effect.Model, error)`; `effect.Model.Prop() float64`. |
| `services/atlas-drops/atlas.com/drops/kafka/message/drop/kafka.go:123-137` | Reference only — the consumer-side `CommandSpawnBody` the channel body must mirror. **atlas-drops is NOT modified.** |

## Locked Decisions (from design.md — do not relitigate)

1. **`Mod = 0`.** The field is dead end-to-end in atlas-drops today; 0 is the semantically-correct "no animation delay" value. Do not wire Mod through atlas-drops.
2. **`DropType = 2`, `PlayerDrop = true`, `OwnerId = attacker`, `OwnerPartyId = 0`.** DropType is client-visual only; universal pickup comes from `PlayerDrop` short-circuiting `CanBeReservedBy`.
3. **Hook widening (design §3.2 option B):** `damageInfoEntryDeps.onDamageApplied` changes from `func(monsterId uint32)` to `func(di packetmodel.DamageInfo)`. No second hook, no post-loop re-iteration (would re-proc reflected entries).
4. **Eager per-attack state (design §3.3):** `pickPocketResolveState` runs once before the DamageInfo loop; whitelist check (pure) gates all I/O. At most 1 buff REST call + 1 effect lookup per attack.
5. **`maxmeso` = buff-captured PICK_POCKET stat `Amount()`** — never re-read current skill level at attack time. `prop` = effect at the buff's captured `Level()`.
6. **`shouldProc` is the single prop-roll function** (generalized from `mpEaterShouldProc`); no duplicate under a second name.
7. **Deviations from Cosmic (approved):** no `d − 2` int-overflow artifact; no 100 ms drop stagger; no GM special-casing.
8. **X offset:** `mon.X() + int16(rand.Intn(100)-50)` → uniform [−50, +49]; `Y = mon.Y()` exactly. `DropperX/Y` = monster position.
9. **Kafka key:** `producer.CreateKey(int(dropperId))`.
10. **Whitelist (Cosmic parity, no attack-type gate):** skillId 0 (basic attack), 4001334 Double Stab, 4201005 Savage Blow, 4211002 Assaulter, 4211004 Band of Thieves, 4221001 Assassinate, 4221003 Taunt, 4221007 Boomerang Step — via `skill3` constants, never raw numbers in production code.

## Verified Signatures (read from source, this worktree)

- `buff.Processor.GetByCharacterId(characterId uint32) ([]buff.Model, error)` — interface method; `buff.NewProcessor(l, ctx)` returns the interface.
- `skill2.Processor.GetEffect(uniqueId uint32, level byte) (effect.Model, error)` — `*ProcessorImpl` method value works as `func(uint32, byte) (effect.Model, error)`.
- `packetmodel.DamageInfo` methods are pointer-receiver (`Damages() []uint32`, `MonsterId() uint32`) but the struct is passed by value in `processDamageInfoEntry` — fine, parameters are addressable. Tests construct via `packetmodel.NewDamageInfo(hits).SetMonsterId(...).SetDamages(...)` (returns `*DamageInfo`; dereference when passing).
- `packetmodel.NewAttackInfo(attackType)` returns `*AttackInfo`; dereference for `processDamageInfoEntry`.
- `monster.NewModelBuilder(uniqueId, field, monsterId).SetX(...).SetY(...).MustBuild()` — Build only validates `uniqueId != 0`.
- `effect.Extract(effect.RestModel{Prop: ...})` is the only way to build an `effect.Model` in tests (fields unexported, no builder).
- `buff.NewBuff(sourceId int32, level byte, duration int32, changes []stat.Model, createdAt, expiresAt time.Time)` and `stat.NewStat(statType string, amount int32)` — exported test-usable constructors.
- `field.NewBuilder(world.Id, channel.Id, _map.Id).Build()` returns `field.Model` (no error). Handler tests can reuse `testField(mapId)` from `mystic_door_enter_test.go` (same package).
- `charconst.TemporaryStatTypePickPocket` = `"PICK_POCKET"` (`libs/atlas-constants/character/temporary_stat.go:33`); buff stat comparison is `ch.Type() == string(charconst.TemporaryStatTypePickPocket)` (same pattern as `IsMount`).
- `monster.ReflectInfo` fields: `Kind, Percent, LtX, LtY, RbX, RbY, MaxDamage, ExpiresAt`.

## Dependencies & Order

Tasks 1–4 are independent of Task 5 but all precede Task 6 conceptually; the plan order (1 → 2 → 3 → 4 → 5 → 6 → 7) is safe and each task compiles + passes tests on its own. Task 7 requires everything.

## Gotchas

- **Run all Go commands from the module dir** `services/atlas-channel/atlas.com/channel` (or via `(cd ...)`); the plan's commands do this. `docker buildx bake atlas-channel` and `tools/redis-key-guard.sh` run from the worktree root.
- **Do not prefix guard scripts with `GOWORK=off`** (known false-FAIL footgun).
- Handler tests are `package handler` (internal); drop tests are `package drop_test` (external). Match the existing files.
- No `*_testhelpers.go` files — small helpers like `ppTestBuff` live inside the `_test.go` file itself.
- atlas-drops, atlas-buffs, atlas-data: **zero changes**. If something looks like it needs a consumer-side change, stop and re-read design §2 — the meso-only SPAWN path is already exercised in production by atlas-monster-death and atlas-saga-orchestrator.
- Line numbers cited in the plan are pre-change positions in `character_attack_common.go`; they shift as tasks land. Anchor on the quoted code, not the numbers.
- Kafka emission goes through the service-local `atlas-channel/kafka/producer` `ProviderImpl` (tenant headers from ctx) — same as every existing producer call; no new tenant plumbing.

## Verification Gate (Task 7 / PRD §10)

```
(cd services/atlas-channel/atlas.com/channel && go test -race ./... && go vet ./... && go build ./...)
tools/redis-key-guard.sh
docker buildx bake atlas-channel
```

All must be clean before invoking code review / finishing-a-development-branch.
