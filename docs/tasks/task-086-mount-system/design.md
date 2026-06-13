# Mount / Monster-Rider System — Design

Task: task-086-mount-system
Status: Approved for planning
PRD: `docs/tasks/task-086-mount-system/prd.md`
Created: 2026-06-12

---

## 1. Summary of decisions

This design implements the tamed-monster and skill-only mount families end-to-end, grounded
in code reconnaissance of the existing Atlas patterns and **IDA verification of the v83 client**
(GMS v83, `MapleStory_dump.exe`, port 13337). The three architectural choices made before
writing this document:

1. **A new `atlas-mounts` service owns persistent mount state, the active-mount registry,
   and the tiredness ticker** — modeled directly on `atlas-pets` (which already owns pet
   persistence + a logged-in-character registry + a 3-minute `HungerTask`, the exact analog of
   our tiredness tick). The mount *rendering* still rides the existing `MONSTER_RIDING`
   character-temporary-stat buff via `atlas-buffs`; `atlas-mounts` owns only the
   progression/lifecycle state and the `SET_TAMING_MOB_INFO` data.
2. **The Riding Mimiana skill-acquisition questline is included in this task** (FR-9),
   authored via the project's `convert-quest` / `convert-npc` conventions.
3. **The four wire-behavior questions were verified in IDA now**, not deferred. Results are
   baked into the requirements below (Section 2).

### 1.1 IDA-confirmed wire facts (v83, port 13337)

| Question | Verdict | Evidence |
|---|---|---|
| MonsterRiding temp-stat encoding | **`nOption` = vehicle/taming-mob item id (1st int), `rOption` = source skill id (2nd int), then expire time** | `TemporaryStatBase<long>::DecodeForClient` @0x793ef2; riding field `SecondaryStat+0xCBC` via `IsRidingSkillVehicle` @0x8c46ab |
| OQ 9.1 — saddle vs taming-mob | **BOTH required**: taming-mob in slot -18 AND saddle in slot -19; either empty blocks the cast. Plus a cash cover-match guard | `CUserLocal::DoActiveSkill` case 1004 @0x968b9d (`CharacterData+0x17B` = -18, `+0x183` = -19) |
| OQ 9.2 — re-cast | **Server-driven toggle.** Re-cast sends the *same* `CP_UserSkillUseRequest` (0x5B); the client never sends a dedicated dismount. The server decides mount-vs-dismount | `CUserLocal::DoActiveSkill_StatChange` @0x969e21, `SendSkillUseRequest` @0x96d399 |
| OQ 9.3 — mount food | **Dedicated client→server opcode 0x4D** (`SendTamingMobFoodItemUseRequest` @0xa09a64), gated on item category **226** + active ride skill 190x. Body: `ts(4), slot(2), itemId(4)`. NOT the generic item-use path | `CDraggableItem::OnDoubleClicked` @0x4efd25 |
| `SET_TAMING_MOB_INFO` (s→c) field order | `characterId(4), level(4), exp(4), tiredness(4), levelUp(1 byte)` | `CWvsContext::OnSetTamingMobInfo` @0xa29115 |

These supersede the corresponding PRD open questions. Requirements below are written against
these confirmed facts.

---

## 2. Resolved requirements (overriding PRD open questions)

- **FR-1.3 / OQ 9.1 → BOTH equips required.** A tamed-mount cast (`skillId % 10000000 == 1004`)
  requires a taming-mob item in slot -18 **and** a saddle in slot -19. Missing either is a
  silent no-op (re-enable client, no buff). The cash cover-match guard is a client-side UX
  check and is out of server scope.
- **FR-3.5 / FR-4.1 / OQ 9.2 → server toggle.** The mount skill-use packet is the *same*
  packet whether mounting or dismounting. The channel determines current mount state and
  routes: not mounted → validate + `Apply`; already mounted on that skill → `Cancel`
  (dismount). The client does not distinguish; the server owns the toggle.
