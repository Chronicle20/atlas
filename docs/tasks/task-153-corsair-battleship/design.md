# Corsair Battleship (5221006) — Design

Task: task-153-corsair-battleship
Status: Approved PRD → this document covers architecture, alternatives, and tradeoffs.
Inputs: `docs/tasks/task-153-corsair-battleship/prd.md` (v1), code research against this worktree, IDA verification against v83/v84/v87/v95/JMS185 IDBs (2026-07-10).

---

## 1. Resolution of the PRD's open questions

All four design-time questions from PRD §9 are resolved with verified evidence.

### 1.1 The 5221999 gauge is client-verified on every version with an IDB

Verified 2026-07-10 by decompiling `CUserLocal::OnSkillCooltimeSet` in each IDB. Every
version carries the identical special case — when the decoded skill id equals 5221999, the
raw uint16 value is stored directly (ship HP) instead of being converted to a
`timeGetTime()`-relative expiry:

| Version | IDB | Evidence |
|---|---|---|
| GMS v83 | port 13342 | `OnSkillCooltimeSet` @ 0x95BEBB: `if (v2 == 5221999) v4 = v3;` |
| GMS v84 | port 13345 | equivalent fn @ 0x99A14F: `if (v1 == &loc_4FAE6F)` (0x4FAE6F = 5221999) |
| GMS v87 | port 13343 | `OnSkillCooltimeSet` @ 0x9DE5A0: same pattern |
| GMS v95 | port 13341 | `OnSkillCooltimeSet` @ 0x908C0F: same pattern |
| JMS v185 | port 13344 | `OnSkillCooltimeSet` @ 0xA274D4: same pattern |
| GMS v92 | — | **No IDB exists. Unverified**, but bracketed by verified v87 and v95; worst case the gauge is cosmetic there (break logic unaffected). |

Additional v83 findings that shape the design:

- `CWvsContext::SetSkillCooltimeOver` @ 0xA123B0: on 5221999, if
  `SecondaryStat::IsRidingSkillVehicle()`, the client updates the temporary-stat view with
  (remaining = packet value, max = client-computed). So the gauge only renders while the
  client believes it is riding — the server must apply the mount buff before gauge updates
  mean anything.
- The client's max-HP function (sub_7665F1, v83) is `200 * (charLevel + 2*skillLevel − 120)`
  for skill 5221006 — algebraically identical to Cosmic's
  `400*skillLevel + (charLevel−120)*200` whenever charLevel ≥ 120 (always true for a 4th-job
  Corsair; Cosmic's `max(...,0)` clamp only diverges below 120, which is unreachable). The
  server-side formula in FR-2.2 is therefore consistent with what the client will draw.
- v95 `CWvsContext::RemoveSkillCooltimeOver` @ 0x9CCF80 resets the gauge temp view when
  5221999 is removed — no server action needed; dismount already clears it via riding state.

### 1.2 Cooldown transport: the existing `SET_COOLDOWN` command — no new surface

atlas-skills already exposes exactly what FR-4.2 needs:

- Kafka command `SET_COOLDOWN` on `COMMAND_TOPIC_SKILL`
  (`services/atlas-skills/atlas.com/skills/kafka/message/skill/kafka.go:16`,
  handler `kafka/consumer/skill/consumer.go:74-82`) → `SetCooldownAndEmit`
  (`skill/processor.go:212-220`) → Redis cooldown registry
  (`skill/cooldown_registry.go:39-48`) → `COOLDOWN_APPLIED` status event.
- atlas-channel already **produces** this command:
  `character/skill/processor.go:45-49` (`ApplyCooldown`) via
  `data/skill/producer.go:11-22`. The existing consumer
  `kafka/consumer/skill/consumer.go:111-127` (`handleCooldownApplied`) then announces the
  client packet. **Zero new topics, commands, or consumers.**

Two facts discovered here that become work items:

1. **The cast-time carve-out (FR-2.3) is a real code change, not a verification.**
   `skill/handler/common.go:93-95` applies `e.Cooldown()` on *every* cast **before** the
   mount short-circuit at :99-105. Once 5221006 is classified as a mount, its WZ
   `cooltime=90` would be applied on cast — exactly the bug the carve-out forbids. The
   generic block must exempt 5221006.
2. **No cast-time cooldown rejection exists anywhere** (FR-2.4 is new logic). The good news:
   `character_skill_use.go:42-43` already loads the character with `SkillModelDecorator`,
   whose skill models carry `CooldownExpiresAt` decorated live from atlas-skills Redis — the
   rejection check costs **zero extra round-trips**.

### 1.3 Riding-state source: a channel-local in-memory ride mirror

Decision: track "character is riding the battleship" in a small in-memory registry inside
atlas-channel, fed by the buff status events the channel already consumes. See §3.1 for the
alternatives analysis. Neither hot path (damage, attack) gains any network call for the
riding *check*; only riders pay one Redis round-trip for the HP drain itself, satisfying
FR-6.2 and NFR "hot path".

### 1.4 atlas-data emission: the PRD's assumption was wrong, but no atlas-data change is needed

`services/atlas-data/atlas.com/data/skill/reader.go:467-471` places `CorsairBattleshipId`
in the *tamed/placeholder* band: the MONSTER_RIDING statup amount is the **skill id
(5221006), not 1932000**. The doc comment at reader.go:462-465 says this is intentional:
"Tamed mounts (and Battleship) emit the skill id as a placeholder (the channel overrides
it)." So the ingestion layer already anticipates a channel-side vehicle override for
battleship, symmetric with how tamed mounts override with the equipped taming-mob item
(`skill/handler/mount.go:61-76`).

