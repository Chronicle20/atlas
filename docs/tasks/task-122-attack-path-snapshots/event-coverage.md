# task-122 — Event-Coverage Audit (FR-2)

Status: design-phase audit, v1
Method: source reading of owning-service producers and atlas-channel consumers. Every claim cites file:line. Items marked **VERIFY-AT-EXECUTION** must be re-confirmed against the code as it stands when `/execute-task` runs (this audit was performed while task-120/121 were unmerged).

Component verdict legend (FR-2.2): **Covered** (rich event, update in place) / **Covered-but-thin** (invalidate-and-refetch) / **Gap** (no observable event → FR-2.4 disposition required).

Consumer-side note that applies to every component: all atlas-channel consumers use `SetStartOffset(kafka.LastOffset)` — there is no historical replay, so a snapshot registry starts empty at pod boot and MUST be seeded by REST (lazy populate), never by events alone. Event handlers must never *create* snapshot entries (task-120 discipline: update-only, no-op when entry absent).

---

## 1. Character core (Id, Level, JobId — attack read-set; full stat block for reuse)

Owner: `services/atlas-character`. All mutations flow through `atlas.com/character/character/processor.go` and emit on `EVENT_TOPIC_CHARACTER_STATUS`.

### Producer-side mutation inventory

| Mutation entry point (trigger) | Emits? | Event(s) → producer evidence |
|---|---|---|
| CHANGE_JOB cmd | yes | JOB_CHANGED + STAT_CHANGED(Job) — `character/processor.go:474-475` |
| CHANGE_HAIR / FACE / SKIN cmd (+ REST PATCH) | yes | STAT_CHANGED — `processor.go:507,539,571`; PATCH per-field events `processor.go:1671-1747` |
| AWARD_EXPERIENCE / DEDUCT_EXPERIENCE cmd | yes | EXPERIENCE_CHANGED + STAT_CHANGED — `processor.go:639-640,683-684` |
| AWARD_LEVEL cmd | yes | LEVEL_CHANGED + STAT_CHANGED(Level) — `processor.go:722-723` |
| Level-up growth (self-consumed LEVEL_CHANGED) | yes | STAT_CHANGED(AP/SP/HP/MaxHP/MP/MaxMP) **with Values** — `processor.go:1408` |
| REQUEST_CHANGE_MESO / DROP_MESO / meso pickup | yes | MESO_CHANGED / STAT_CHANGED(Meso) — `processor.go:750-751,796,768` |
| REQUEST_CHANGE_FAME cmd | yes | FAME_CHANGED + STAT_CHANGED(Fame) — `processor.go:812-813` |
| REQUEST_DISTRIBUTE_AP cmd | yes | STAT_CHANGED **with Values** — `processor.go:905` |
| REQUEST_DISTRIBUTE_SP cmd | yes | STAT_CHANGED(AvailableSP) + skill events — `processor.go:931-937` |
| CHANGE_HP / CHANGE_MP / SET_HP / CLAMP_HP / CLAMP_MP cmd | yes | STAT_CHANGED(Hp/Mp) — `processor.go:1155,1237,1205,1270,1304` |
| RESET_STATS / REBALANCE_AP cmd | yes | STAT_CHANGED **with Values** — `processor.go:1623,1867,1918` |
| CREATE / DELETE character (saga, account-delete) | yes | CREATED / DELETED — `consumer/character/consumer.go:353`, `processor.go:357` |
| **Movement** (`COMMAND_TOPIC_CHARACTER_MOVEMENT`) | **NO EMIT** | `consumer.go:366` → `Move()` `processor.go:728-731` writes only the Redis temporal registry (`temporal_data.go:69`) |

### Consumer side (atlas-channel, `kafka/consumer/character/consumer.go`, message defs `kafka/message/character/kafka.go`)

| Event | Payload | Classification |
|---|---|---|
| STAT_CHANGED | `Updates []stat.Type`, `Values map[string]interface{}` (optional) — `kafka.go:107` | **Covered-but-thin today**: producer populates `Values` on only 4 paths (AP-distribute, ResetStats, RebalanceAP, level-up growth); nil everywhere else, and the existing channel handler ignores it and refetches (`consumer.go:90,100`) |
| LEVEL_CHANGED | `Current byte` (absolute) — `kafka.go:141` | Covered (rich) |
| EXPERIENCE_CHANGED | `Current uint32` (absolute) — `kafka.go:129` | Covered (rich) |
| MESO_CHANGED / FAME_CHANGED | delta only — `kafka.go:153,147` | Thin (delta; not idempotent to apply) |
| MAP_CHANGED | full target field + `UseTargetPosition, TargetX, TargetY` — `kafka.go:114` | Covered (rich) |
| JOB_CHANGED | constant defined, **no channel handler** (`kafka.go:74`); job also arrives via STAT_CHANGED(Job) | Thin via STAT_CHANGED |