- **FR-8 / OQ 9.3 → dedicated opcode 0x4D.** Feeding is a **new inbound handler** for the
  taming-mob-food opcode (per-tenant handler opcode, value 0x4D in v83), not the standard
  use-item handler. Revitalizers are **item classification 226** (`itemId / 10000 == 226`,
  ids 2260000–2269999).
- **OQ 9.4 (exp table, cap 31) and OQ 9.6 (full skill-id/vehicle-id set)** are *game-data*
  values, not wire facts. Per the project's verify-over-memory rule they are sourced from
  local skill/WZ data and the reference-server `getMountExpNeededForLevel` during the plan
  phase — not from memory. This design specifies the *mechanism*; the plan pins the values.
- **OQ 9.7** — the new inbound (0x4D food) and outbound (`SET_TAMING_MOB_INFO`) opcodes must be
  patched into **live** tenant configs, not only seed templates (known pitfall). Captured as a
  deployment task.

---

## 3. Architecture overview

### 3.1 Service responsibilities

| Service / lib | Role in this feature |
|---|---|
| **`atlas-mounts` (NEW)** | Owns `character_mounts` persistence (level/exp/tiredness). Holds the active-mount registry + 60s tiredness `TirednessTask`. Consumes buff `APPLIED`/`EXPIRED` (MONSTER_RIDING) and character login/logout to maintain the registry. Consumes a "taming-mob fed" event to apply heal→exp→level. Emits a `mount status` event that drives `SET_TAMING_MOB_INFO`. |
| **`atlas-buffs`** | Unchanged lifecycle. The `MONSTER_RIDING` stat carries `amount = vehicle id`, `sourceId = skill id`. Existing `Apply`/`Cancel`/`CancelByStatTypes` reused; the APPLIED/EXPIRED events are now *also* consumed by atlas-mounts. |
| **`atlas-channel`** | Mount branch in the skill-use handler (toggle + prereq validation + vehicle-id resolution → `Apply`/`Cancel`). New inbound handler for food opcode 0x4D. New `SET_TAMING_MOB_INFO` writer + a consumer that broadcasts it. The existing buff give/foreign path renders the vehicle unchanged once the packet fix lands. |
| **`libs/atlas-packet`** | Fix `getBaseTemporaryStats` so the Monster Riding base stat encodes `nOption = amount`, `rOption = sourceId` (replacing the zeroed placeholder at `character_temporary_stat.go:720-721`). |
| **`atlas-data`** | Extend `skill/reader.go` (TODO @ line 227) so skill-only mount skills emit `MONSTER_RIDING` with the correct **vehicle id** as the statup amount. Extend the consumable reader to expose the revitalizer's tiredness-heal value. |
| **`atlas-consumables`** | New handler for the food command: validate classification-226 item + active mount context, decrement one from inventory (existing `ConsumeItem`), emit "taming-mob fed". The heal→exp math lives in atlas-mounts. |
| **`atlas-constants`** | Add the missing mount skill-id constants (SpaceShip 1013, Yeti 1017/1018, Broomstick 1019, Balrog 1031, plus Noblesse/Legend bands) and item classification 226 (revitalizer / taming-mob food). Slots -18/-19, classes 190/191, `TemporaryStatTypeMonsterRiding` already exist. |
| **`atlas-quest` / NPC data** | Author the Riding Mimiana questline (quest + NPC conversation) granting the Monster Rider skill + starter saddle/taming-mob via the existing reward path. |

### 3.2 Why a new service (vs. atlas-character)

`atlas-pets` is the precedent: pet level/closeness/fullness is a *companion-progression*
sub-domain with its own DB, its own logged-in registry, and a periodic hunger tick that emits
status events consumed by `atlas-channel`. Mount level/exp/tiredness is structurally identical
(progression + a periodic decay tick + status broadcast). Putting it in `atlas-character`
would either bloat the hot character row or graft a Kafka producer + task onto the REST-only
`saved_location` pattern. A dedicated `atlas-mounts` keeps each service's responsibility clean
and lets us copy `atlas-pets` almost verbatim.