Consequences:

- **atlas-data: no code change** (PRD §7's "verify it emits 1932000, or fix" resolves to
  "it emits a placeholder by design; the channel overrides"). Not changing the reader also
  avoids per-tenant WZ re-ingestion, which a reader change would force on every environment.
- **atlas-constants: classification only, no vehicle id.** 5221006 must NOT be added to
  `SkillOnlyMountVehicleId` (that function is consumed by the atlas-data reader to bake
  fixed vehicle ids into ingested data — adding battleship there would both require
  re-ingestion and hardcode the wire value in violation of FR-7.2). Instead a new
  classification predicate routes 5221006 into `HandleMount` (§4.1).
- The vehicle id 1932000 is injected at buff-apply time in atlas-channel, resolved from
  tenant configuration (§4.6). There is currently no 1932000 constant anywhere in
  `libs/atlas-constants` — and none is added, per DOM-25.

---

## 2. Architecture overview

Everything new lives in atlas-channel plus two small shared-lib additions. No new services,
topics, REST endpoints, or DB tables.

```
                         cast 5221006 (character_skill_use.go)
                              │  cooldown-rejection gate (skill model already loaded)
                              ▼
                    UseSkill (common.go)
                    ├─ generic cooldown block: EXEMPT 5221006      (carve-out, FR-2.3)
                    └─ mount gate → HandleMount battleship arm
                         ├─ toggle: mounted? → cancel buff (existing Case 1)
                         ├─ vehicle id ← tenant writer-options table   (DOM-25)
                         ├─ apply MONSTER_RIDING buff (amount = vehicle id)
                         └─ ship HP := formula → Redis counter (TTL = effect duration)

  atlas-buffs BUFF_APPLIED/EXPIRED ──► channel buff consumer ──► ride mirror (in-memory)
                                                    │                {skillLevel} per char
  session Destroy funnel ───────────────────────────┴──► mirror remove + Redis delete

  damage taken (character_damage.go)
      ├─ mirror: riding? ── no ──► (existing behavior only)
      └─ yes ► Redis DecrByIfExists(damage)
                ├─ missing ► lazy re-init (formula − damage)          (FR-3.3)
                ├─ newHp > 0 ► announce gauge: SkillCooldown(5221999*, hp)  (FR-3.4)
                └─ crossed 0 ► BREAK: delete state + mirror,
                               cancel buff (dismount + foreign broadcast),
                               ApplyCooldown(5221006, effect cooltime)      (FR-4)
                                   └─► atlas-skills SET_COOLDOWN → COOLDOWN_APPLIED
                                        └─► existing consumer → client cooldown packet

  attack 5221007/5221008 (processAttack)
      └─ mirror: riding? ── no ──► reject, debug log, no side effects   (FR-6)

  (* 5221999 resolved from tenant writer options, never hardcoded — DOM-25)
```

Component inventory:

| Unit | Location | Purpose | Depends on |
|---|---|---|---|
| Mount classification | `libs/atlas-constants/skill/mount.go` | route 5221006 into HandleMount | — |
| `TenantCounter` | `libs/atlas-redis` (new type) | atomic int with init/decr-if-exists/TTL/delete | go-redis |
| uint32 resolve helper | `libs/atlas-packet/resolve.go` | config-resolve non-byte wire values | writer options |
| Ride mirror | atlas-channel `battleship` pkg (new) | in-memory riding truth + skill level | buff consumer, session destroy |
| Ship-HP registry | atlas-channel `battleship` pkg | Redis-backed HP pool behind a processor interface | TenantCounter |
| Battleship processor | atlas-channel `battleship` pkg | Mount / Drain / Break / Clear verbs; sole owner of mirror+registry | above + buff/skill processors |
| Mount arm | `skill/handler/mount.go` | battleship case: vehicle override + HP init | battleship processor, tenant config |
| Damage drain | `socket/handler/character_damage.go` | replaces the `// TODO` at :31 | battleship processor |
| Attack gate | `socket/handler/character_attack_common.go` | reject 5221007/5221008 on foot | ride mirror |
| Config tables | `services/atlas-configurations/seed-data/templates/*` + live backfill | 5221999 / 1932000 per tenant | — |

---

## 3. Alternatives considered

### 3.1 Riding-state source (PRD open question 3)

| Option | Damage-path cost | Gate cost | FR-3.3 (lazy re-init) | Verdict |
|---|---|---|---|---|
| **A. In-memory ride mirror** (chosen) | O(1) map read | O(1) | ✅ mirror is independent of the HP entry | **Chosen** |
| B. REST to atlas-buffs per check (existing `isMounted` pattern) | 1 HTTP call per damage packet for **every** player, riding or not | 1 HTTP per attack | ✅ | Rejected: unacceptable hot-path cost; violates NFR "no new synchronous cross-service REST calls in either path" |
| C. Redis HP-entry existence as the riding signal | 1 Redis op | 1 Redis op | ❌ impossible — a lost entry is indistinguishable from "not riding", so lost state silently disables the drain and wrongly rejects Cannon/Torpedo while visibly riding | Rejected: contradicts FR-3.3 and produces the worst failure mode |

Why the mirror is sound despite being pod-local and event-fed:

- A character's socket session lives on exactly one channel pod; the damage and attack
  packets we gate arrive only on that pod.
- Riding state cannot outlive that pod's knowledge: mounts are applied with
  `MountBuffDuration = MaxInt32` (`skill/handler/mount.go:22`) and are cancelled at next
  login by the session consumer ("mounts are transient",
  `kafka/consumer/session/consumer.go:290-303`). A pod restart kills the TCP session; the
  relog cancels any lingering mount buff. So there is no scenario where a player is
  legitimately riding on a pod whose mirror never saw the BUFF_APPLIED event.
- Precedent: `monster.StatusMirror` (atlas-channel `monster/status_mirror.go`) exists for
  exactly this reason — hot-path math without a network round-trip per damage entry.
- The cast→BUFF_APPLIED asynchrony leaves a few-ms window where the rider is not yet in the
  mirror. Damage in that window doesn't drain (ship briefly "invulnerable" — matches the
  reset-semantics spirit); an instant Cannon cast in that window is rejected once (client
  retries next attack). Both are harmless and self-heal.

### 3.2 Atomic drain + exactly-once break (NFR concurrency)

| Option | Loses concurrent drains? | Exactly-once crossing? | Verdict |
|---|---|---|---|
| **A. Lua `DecrByIfExists` in libs/atlas-redis** (chosen) | Never — DECRBY serializes | ✅ `new ≤ 0 && new+delta > 0` is true for exactly one caller per depletion | **Chosen** |
| B. `TenantRegistry.Update` (WATCH/MULTI CAS) | Yes — single-attempt CAS (`tenant_registry.go:130-161` has no retry loop); the losing concurrent drain errors out | needs a claimed-flag encoded in the value | Rejected: violates FR-3.2 "must not lose concurrent drains" without adding retry loops and a struct value |
| C. Raw `client.DecrBy` via the registry's `.Client()` escape hatch | Never | ✅ same predicate | Rejected: `DecrBy` isn't in redis-key-guard's banned list *today*, but bypassing the lib defeats the namespacing/tenancy the guard exists for; the convention (FR-1.5) is to add the primitive to `libs/atlas-redis` |

The lib has exactly one Lua script today (`lock.go:15-20`, atomic compare-and-delete) —
established precedent for adding a second.