### Verdict: **Covered after one bounded producer change** (FR-2.4(a))

Disposition: extend atlas-character to populate `Values` (absolute post-mutation values, same key convention as the four already-populated sites) on **every** STAT_CHANGED emission. The field already exists in the schema (`omitempty`) — additive, backward-compatible, single service. Without this, every MP-consuming skill swing self-invalidates the snapshot (attack → CHANGE_MP → STAT_CHANGED(Mp, nil) → invalidate → refetch), defeating the zero-REST goal.

Snapshot handling: STAT_CHANGED with complete `Values` → update in place (idempotent, absolute). STAT_CHANGED without values (old events in flight during rollout, or any missed site) → invalidate core component. MESO/FAME deltas → invalidate core (attack path unaffected; correctness preserved for reuse). LEVEL_CHANGED / EXPERIENCE_CHANGED → in place.

---

## 2. Position (FR-2.5 — feeds the reflect bounds check)

### Producer-side finding: position is a **Gap** in event space — by construction

- atlas-channel folds each movement packet to a final `summary{X,Y,Fh,Stance}` locally (`services/atlas-channel/.../movement/processor.go:46-59`, folder `:231`) and emits `COMMAND_TOPIC_CHARACTER_MOVEMENT`.
- atlas-character's `Move()` writes X/Y/Stance to a Redis temporal registry only (`processor.go:728-731`) — **no status event, no DB row**. The atlas-maps location entity stores World/Channel/Map/Instance only (`atlas-maps/.../location/entity.go:18-24`).
- Therefore today's REST read of `c.X()/c.Y()` is a projection of atlas-channel's *own* movement fold, round-tripped through Kafka + Redis + nginx.

### Disposition (FR-2.4, resolves PRD Open Question 1)

Source position **locally**: the same movement-fold summary that atlas-channel already computes writes the snapshot's position component synchronously (before the Kafka emit). This is strictly fresher than the REST projection — zero hops instead of three. Supplemented by MAP_CHANGED (`TargetX/TargetY` when `UseTargetPosition`; otherwise position is marked invalid until the first movement packet or a REST fallback). REST fallback (base character GET, which reads the temporal registry) covers reads before first movement on a map. This meets FR-2.5's bar ("no worse than today's async projection") with margin.

Secondary gap (accepted): atlas-maps' forced-return on LOGOUT persists a different map with no MAP_CHANGED (`atlas-maps/.../consumer/character/consumer.go:110-127`, Set at `:125`) — irrelevant here because the snapshot is evicted with the session at logout.

---

## 3. Skills (owned skill id + level; masterLevel/expiration for reuse)

Owner: `services/atlas-skills`. Events on `EVENT_TOPIC_SKILL_STATUS`: CREATED, UPDATED, DELETED, COOLDOWN_APPLIED, COOLDOWN_EXPIRED (`kafka/message/skill/kafka.go:54-58`).

| Mutation entry point | Emits? | Evidence |
|---|---|---|
| REQUEST_CREATE (job advance, GM) | yes | CREATED — `processor.go:130` (consumer `consumer.go:54`) |
| REQUEST_UPDATE (SP assign) | yes | UPDATED — `processor.go:174` |
| SET_COOLDOWN / cooldown expiry ticker | yes | COOLDOWN_APPLIED `processor.go:205` / COOLDOWN_EXPIRED `processor.go:229` (`tasks/expiration.go:29`) |
| REQUEST_DELETE (saga compensation) | yes | DELETED — `processor.go:284` |
| char LOGOUT → `ClearAll` (cooldown registry) | NO EMIT | `processor.go:235-237` — transient cooldown state only |
| char DELETED → bulk `Delete` of skill rows | **NO EMIT** | `processor.go:240-244` (character consumer `consumer.go:65`) |

Consumer side (atlas-channel `kafka/consumer/skill/consumer.go`): CREATED/UPDATED carry `Level, MasterLevel, Expiration` (rich — `kafka/message/skill/kafka.go:36,42`), skillId in envelope; handlers at `consumer.go:65,83,111,143`.

### Verdict: **Covered**

CREATED/UPDATED → update in place. Add a snapshot handler for DELETED (event exists; channel has no handler today) → remove skill in place. The two NO-EMIT paths are neutralized by session-scoped eviction: character-DELETED and LOGOUT both imply the session is destroyed, which evicts the snapshot (FR-3.2). Cooldowns are not in the attack read-set; not stored in v1 of the snapshot.

---

## 4. Inventory / compartments / assets (weapon slot, consumable + cash assets)