### 3.3 End-to-end data flow

**Mount (tamed):**
```
client ──CP_UserSkillUseRequest(0x5B, skill%1e7==1004)──▶ atlas-channel skill-use handler
   ├─ not mounted: validate slots -18 + -19 → vehicleId = item@-18, sourceId = skillId
   │     └─▶ atlas-buffs Apply(MONSTER_RIDING, amount=vehicleId, sourceId=skillId, long-duration)
   └─ already mounted on this skill: ─▶ atlas-buffs Cancel(sourceId=skillId)   [dismount]

atlas-buffs ──APPLIED(MONSTER_RIDING)──▶ atlas-channel buff consumer
        ──▶ CharacterBuffGive (self) + CharacterBuffGiveForeign (map)   [vehicle renders]
atlas-buffs ──APPLIED(MONSTER_RIDING)──▶ atlas-mounts
        ──▶ register active mount; load/create character_mounts; emit MountStatus(SET)
atlas-mounts ──MountStatus──▶ atlas-channel mount consumer ──▶ SET_TAMING_MOB_INFO (map)
```

**Mount (skill-only):** identical, except no slot validation and `vehicleId` comes from the
skill effect (atlas-data reader), and atlas-mounts does **not** register it for the tiredness
ticker (no progression).

**Tiredness tick (atlas-mounts, every 60s):**
```
TirednessTask ─▶ for each active tamed mount of a logged-in character:
     tiredness = min(99, tiredness+1); persist; if hit 99 → notice flag
     emit MountStatus(TICK) ─▶ atlas-channel ─▶ SET_TAMING_MOB_INFO (map)
```

**Dismount (recast / job change / ladder auto-cancel / logout):**
```
atlas-buffs ──EXPIRED(MONSTER_RIDING)──▶ atlas-channel ──▶ CharacterBuffCancel(+Foreign)  [vehicle removed]
atlas-buffs ──EXPIRED(MONSTER_RIDING)──▶ atlas-mounts ──▶ deregister active mount (state already persisted)
```

**Feed:**
```
client ──food opcode 0x4D {ts, slot, itemId}──▶ atlas-channel food handler
     ──▶ atlas-consumables "use taming-mob food" command {characterId, slot, itemId}
atlas-consumables: validate class 226 + decrement item ──▶ emit "TamingMobFed" {characterId, itemId, tirednessHeal}
atlas-mounts: heal = min(tirednessHeal, tiredness); tiredness -= heal;
     exp += ceil((heal / tirednessHeal) * (2*level + 6)); level-up to cap; persist;
     emit MountStatus(FEED, levelUp) ──▶ atlas-channel ──▶ SET_TAMING_MOB_INFO (map)
```

---

## 4. atlas-mounts service detail

Directory: `services/atlas-mounts/atlas.com/mounts` — module name `atlas-mounts` (short),
mirroring `atlas-pets`.

### 4.1 Persistence (`mount/` package)

`character_mounts` entity (GORM), table `character_mounts`:

| Column | Type | Notes |
|---|---|---|
| `tenant_id` | uuid, not null | uniqueIndex `idx_character_mount_lookup` priority 1 |
| `character_id` | uint32, not null | uniqueIndex priority 2 |
| `id` | uuid, pk | |
| `level` | int, not null, default 1 | cap 31 (OQ 9.4 — confirm during plan) |
| `exp` | int, not null, default 0 | |
| `tiredness` | int, not null, default 0 | clamp 0–99 |
| `last_tiredness_tick_at` | timestamp, nullable | tick accounting across restarts |