Bare `DECRBY` on a missing key silently creates it at `-delta`, which would turn "state
lost" into "instant break" — the opposite of FR-3.3. Hence *decrement-if-exists*: the script
returns a `existed=false` signal instead of creating the key, and the caller takes the lazy
re-init path.

### 3.3 Where the config-resolved values live (FR-7.2 / DOM-25)

The two wire values ride different transports, so they get different (but both
writer-options-based) homes; a new atlas-tenants configuration resource was considered and
rejected as overkill for two integers.

- **5221999 (gauge pseudo-skill id)** — flows through the `CharacterSkillCooldown` writer.
  Natural home: an options table **on that writer's template entry**, resolved at encode
  time exactly like dispatcher mode bytes. One wrinkle: `WithResolvedCode`/`ResolveCode`
  (`libs/atlas-packet/resolve.go:13-60`) return a **byte**; 5221999 needs 4 bytes. A
  sibling helper (`ResolveValue`/`WithResolvedValue` returning uint32, same miss-handling
  semantics: loud error log) is added to `libs/atlas-packet` — mechanical, reusable.
- **1932000 (vehicle id)** — is NOT written by any specific writer; it travels as the
  MONSTER_RIDING statup amount through atlas-buffs and back, encoded generically by the CTS
  model (`libs/atlas-packet/model/character_temporary_stat.go:833-840`, nOption=amount).
  The only correct injection point is buff-apply time in `HandleMount`. atlas-channel
  already holds each tenant's `[]opcodes.WriterConfig` (it builds the writer producer from
  it, main.go:602-606); the design adds a tenant-scoped accessor for a named writer's
  options table so the mount arm can resolve `vehicles.CORSAIR_BATTLESHIP` from the
  `CharacterBuffGive` writer entry. On a resolve miss: log error and **abort the mount**
  (no buff, no HP state) — loud and safe, consistent with ResolveCode's fail-loud
  philosophy, rather than mounting with a placeholder that crashes the client.

Rejected alternative: hardcoding both values in `libs/atlas-constants` next to the Yeti /
Broomstick vehicle ids (task-086 precedent). That precedent predates DOM-25 enforcement
(task-102); FR-7.2 explicitly forbids it ("version-stable never exempts"), and the owner
has overruled the version-stable argument before (task-103).

---

## 4. Detailed design

### 4.1 libs/atlas-constants — classification only

`skill/mount.go` gains a third predicate (name illustrative):

- `IsBattleshipMountSkill(id Id) bool` — true only for `CorsairBattleshipId`.
- `IsTamedMountSkill` and `SkillOnlyMountVehicleId` are **unchanged** (§1.4: the latter is
  an atlas-data ingestion contract; battleship must not enter it).
- `mount_test.go` flips: 5221006 moves from "out of scope" (`mount_test.go:16,44`) to
  asserted-true for the new predicate and asserted-unchanged (false / (0,false)) for the
  two existing ones — pinning that battleship never leaks into the tamed or
  fixed-vehicle-ingestion bands.

The mount gate at `skill/handler/common.go:99-105` extends to
`IsTamedMountSkill || isSkillOnlyMount || IsBattleshipMountSkill`.

### 4.2 libs/atlas-redis — `TenantCounter`

New type, following the lib's existing construction/namespacing conventions
(`NewTenantRegistry` shape, tenant spliced into the key by the lib):

```go
type TenantCounter struct { ... } // namespace-scoped, tenant-keyed int64

NewTenantCounter(client *goredis.Client, namespace string, l logrus.FieldLogger) *TenantCounter

Set(ctx, t tenant.Model, key string, value int64, ttl time.Duration) error   // SET + EX (mount-time reset)
DecrByIfExists(ctx, t, key string, delta int64, ttl time.Duration) (newValue int64, existed bool, err error)
Remove(ctx, t, key string) error
```

`DecrByIfExists` is a Lua script (`EXISTS` → `DECRBY` + `PEXPIRE` refresh → return
`{new, 1}`, else `{0, 0}`), guaranteeing: no lost decrements, no key creation on the
missing-key path, TTL refreshed on every touch. Unit-tested in the lib (same test approach
as the existing lock-script tests).

### 4.3 atlas-channel — the `battleship` package

New package encapsulating **all** battleship state behind a processor interface (NFR "state
home caveat": handlers never see Redis or the map directly, so a future move to a
service-owned store only touches this package).