Owner: `services/atlas-inventory`. Every command handler (`kafka/consumer/compartment/consumer.go:36-90`) calls an `*AndEmit` variant; asset primitives each emit (`asset/processor.go`: Delete→DELETED `:154`, Drop→DELETED `:184`, UpdateSlot→MOVED `:208`, UpdateQuantity→QUANTITY_CHANGED `:223`, UpdateEquipmentStats→UPDATED `:264`, ChangeTemplate→UPDATED `:291`, Create→CREATED `:375/394`, Accept→ACCEPTED `:429`, Release→RELEASED `:448`, Expire→EXPIRED `:169`).

| Mutation entry point | Emits? | Evidence |
|---|---|---|
| EQUIP / UNEQUIP (3-step slot swap) | yes | MOVED per step — `compartment/processor.go:470,477,484` |
| MOVE / MERGE / SORT | yes | MOVED / QUANTITY_CHANGED + MERGE_COMPLETE/SORT_COMPLETE — `processor.go:467-516,1427,1738` |
| DROP / DESTROY / EXPIRE | yes | DELETED / QUANTITY_CHANGED / EXPIRED — `processor.go:700-728,891,898,914+` |
| REQUEST_RESERVE / CANCEL_RESERVATION | yes | RESERVED `processor.go:767` / RESERVATION_CANCELLED `processor.go:801` (registry-only state) |
| CONSUME (reservation commit — projectile use) | yes | QUANTITY_CHANGED `processor.go:850` or DELETED `:842` (rechargeable→0 `:840`); no paired RESERVATION_CANCELLED |
| CREATE_ASSET (shops, pickup, quests, gachapon) / RECHARGE | yes | CREATED / QUANTITY_CHANGED |
| ACCEPT / RELEASE (trade, storage, cash shop) | yes | asset+compartment ACCEPTED / RELEASED — `processor.go:1518/1535,1609/1622` |
| INCREASE_CAPACITY | yes | CAPACITY_CHANGED — `processor.go:627` |
| char CREATED / DELETED | yes | inventory CREATED / DELETED — `consumer/character/consumer.go:48,60` |
| REST mutations | none exist | inventory REST is read-only |

Consumer side (atlas-channel `kafka/consumer/asset/consumer.go`, defs `kafka/message/asset/kafka.go`): envelope always carries `CharacterId, CompartmentId, AssetId, TemplateId, Slot` (`kafka.go:21`). CREATED/UPDATED/ACCEPTED carry the **full asset** including quantity and every equip stat (`kafka.go:31,65,111`) — rich. QUANTITY_CHANGED carries absolute quantity (`kafka.go:106`) — rich. MOVED carries `OldSlot` + new slot in envelope (`kafka.go:102`) — rich enough for an idempotent *set-slot-absolute* by AssetId (each leg of a swap gets its own MOVED). DELETED — removable by AssetId from envelope. RELEASED/EXPIRED — thin (`kafka.go:149,154`) → invalidate. Compartment CREATED/DELETED/CAPACITY_CHANGED defined (`kafka/message/compartment/kafka.go:134,139,142`) but unhandled today → add snapshot handlers (apply/invalidate).

### Verdict: **Covered**

In place: asset CREATED/UPDATED/ACCEPTED (full replace by AssetId), QUANTITY_CHANGED (absolute), MOVED (set slot absolute), DELETED (remove). Invalidate inventory component: RELEASED, EXPIRED, compartment DELETED/CAPACITY_CHANGED, MERGE_COMPLETE/SORT_COMPLETE (bulk rearrangements — invalidation is simpler and safer than replaying N MOVEDs against a possibly-mid-sequence registry).

Noted risks, with dispositions:
- **Reservations are registry-only in atlas-inventory and not reflected in asset rows.** VERIFY-AT-EXECUTION: whether `GET /characters/{id}/inventory` nets out reservations. Expected: it does not — in which case the snapshot ignoring RESERVED/RESERVATION_CANCELLED is exact REST parity (the attack planner sees the same quantities it sees today).
- **`RequestReserve` returns inside its request loop** (`processor.go:767`) — only the first batched reserve is processed or emitted. UNVERIFIED whether any caller batches >1. Not snapshot-relevant (reservations excluded), but flagged for the owner as an atlas-inventory bug candidate.
- CONSUME emits no RESERVATION_CANCELLED (quantity events are the release signal) — irrelevant to the snapshot for the same reason.

---

## 5. Buffs (projectile gate: SOUL_ARROW, SHADOW_PARTNER)

Owner: `services/atlas-buffs` (in-memory registry, no DB). Events APPLIED/EXPIRED on `EVENT_TOPIC_CHARACTER_BUFF_STATUS`.