- Immutable `Model` + `Builder` + `Processor` (interface + impl, `NewProcessor(l, ctx, db)`),
  `administrator.go` with an upsert keyed on `(tenant_id, character_id)` — copied from
  `saved_location` / `pet`.
- Tenant scoping via `tenant.MustFromContext(ctx)` in the constructor.
- Default-on-first-read: a character with no row gets `level 1 / exp 0 / tiredness 0`.
- REST: JSON:API `GET /characters/{characterId}/mount` (read) and `PATCH`/internal update
  for tooling parity. `GetName()` → `"mounts"`. Not strictly required by the packet flow
  (events drive everything) but mirrors atlas-pets and aids debugging.

### 4.2 Active-mount registry + tiredness ticker

- **Registry**: Redis-backed `TenantRegistry[uint32, MountRideContext]` keyed by character id,
  storing `worldId` + `skillId` + `vehicleId` + `mounted-at`. Initialized in `main.go` (mirrors
  `pet.InitTemporalRegistry`). Routed through `libs/atlas-redis` (repo Redis invariant).
- **Population**: consume buff `APPLIED(MONSTER_RIDING)` → if the source skill is a *tamed*
  mount (1004 band) add to registry; `EXPIRED(MONSTER_RIDING)` → remove. Skill-only mounts are
  **not** added (FR-2.2: no tiredness progression).
- **Logged-in gating**: consume the character login/logout events (same topics atlas-pets
  consumes) so the ticker only touches online characters and deregisters on logout (FR-4.4).
- **`TirednessTask`** (`tasks/`): `Task{Run, SleepTime}` registered via `tasks.Register`,
  cadence 60s. `Run()` iterates the active tamed mounts of logged-in characters, increments
  tiredness (clamp 99), persists, and emits `MountStatus(TICK)`. At the 99 clamp it sets the
  `tooTired` notice flag in the emitted event so the channel can surface FR-6.3's message.
  One task iterating the registry — no per-character goroutines/timers (NFR performance).

### 4.3 Feed application

Consume `TamingMobFed {characterId, itemId, tirednessHeal}`:
```
heal     = min(tirednessHeal, currentTiredness)
tiredness = currentTiredness - heal
gained   = ceil( (heal / tirednessHeal) * (2*level + 6) )    # heal fraction → exp
exp      = exp + gained
while exp >= expNeededForLevel(level) and level < CAP: exp -= need; level++; levelUp = true
persist; emit MountStatus(FEED, levelUp)
```
`expNeededForLevel` and `CAP` come from local data / reference parity (OQ 9.4), pinned in the
plan. The whole apply is transactional w.r.t. the consumed message (NFR resilience) — persist
and emit inside one `message.Emit` buffer so a crash neither double-applies exp nor loses
tiredness.

### 4.4 Kafka surface (atlas-mounts)

- **Consumes**: `EVENT_TOPIC_CHARACTER_BUFF_STATUS` (filter MONSTER_RIDING APPLIED/EXPIRED);
  character login/logout topic(s); `TAMING_MOB_FOOD` event from atlas-consumables.
- **Emits**: `EVENT_TOPIC_MOUNT_STATUS` with `StatusEvent[Body]` shaped like the pet events —
  `{WorldId, CharacterId, Type ∈ {SET, TICK, FEED}, Body{Level, Exp, Tiredness, LevelUp, TooTired}}`.
  Keyed by character id.

---

## 5. atlas-channel detail

### 5.1 Mount branch in the skill-use handler

In `socket/handler/character_skill_use.go` → `skill/handler` dispatch, add a mount branch
keyed on `skillId % 10000000 == 1004` (tamed) and the skill-only mount skill ids:

1. Decorate the character load with the inventory decorator to read equip slots.
2. **Toggle decision**: determine current mount state from session-local buff state (the buff
   give/cancel consumer already tracks active buffs per session) — presence of a
   `MONSTER_RIDING` buff. If present → emit `Cancel(sourceId = skillId)` to atlas-buffs
   (dismount) and `enableActions`. Done.