**Ride mirror** (in-memory, `sync.RWMutex` + `sync.Once` singleton per the project registry
pattern), keyed `(tenantId, characterId)` → `rideState{skillLevel byte}`:

- `put` on BUFF_APPLIED events whose changes include MONSTER_RIDING with
  `SourceId() == 5221006` and whose character has a session on this pod (same
  `IfPresentByCharacterId` guard the buff consumer already uses).
- `remove` on the matching BUFF_EXPIRED events (covers manual dismount toggle, server
  cancel, and theoretical expiry — atlas-buffs emits one event type for all three), and on
  the session `Destroy` funnel (covers logout, disconnect, timeout, and channel change —
  all call sites converge on `session/processor.go:330-348`).
- Map transitions do **not** touch the mirror: v83 keeps the rider mounted across maps, and
  FR-5.1 does not list map change as a ride end.
- Storing `skillLevel` (from the buff event's `Level()`) makes the break path and lazy
  re-init self-sufficient without a REST call for the skill book.

**Ship-HP registry**: `TenantCounter` with namespace `battleship-hp`, key = characterId.
Wiring mirrors atlas-mounts (`main.go:51-52`): `rc := atlas.Connect(l);
battleship.InitRegistry(rc)`. This is atlas-channel's first Redis dependency: `go.mod`
gains the `require` (a `replace` already exists at go.mod:82), and the channel deployment
manifests gain `REDIS_URL`/`REDIS_PASSWORD` env (copy from atlas-mounts). The shared
Dockerfile already COPYs `libs/atlas-redis`; `docker buildx bake atlas-channel` is still
mandatory verification.

**Processor verbs** (interface + Impl, `NewProcessor(l, ctx)`):

- `InitShipHP(characterId, skillLevel, charLevel)` — `Set(formula, ttl)`;
  `formula = 400*skillLevel + max(charLevel−120, 0)*200`; `ttl` = the effect's duration
  (35 min for 5221006 WZ data), converted from `e.Duration()` ms.
- `IsRiding(characterId) (bool, skillLevel)` — mirror read.
- `Drain(s session.Model, damage int32)` — the FR-3/FR-4 flow (§4.5).
- `Clear(characterId)` — mirror remove + Redis Remove; idempotent (remove-on-missing is a
  no-op, matching the atlas-mounts registry convention).

**TTL note (PRD FR-5.2 divergence, accepted):** the PRD assumed the buff duration (35 min)
bounds a ride, but mounts are applied with `MaxInt32` duration and never expire on their
own. The TTL therefore works as *idle* expiry: refreshed on every drain, so an active ship
never loses state mid-combat, while a ship untouched for 35 minutes lazily re-initializes
to full on the next hit — indistinguishable from the documented reset semantics, and
orphaned entries still self-expire (the FR-5.2 intent).

### 4.4 Mount cast path

In order, within the existing flow:

1. **Cast rejection (FR-2.4)** — in `character_skill_use.go`, after the existing
   skill-level validation (:60-70): if the skill is 5221006 and the already-loaded skill
   model's `CooldownExpiresAt()` is in the future → log debug, re-enable actions, return.
   Zero extra round-trips (§1.2). Scoped to battleship; a generic cast-time cooldown gate
   is explicitly out of scope (task-155 territory).
2. **Carve-out (FR-2.3)** — `common.go:93-95` becomes conditional: skip the generic
   `ApplyCooldown` when `IsBattleshipMountSkill(skillId)`, with a comment citing the
   break-only cooldown rule.
3. **Toggle (FR-2.5)** — no change: `HandleMount`'s existing Case 1 (`isMounted` →
   `cancelBuff`) already handles cast-while-riding for any mount skill; the resulting
   BUFF_EXPIRED event clears mirror + HP state through the §4.3 hooks.
4. **Battleship arm (FR-2.1/2.2)** — new case in `HandleMount`: resolve the vehicle id
   from tenant config (§4.6; on miss, log error + abort), override the effect's
   MONSTER_RIDING statup amount with it (symmetric to `tamedMountStatups`,
   mount.go:61-76), `applyBuff` with `MountBuffDuration` like the other mounts, then
   `InitShipHP(characterId, skillLevel, charLevel)`. Character level is not available
   inside `HandleMount` today (only the skill level is); it is threaded in from
   `character_skill_use.go`, where the loaded character model is already in scope — via
   the `mountDeps` seam so tests can inject it.