| Mutation entry point | Emits? | Evidence |
|---|---|---|
| APPLY cmd | yes | APPLIED — `character/processor.go:59` |
| CANCEL / CANCEL_ALL / CANCEL_BY_TYPES cmd | yes | EXPIRED per buff — `processor.go:80,95,122` |
| Expiration ticker | yes | EXPIRED — `processor.go:136` (`tasks/expiration.go:27`) |
| Poison tick | n/a | mutates only its tick timestamp; HP damage routed as CHANGE_HP command (mirrored as STAT_CHANGED) — `processor.go:180` |

Consumer side (atlas-channel `kafka/consumer/buff/consumer.go:57,93`): APPLIED/EXPIRED carry `SourceId, Level, Duration, Changes []{Type, Amount}, CreatedAt, ExpiresAt` (`kafka/message/buff/kafka.go:58,68`) — **rich**.

### Verdict: **Covered** (resolves PRD Open Question 2 for buffs: join the snapshot)

APPLIED → add in place; EXPIRED → remove in place. Defense in depth: each snapshot buff entry carries `ExpiresAt` from the event, and reads self-filter entries past their expiry — so a lost EXPIRED event degrades to at most the buff's natural duration.

Accepted residual risk (documented, owner-visible): atlas-buffs holds buffs in memory only; an atlas-buffs pod restart silently drops all buffs without EXPIRED events. The snapshot would keep them until their `ExpiresAt`; today's REST read would return empty immediately. Neither matches the client (which also still displays the buff). Blast radius: projectile-consumption divergence for the buff's remaining duration in the rare restart window. No cheap mitigation exists without persisting buffs (out of scope); the self-expiry bound caps it.

---

## 6. Effective stats (venom DPT: Luck, MagicAttack)

Owner: `services/atlas-effective-stats` — a Redis-projected aggregation (`character/registry.go:18-33`) with lazy on-demand init (`processor.go:53-64`). **It emits no status event of any kind**; its only Kafka output is CLAMP_HP/CLAMP_MP commands (`character/producer.go:13,28`).

### Verdict: **Gap → stays on REST** (FR-2.4(b); resolves PRD Open Question 2 for effective stats)

No event exists to maintain a snapshot component, and re-aggregating character+equips+buffs+passives inside atlas-channel would duplicate the owning service's logic (prohibited complexity, high divergence risk). The existing per-swing memoized lazy fetch (`character_attack_common.go:327-339`) already bounds cost to ≤1 GET per swing, only on swings that apply VENOM. Retained as-is.

---

## 7. Skill data / effects (atlas-data — attack `se`, MP-Eater effect, writer mastery)

Immutable game data per tenant version. No events exist or are needed. Verdict: **in-process TTL cache** (FR-4.3), copying task-060's landed semantics (`services/atlas-monsters/atlas.com/monsters/monster/information/cache.go`: positive 5m / negative 30s / `requests.ErrNotFound`-only negative caching / env kill-switch) re-expressed in-process per task-120 design §5.4. No event invalidation; TTL only.

---

## 8. Monster (reflect X/Y + template id; MP Eater Mp/MaxMp)

Deferred to task-120's live-monster mirror (design §5.1-5.3 in that task's worktree) with one in-scope extension (task-122 PRD FR-5.1): `LiveEntry` gains `X, Y int16`, seeded from the monster CREATED handler's already-fetched model, updated synchronously from atlas-channel's local monster-movement fold (the same local-write pattern as §2), backfilled by the REST fallback `Put`. Monster event coverage itself was audited by task-120 and is not re-audited here. VERIFY-AT-EXECUTION: reconcile with task-120's landed shape (it was unimplemented when this audit was written).

---

## FR-2.4 disposition summary

| # | Gap / thinness | Disposition | Type |
|---|---|---|---|
| 1 | STAT_CHANGED `Values` nil on most paths | Populate `Values` on all STAT_CHANGED emissions in atlas-character (bounded producer change) | FR-2.4(a) |
| 2 | Character position has no event | Feed snapshot position from atlas-channel's own movement fold + MAP_CHANGED target coords; REST fallback otherwise | FR-2.4 local-source (better than (a)) |
| 3 | Skill bulk-delete on char-DELETED, no events | Neutralized by session-scoped eviction | accepted (no change) |
| 4 | Effective stats: no event | Component stays on REST (existing memoization) | FR-2.4(b) |
| 5 | atlas-buffs restart drops buffs silently | Self-expiry via event `ExpiresAt` bounds divergence; documented | accepted residual |
| 6 | Reservations invisible in asset rows | Snapshot ignores reservation events = REST parity (VERIFY-AT-EXECUTION) | accepted (parity) |
| 7 | atlas-maps forced-return map on LOGOUT, no MAP_CHANGED | Snapshot evicted at logout | accepted (no change) |
| 8 | `RequestReserve` processes only first batched request | Not snapshot-relevant; flagged to owner as candidate atlas-inventory bug | escalate separately |