3. **Mount path (not mounted)**:
   - *Tamed (1004 band)*: require non-empty slot -18 **and** -19 (FR-1.3 confirmed). If either
     empty → no-op + `enableActions`. `vehicleId = itemId@slot(-18)`, `sourceId = skillId`.
   - *Skill-only*: no equip check. `vehicleId =` the skill effect's `MONSTER_RIDING` statup
     amount (now produced correctly by atlas-data). `sourceId = skillId`.
   - Emit `Apply(MONSTER_RIDING, amount = vehicleId, sourceId = skillId, level, duration)`.
4. **Duration**: mounts persist until explicit dismount, so apply with an effectively-infinite
   duration sentinel rather than a finite skill duration (a finite duration would expire and
   surprise-dismount). The exact sentinel mirrors however existing toggle/aura buffs are
   applied — pinned in the plan.

Rendering needs no new channel code: once the packet fix (Section 6) lands, the existing
`CharacterBuffGive` / `CharacterBuffGiveForeign` emit the vehicle for self + observers.

### 5.2 Food opcode 0x4D inbound handler

New handler registered against the per-tenant taming-mob-food handler opcode. Decode
`ts(4), slot(2), itemId(4)` (matches `SendTamingMobFoodItemUseRequest`). Emit a "use
taming-mob food" command to atlas-consumables with `{characterId, slot, itemId}`. The handler
itself does no item mutation — consumption is atlas-consumables' job.

### 5.3 SET_TAMING_MOB_INFO writer + consumer

- **Writer** in `libs/atlas-packet` (+ registered in atlas-channel's writer set, opcode per
  tenant config): encode `characterId(4), level(4), exp(4), tiredness(4), levelUp(1)` — exact
  v83 field order confirmed @0xa29115.
- **Consumer** for `EVENT_TOPIC_MOUNT_STATUS`: resolve the live session by character id, get
  its field, broadcast `SET_TAMING_MOB_INFO` to the map (`ForSessionsInMap`). On `TooTired`,
  also send the FR-6.3 notice to the rider only.

---

## 6. Packet fix (`libs/atlas-packet`)

`getBaseTemporaryStats` (`model/character_temporary_stat.go:715-726`) currently appends a
zeroed `NewCharacterTemporaryStatBase(false)` for Monster Riding with a TODO. Replace it so the
Monster Riding base stat encodes the stored stat's `amount` as `nOption` and `sourceId` as
`rOption`:

- `AddStat(...)` already receives `(type, sourceId, amount, level, expiresAt)` — thread the
  MONSTER_RIDING `amount`/`sourceId` into the base-stat encoder instead of discarding them.
- Both the self path (`Encode`) and the observer path (`EncodeForeign`) append this block, so
  both must carry the real values.
- **Test**: byte-level encode test asserting, for a MONSTER_RIDING buff with `amount=1902000`,
  `sourceId=1004`, that the trailing base-stat block emits `nOption=1902000` then
  `rOption=1004` in that order, for both `Encode` and `EncodeForeign`. This is the
  acceptance-critical wire test.

---

## 7. atlas-data detail

- **`skill/reader.go:227`** — extend the mount branch so each skill-only mount skill emits the
  `MONSTER_RIDING` statup with the correct **vehicle id** as the amount:
  Yeti1→1932003, Yeti2→1932004, Broomstick→1932005, Balrog→1932010, SpaceShip→`1932000 + level`
  (per-level effect). The 1004-band *tamed* skills continue to emit the stat, but the channel
  overrides the amount with the equipped taming-mob id (the reader cannot know the per-character
  equip). The exact id set + Noblesse/Legend bands are pinned from local skill data in the plan
  (OQ 9.6) — not from memory.
- **Consumable reader** — expose the revitalizer's tiredness-heal value (the food item's spec)
  so atlas-consumables can pass `tirednessHeal` rather than hardcoding 30. If the WZ spec lacks
  it, fall back to the reference-parity value, documented in the plan (OQ 9.4).