### 4.5 Damage drain, gauge, break

`character_damage.go` — the `// TODO decrease battleship hp` at :31 is replaced by a single
call `battleship.NewProcessor(l, ctx).Drain(s, p.Damage())` placed alongside the existing
`ChangeHP` (:43), which stays untouched (parallel pool, FR-3.1). `Drain` logic:

```
if damage <= 0 or !IsRiding(characterId): return          // O(1), non-riders exit here
newHp, existed := counter.DecrByIfExists(charId, damage, ttl)
if !existed:                                              // FR-3.3 lazy re-init (rare)
    charLevel ← character REST GetById (only on this path)
    full := formula(mirror.skillLevel, charLevel)
    newHp = full − damage
    if newHp > 0: counter.Set(newHp, ttl)                 // log debug "lazy re-init"
if newHp > 0:                                             // FR-3.4 gauge
    announce CharacterSkillCooldownWriter to s:
        (resolved BATTLESHIP_HP_GAUGE id, uint16(newHp))
else if newHp + damage > 0:                               // crossed zero: exactly-once break (FR-4)
    Clear(characterId)                                    // mirror + Redis, immediately
    buff.Cancel(s.Field(), characterId, 5221006)          // dismount + foreign broadcast
    cooltime := GetEffect(5221006, mirror.skillLevel).Cooldown()   // from data, not hardcoded
    skill.ApplyCooldown(s.Field(), 5221006, cooltime)     // → SET_COOLDOWN → client packet
// else: concurrent drain after the crossing — no-op
```

- The gauge is a direct session announce (the handler already holds `s` and `wp`; same
  `session.Announce` shape as the foreign-damage broadcast at :38). Ship HP fits uint16:
  the formula maxes at 28 000 (skill 30, level 200); clamp defensively anyway.
- **Exactly-once:** the crossing predicate is true for exactly one DECRBY per depletion
  (Redis serializes the script). Every downstream step is also idempotent — `Clear` is a
  no-op on missing state, atlas-buffs `Cancel` of an absent buff emits nothing, and a
  duplicate `SET_COOLDOWN` would re-apply the same expiry — so even the theoretical
  re-init-window double-break degrades to a no-op, satisfying the NFR without locks.
- Break intentionally sends no `(5221999, 0)` packet — Cosmic parity: the dismount itself
  stops the client rendering the gauge (`IsRidingSkillVehicle` gates it, §1.1).
- **Degraded mode:** any Redis error → log warn, skip the drain, never fail damage
  processing (ship is temporarily undrainable during a Redis outage; character HP flow
  unaffected).