---

## 8. atlas-consumables detail

- New "use taming-mob food" command handler: validate the item is classification 226 and the
  character has an active mount context, decrement one via the existing `ConsumeItem` path
  (emits the standard CONSUME event), then emit `TamingMobFed {characterId, itemId,
  tirednessHeal}`. The heal→exp→level math stays in atlas-mounts (mount domain).
- No change to the generic `ApplyItemEffects` flow — food is its own path because it arrives on
  a dedicated opcode and has mount-specific semantics.

---

## 9. atlas-constants detail

- **Skill ids** (`skill/constants.go`): add SpaceShip 1013, YetiMount1 1017, YetiMount2 1018,
  Broomstick 1019, Balrog 1031 in the beginner band, plus their Noblesse/Legend equivalents
  (verified against skill data in the plan). BeginnerMonsterRidingId 1004, Noblesse/Legend/Evan
  variants already exist.
- **Item classification 226** (revitalizer / taming-mob food): add to `item/constants.go`
  alongside TamedMob 190 / Saddle 191.
- A small helper `IsTamedMountSkill(skillId) = skillId % 10000000 == 1004` and a
  skill-only-mount predicate/set for the channel branch.

---

## 10. Questline (FR-9)

Author the Riding Mimiana skill-acquisition line via `convert-quest` / `convert-npc`:

- Quest definition(s) whose reward grants the Monster Rider skill (1004 in the appropriate
  band) plus a starter saddle (class 191) and taming-mob (class 190), using the existing
  atlas-quest → atlas-skills skill-grant + inventory item-grant reward paths.
- NPC conversation state machine following the project's JSON conventions.
- Skip Player-NPC spawning (project convention — not implemented). Quest ids, item ids, and NPC
  names sourced from script comments / local data, not memory.

This is the most data-driven, lowest-coupling slice and can be built/tested largely in parallel
with the mechanic.

---

## 11. Build-system & deploy changes (new service)

Mirror atlas-pets. Files to add `atlas-mounts` to:

- `.github/config/services.json` — new go-service entry (single source of truth).
- `go.work` — `./services/atlas-mounts/atlas.com/mounts`.
- `docker-bake.hcl` — `"atlas-mounts"` in `go_services` (alphabetized).
- `deploy/k8s/base/atlas-mounts.yaml` — new manifest (copy atlas-pets.yaml; `DB_NAME`,
  env, service name swapped). **No LB socket ports** — atlas-mounts is REST+Kafka only, so the
  per-version socket-port pitfall does not apply.
- Repo-root `Dockerfile` — **no edit** (parameterized by `ARG SERVICE`; no new shared lib).

`docker buildx bake atlas-mounts` is mandatory verification once the service exists, plus a
bake for every other service whose `go.mod` is touched (channel, buffs, data, consumables,
constants consumers).

### 11.1 Live-config deployment (OQ 9.7)