- Drain applies to every positive damage amount, mirroring the unconditional `ChangeHP`
  the handler already performs (Cosmic's TakeDamageHandler does the same).

### 4.6 Cannon / Torpedo gate

In `processAttack` (`character_attack_common.go`), immediately after the existing
skill-ownership rejection (:287-290): if the attack's skill id is 5221007 or 5221008 and
`IsRiding` is false → log debug with character/tenant fields and return nil (no costs, no
damage application, no broadcast). Soft rejection matches the mobCount-cap precedent
(`common.go:141-152`) rather than the session-destroy sanction — a desynced legitimate
client (e.g. the cast/BUFF_APPLIED window) must not be disconnected. Pure map read: zero
I/O in the attack hot path (FR-6.2). The gate covers all attack entry points (melee /
ranged / magic / energy) since they all funnel through `processAttack`.

### 4.7 Config resolution and backfill (FR-7.2)

Template changes, all six versions (`template_gms_83_1.json`, `_gms_84_`, `_gms_87_`,
`_gms_92_`, `_gms_95_`, `_jms_185_`):

- `CharacterSkillCooldown` writer entry gains
  `"options": {"skills": {"BATTLESHIP_HP_GAUGE": 5221999}}`.
- `CharacterBuffGive` writer entry gains
  `"options": {"vehicles": {"CORSAIR_BATTLESHIP": 1932000}}`.
- Both values are version-stable across the five verified clients (§1.1: same literal in
  every IDB), so the tables are identical per version — but they remain config, per DOM-25.

Verification item folded into the plan: confirm the `CharacterSkillCooldown` and
`CharacterBuffGive` writers are actually routed (opcode present) in all six templates —
the v87 template has a history of missing writer wirings; any gap found is fixed in-scope.

Code side:

- `libs/atlas-packet/resolve.go`: `ResolveValue`/`WithResolvedValue` (uint32 variant of
  the existing byte resolver, same fail-loud miss logging), used by the gauge announce.
- atlas-channel: tenant-scoped accessor exposing a named writer's options table from the
  already-loaded socket configuration (used by the mount arm for the vehicle id, §3.3).

**Live-tenant backfill (ops step, documented in the task):** seed templates apply only at
tenant creation. Each provisioned tenant's channel socket configuration must be PATCHed to
add the two options tables (and any missing writer routings), then atlas-channel restarted
— handlers/writers do not hot-reload (known gotcha).

### 4.8 Services with no code change

- **atlas-skills** — consumed via existing `SET_COOLDOWN`. Note: `SetCooldown` requires the
  character to own a persisted skill row for 5221006 (processor.go:199) — always true for
  the caster who just broke their own ship.
- **atlas-buffs** — existing apply/cancel paths; cancel already emits the foreign broadcast
  via the channel's buff consumer.
- **atlas-data** — placeholder emission retained (§1.4).
- **atlas-mounts** — its buff consumer's non-tamed branch already ignores skill-only
  mounts (log only); battleship follows the same branch. No tiredness, no registry entry.

---

## 5. Testing strategy

Unit tests (project Builder pattern; the existing `mountDeps` / damage-handler seams —
no `*_testhelpers.go` files):

- **atlas-constants:** predicate flips in `mount_test.go` (battleship in the new predicate,
  still excluded from tamed/fixed-vehicle).
- **atlas-redis:** `TenantCounter` — set/TTL, decr-returns-new-value, decr-on-missing
  returns `existed=false` without creating the key, TTL refresh on decr, concurrent decrs
  lose nothing and exactly one caller observes the crossing.
- **atlas-packet:** `ResolveValue` hit/miss/malformed-table cases.
- **atlas-channel:**
  - formula: boundary cases (skill 1/30, level 120/121/200; sub-120 clamp).
  - mount arm: HP init on cast, vehicle-id override present in applied statups, abort on
    resolve miss, toggle dismount clears state.
  - carve-out: casting 5221006 emits **no** `SET_COOLDOWN`; a control skill with cooltime
    still does. Cast-while-cooling rejected.
  - drain: non-rider no-op; drain + gauge announce (asserting the resolved pseudo-id and
    uint16 HP); lazy re-init; break exactly-once under simulated concurrent drains
    (cancel + cooldown emitted once); Redis-error degraded path.
  - gate: 5221007/5221008 rejected on foot (no emissions), pass while riding; other
    skills unaffected.
  - mirror: apply/expire/session-destroy lifecycle.

Verification (CLAUDE.md): `go test -race ./...`, `go vet ./...`, `go build ./...` in every
changed module (`libs/atlas-constants`, `libs/atlas-redis`, `libs/atlas-packet`,
`services/atlas-channel`); `docker buildx bake atlas-channel`; `tools/redis-key-guard.sh`
clean from repo root. Runtime verification on a live tenant: mount/dismount visuals (self +
foreign), gauge movement under damage, break → dismount + 90 s cooldown + grey icon,
remount rejection while cooling, Cannon/Torpedo on foot rejected.

---

## 6. Accepted divergences and risks

| Item | Position |
|---|---|
| Ship HP resets on remount (vs Cosmic's persistence) | PRD interview Q3 decision; unchanged. |
| v92 gauge unverified | No IDB exists; bracketed by verified v87/v95. Worst case: cosmetic no-gauge on v92; break/cooldown logic is version-independent. Documented, not blocking. |
| FR-5.2 TTL basis | PRD assumed the buff expires at 35 min; the mount buff is actually `MaxInt32`. TTL becomes idle-expiry refreshed per drain (§4.3) — preserves the safety-net intent; a >35-min-idle ship re-initializes full, which the reset semantics already permit. |
| PRD §7 atlas-data row ("confirm it emits 1932000") | Factually wrong — it emits the skill-id placeholder *by design*; resolved with no atlas-data change (§1.4). |
| Cast→BUFF_APPLIED mirror window | Few-ms desync window: no drain / one rejected attack; self-heals. Accepted (§3.1). |
| Redis outage | Drain disabled (warn), everything else functions; state self-repairs via lazy re-init after recovery. |
| Mirror memory | One tiny struct per *currently riding corsair* per pod — negligible. |