The new **inbound** food handler opcode (0x4D) and the new **outbound** `SET_TAMING_MOB_INFO`
writer opcode must be added to **live tenant configs**, not just seed templates — otherwise the
food packet is silently dropped and the info packet never sends. Captured as an explicit
post-deploy task with a channel restart (projection doesn't hot-reload handlers/writers).

---

## 12. Kafka topic / event catalog

| Topic (env var) | Producer | Consumer(s) | Shape |
|---|---|---|---|
| `COMMAND_TOPIC_CHARACTER_BUFF` (existing) | atlas-channel | atlas-buffs | `Apply`/`Cancel` MONSTER_RIDING |
| `EVENT_TOPIC_CHARACTER_BUFF_STATUS` (existing) | atlas-buffs | atlas-channel (render), **atlas-mounts (registry)** | APPLIED/EXPIRED |
| character login/logout (existing) | atlas-character | **atlas-mounts** | online gating |
| `COMMAND_TOPIC_TAMING_MOB_FOOD` (new) | atlas-channel | atlas-consumables | `{characterId, slot, itemId}` |
| `EVENT_TOPIC_TAMING_MOB_FOOD` (new) | atlas-consumables | atlas-mounts | `{characterId, itemId, tirednessHeal}` |
| `EVENT_TOPIC_MOUNT_STATUS` (new) | atlas-mounts | atlas-channel | `{Type∈SET/TICK/FEED, Level, Exp, Tiredness, LevelUp, TooTired}` |

---

## 13. Testing strategy

- **Packet (unit, acceptance-critical)**: byte-level encode test for the MONSTER_RIDING base
  stat (`nOption`=vehicle, `rOption`=skill) for both `Encode` and `EncodeForeign`; encode test
  for `SET_TAMING_MOB_INFO` field order.
- **atlas-mounts (unit)**: tiredness clamp at 99 + notice flag; feed heal→exp→level math
  including level-up-to-cap and the `levelUp` flag; default-on-first-read; persistence upsert
  scoping by tenant+character. Use the project Builder pattern for fixtures (no `*_testhelpers`).
- **Channel (unit)**: toggle decision (mounted→cancel, not→apply); tamed prereq (both slots);
  skill-only no-prereq + vehicle-id source; food opcode decode.
- **atlas-data (unit)**: skill reader emits correct vehicle id per skill-only mount + per-level
  SpaceShip.
- **Quest**: conversion validated via the convert-quest/convert-npc tooling.
- **Per-module gate**: `go test -race ./...`, `go vet ./...`, `go build ./...` clean; `docker
  buildx bake` for each touched service; `tools/redis-key-guard.sh` clean.

---

## 14. Risks & deferrals

- **Game-data values (OQ 9.4, 9.6)** — exp-to-level table, level cap (31), the full mount
  skill-id/vehicle-id set, and the per-item tiredness-heal value are pinned from local
  skill/WZ data and reference-server source **in the plan phase**, never from memory. If the
  WZ food spec lacks a heal value, the reference parity value is used and documented.
- **Indefinite-duration buff** — the mount buff must not expire on its own; the apply-duration
  sentinel is pinned against how existing toggle/aura buffs are applied.
- **Live opcode config (OQ 9.7)** — both new opcodes must reach live tenant configs + channel
  restart, or the feature silently no-ops.
- **Out of scope (unchanged):** Corsair Battleship (HP-gated, separate lifecycle); cash-shop
  mount flows beyond the questline; any atlas-ui work. Taking damage does **not** dismount
  (confirmed: no damage-driven dismount in the tamed path).

---

## 15. Acceptance criteria

Inherits the PRD Section 10 checklist, with these now-confirmed specifics:

- [ ] Tamed mount requires BOTH slot -18 and -19; missing either is a silent no-op.
- [ ] Re-cast of the mount skill dismounts (server toggle), not a re-apply error.
- [ ] Feeding arrives on the dedicated 0x4D handler (not generic item-use) and consumes a
      class-226 item.
- [ ] Self + foreign buff packets encode MONSTER_RIDING `nOption=vehicleId`, `rOption=skillId`
      (byte-level test passes; TODO removed).
- [ ] `SET_TAMING_MOB_INFO` encodes `characterId, level, exp, tiredness, levelUp` in order.
- [ ] Mount level/exp/tiredness persist across logout/login + channel change via atlas-mounts.
- [ ] Tiredness ticks every 60s, clamps at 99 with the notice, broadcasts via SET_TAMING_MOB_INFO.
- [ ] Skill-only mounts render from skill ownership alone, no equip, no tiredness.
- [ ] Riding Mimiana questline awards the skill + starter saddle/taming-mob.
- [ ] All changed Go modules pass test/vet/build + bake; redis-key-guard clean; Battleship untouched.
