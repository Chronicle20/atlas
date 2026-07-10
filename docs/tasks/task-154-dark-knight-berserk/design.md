# Dark Knight Berserk (1320006) — Design

Version: v1
Status: Approved PRD → Design
Task: task-154-dark-knight-berserk

---

## 1. Summary

atlas-buffs gains a `berserk` domain that tracks Dark Knights with Berserk leveled, re-evaluates `hp*100/effectiveMaxHp < x(level)` on the PRD's triggers, and drives the Cosmic-parity 5s-initial/3s-period broadcast by emitting one Kafka event per tick on the **existing** `EVENT_TOPIC_CHARACTER_BUFF_STATUS` topic. atlas-channel adds one stateless handler to its **existing** buff consumer that translates each event into the own + foreign `EffectSkillUse` packets via new `AnnounceBerserkEffect` helpers.

Two mechanism-level corrections to the PRD's assumed shape, both forced by verified codebase reality:

1. **The registry is Redis-backed, not an in-process singleton.** Every piece of atlas-buffs state already lives in `atlas.TenantRegistry` (Redis) — including the "poison registry" the PRD cited as the in-memory precedent (`services/atlas-buffs/atlas.com/buffs/character/registry.go:21-25`). atlas-buffs runs 2 replicas (`deploy/k8s/base/atlas-buffs.yaml`), so an in-memory registry would fork state per replica and double-broadcast. FR-4's intent (transient, tenant-scoped, reconstructable, no database persistence) is preserved; the mechanism follows the service's actual convention.
2. **Scheduling is a shared 1s scan ticker with per-entry deadlines, not a goroutine per character.** Same reason: two replicas each running per-character timers would double-emit. The service's established shape is a `tasks.Register` ticker scanning the registry (`tasks/poison.go`, `tasks/task.go:11-18`); berserk uses that shape with *atomic* per-entry claims (an improvement over the poison ticker's non-atomic get-then-update).

Everything else follows the PRD directly.

## 2. Verified ground truth

Facts the design rests on, all verified this phase (file:line, repo-relative; Cosmic references are to the local Cosmic checkout the PRD cites):

**Cosmic semantics** (`client/Character.java:1843-1870`): `checkBerserk` cancels any prior schedule; gates on Dark Knight job + `skilllevel > 0`; computes `hp*100/currentMaxHp < effect.getX()` (strict `<`, buff-inclusive max); registers a 5000ms-delay/3000ms-period repeating task that sends the self packet + map broadcast with the *captured* bool. Call sites: login (`PlayerLoggedinHandler.java:365`), HP change with death skip (`Character.java:8897-8900` — `chrDied → playerDead()`, no re-check), berserk-effect registration (`Character.java:4446`). The packets write skill id **1320006**, a level byte, skill level, then the berserk bool (`PacketCreator.java:3500-3519`).

**v83 WZ values** (`/…/Cosmic/wz/Skill.wz/132.img.xml`, skill 1320006): 30 levels, `x` = 21 at level 1 rising by 1 per level to 50 at level 30 (damage 32→100). The level nodes carry only `hs`, `x`, `damage` — **there is no `berserk` field in the WZ data**, confirming the threshold is `x` and nothing else.

**atlas-data**: parses effect `x` (`services/atlas-data/atlas.com/data/skill/reader.go:208`) but never calls `SetBerserk` — the effect model's `berserk` field is always zero. Consumers must use `x`; the PRD's "berserk is a type marker only" is actually understated: in Atlas the field is dead.

**Packet layer** (complete, no changes): `libs/atlas-packet/character/effect_body.go:62-84` — `CharacterSkillUseEffectBody(skillId, characterLevel, skillLevel, darkForceEffect, createOrDeleteDragon, left)` derives `isBerserk := skill.Id(skillId) == skill.DarkKnightBerserkId` internally; the **`darkForceEffect` bool parameter is the on/off flag**, written as a trailing byte only when `isBerserk` (`libs/atlas-packet/character/clientbound/effect_skill_use.go:74-76`, foreign at `:171-173`). Writer names `CharacterEffect`/`CharacterEffectForeign` (`clientbound/effect.go:12-13`) are registered in atlas-channel (`main.go:689-690`); mode resolves via the `operations`/`SKILL_USE` table (`effect_body.go:19`). Byte fixtures exist for v83/v84/v87 (`clientbound/effect_skill_use_test.go`).
  - *Naming discrepancy, no wire impact*: the IDA-derived comments in the fixture file label the 1x21001 family "berserk" and 1320006 "monster-magnet" (`effect_skill_use_test.go:17-18`), inverted relative to atlas-constants (`DarkKnightBerserkId = 1320006`, `constants.go:3006`; `DarkKnightMonsterMagnetId = 1321001`). Both client branches decode one trailing byte, and Cosmic empirically sends 1320006 + bool on v83, which is exactly what `effect_body.go` produces today. The labels should be cleaned up in a follow-up to the packet lib comments; nothing functional depends on them.

**STAT_CHANGED carries no HP value.** `StatusEventStatChangedBody{ChannelId, ExclRequestSent, Updates []stat.Type, Values map[string]interface{}}` (`services/atlas-character/atlas.com/character/kafka/message/character/kafka.go:341-346`); every HP-change site emits `Values = nil` (`character/processor.go:1155, 1205, 1270`). `Values` is populated only for level-up/job-change/ability flows and even then holds max-HP/primary stats, never current HP. → Re-evaluation must read HP via REST (PRD open question 3: **answered — REST read required**).

**Death**: `ChangeHP`/`SetHP` emit `DIED` *and then* `STAT_CHANGED(HP)` (`processor.go:1152-1155`, `1202-1205`). There is no revive method in atlas-character; revive flows through `SetHP` → another `STAT_CHANGED(HP)`.

**Skill status**: `EVENT_TOPIC_SKILL_STATUS`, envelope carries `SkillId` top-level; `UPDATED` body carries the new `Level` (`services/atlas-skills/atlas.com/skills/kafka/message/skill/kafka.go:53-80`). REST: `GET /characters/{characterId}/skills/{skillId}` (`skill/resource.go:23`).

**Effective stats**: `GET /worlds/{w}/channels/{c}/characters/{id}/stats` returns buff-inclusive `maxHP` (`services/atlas-effective-stats/atlas.com/effective-stats/character/resource.go:20-21`, `stat/rest.go:8-25`; `HYPER_BODY_HP` → max-HP multiplier at `stat/model.go:428-429`). It emits **no status events** on recompute — only `CLAMP` commands — so reacting to effective-stat changes must be REST-pull (this shapes Decision 5's grace delay).

**atlas-character REST** (`character/rest.go:15-47`): exposes `hp`, `maxHp`, `level`, `mapId` — **not** channelId. Channel must come from events (`STAT_CHANGED`/`MAP_CHANGED`/`LOGIN` bodies all carry `ChannelId`).

**atlas-buffs today**: consumes exactly one topic (`COMMAND_TOPIC_CHARACTER_BUFF`), has **zero outbound REST clients**, emits buff `APPLIED`/`EXPIRED` on `EVENT_TOPIC_CHARACTER_BUFF_STATUS` (envelope `StatusEvent[E]{WorldId, CharacterId, Type, Body}` — no channel, no transaction id; `kafka/message/character/kafka.go:61-90`) and `CHANGE_HP` commands. Ticker fan-out enumerates tenants from a Redis set (`character/registry.go:124-138`) and re-injects tenant context (`character/processor.go:145-160`).

**atlas-channel today**: buff consumer (`kafka/consumer/buff/consumer.go`) is the wiring template — registered at `main.go:200` (consumers) and `main.go:500` (handlers); handlers guard on `e.Type`, resolve the session via `session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(…)` (`session/processor.go:106-114`), announce own packet, then `_map.NewProcessor(l, ctx).ForOtherSessionsInMap(s.Field(), s.CharacterId(), …)` (`map/processor.go:67-70`, membership via REST). The exact own+foreign SKILL_USE pair already exists at `socket/handler/character_skill_use.go:108-110` using the `AnnounceSkillUse`/`AnnounceForeignSkillUse` helpers (`socket/handler/effects.go:19-39`) — which hard-code `darkForceEffect=false`, hence the new helpers below.

**Deploy surface**: `EVENT_TOPIC_CHARACTER_STATUS`, `EVENT_TOPIC_SKILL_STATUS`, `EVENT_TOPIC_CHARACTER_BUFF_STATUS`, and `BASE_SERVICE_URL` are all already in the shared configmap (`deploy/k8s/base/env-configmap.yaml:6,90,94,142`), and `atlas-buffs.yaml` inherits everything via `envFrom` → **zero k8s manifest changes**.

## 3. Decisions and alternatives

### D1 — State: new Redis `TenantRegistry`, namespace `buffs-berserk`

`berserk *atlas.TenantRegistry[uint32, Model]` added to the existing `Registry` struct alongside `characters`/`poisonTicks` (`character/registry.go:21-25` pattern), keyed by character id, tenant-scoped by the lib (`libs/atlas-redis/tenant_registry.go:39`).

- **Alternative — in-process `sync.Once`+`RWMutex` singleton (PRD FR-4 text):** rejected. With 2 replicas, the replica that consumes a character's Kafka partition holds the state while both replicas run tickers → double or missing broadcasts. Also contradicts the service's uniform Redis-state convention.
- **Restart behavior** (better than the PRD asked for): entries survive service restarts in Redis, so broadcasts resume on the next ticker pass without waiting for a per-character trigger.
- The berserk registry registers its tenants in the same `tenants` set the buff registry maintains (`registry.go:108-110`) so ticker fan-out sees tenants whose only tracked state is a Dark Knight with no active buffs.

Entry model (immutable, Builder pattern):

```
berserk.Model {
  worldId          world.Id
  channelId        channel.Id   // 0-sentinel "unknown" until first channel-bearing event
  channelKnown     bool
  characterId      uint32
  characterLevel   byte
  skillLevel       byte
  active           bool         // last-captured state (FR-3: not recomputed per tick)
  dirtyAt          time.Time    // zero = clean; else re-evaluation due at/after this instant
  nextBroadcastAt  time.Time    // next tick deadline
}
```

### D2 — Scheduling: 1s scan ticker + per-entry deadlines with atomic claims

One new task, `tasks.NewBerserkTick(l, 1000)`, registered in `main.go` next to the poison ticker. Each pass fans out per tenant (existing `GetTenants` + `tenant.WithContext` shape, `character/processor.go:145-160`) and for each tracked entry:

1. **Re-evaluate** if `dirtyAt` is set and `dirtyAt <= now`: atomically claim (clear `dirtyAt`), run the FR-1 computation (§5), write back `active`, `characterLevel`, `skillLevel`, `nextBroadcastAt = now + 5s`.
2. **Else broadcast** if `nextBroadcastAt <= now` and `channelKnown`: atomically claim (`nextBroadcastAt = now + 3s`), emit the BERSERK event with the stored state.

Claims use the atlas-redis atomic read-modify-write primitive (`DirectUpdate`-style WATCH/MULTI, `libs/atlas-redis/coalesced.go:217`) so that when both replicas scan the same entry, exactly one wins the claim and the other's transaction retries, re-reads the advanced deadline, and skips. (The poison ticker's `GetLastPoisonTick`→emit→`UpdatePoisonTick` sequence is not atomic and can in principle double-fire across replicas; berserk does not copy that flaw. If the atlas-redis API surface makes claim-result plumbing awkward, the plan phase may fall back to a `SET NX PX` claim key per (character, deadline) — the design constraint is only *at-most-one emitter per deadline*.)

- **Alternative — goroutine timer per character (Cosmic literal, PRD FR-3 wording):** rejected; not replica-safe, dies on restart, and no precedent in this service.
- **Cadence parity:** 5s initial delay and 3s period are honored with ≤1s scan granularity — the same granularity the poison ticker already applies to its 1s tick. Worst-case first-broadcast latency after a trigger is ~1s (dirty scan) + 5s (initial delay), inside the PRD's "within one broadcast tick" acceptance envelope.
- Interval constants (`5*time.Second`, `3*time.Second`, 1s scan) are named constants in the berserk package — no magic numbers.

### D3 — Consumers only mark state; the ticker does all I/O

Kafka handlers in atlas-buffs make no REST calls, with a single exception: the LOGIN handler's one skill-level lookup (§4.1 trigger table), which is unavoidable because no event carries the level at login. Otherwise handlers translate triggers into registry writes (`dirtyAt`, channel updates, entry create/remove) and return. The ticker executes re-evaluations (2 REST reads) and emissions.

Consequences, all desirable: consumer lag can't build up behind slow REST calls (FR-5 resilience); re-evaluations per character are serialized by the atomic claim regardless of which topic triggered them (NFR ordering); a burst of HP changes collapses into one re-evaluation at the next pass (Cosmic's cancel-and-replace semantics, coalesced); the ≤1s deferral is invisible on the wire because the 5s initial delay dominates.

### D4 — Emission split: per-tick events from atlas-buffs; atlas-channel stateless (PRD open question 2: **answered**)

atlas-buffs emits one event per broadcast tick; atlas-channel translates each event to packets with no timer and no state.

- **Alternative — buffs emits state changes only; channel owns the 3s re-broadcast timer:** rejected. atlas-channel would need its own per-character schedule registry (duplicating D1/D2 in every channel instance), plus map-enter hooks to cover late joiners — the periodic re-broadcast covers them for free. The scope interview already assigned timing to atlas-buffs.
- **Cost:** one small keyed message per 3s per tracked Dark Knight (NFR-accepted).

### D5 — Data access: first outbound REST clients in atlas-buffs

New `external/` client family, copied from atlas-effective-stats' layout (`services/atlas-effective-stats/atlas.com/effective-stats/external/`):

| Client | Route | Used for | Root URL domain |
|---|---|---|---|
| `external/character` | `GET characters/{id}` | current `hp`, `level` | `CHARACTERS` |
| `external/skills` | `GET characters/{id}/skills/{skillId}` | Berserk level at login | `SKILLS` |
| `external/effectivestats` | `GET worlds/{w}/channels/{c}/characters/{id}/stats` | buff-inclusive `maxHP` | `EFFECTIVE_STATS` |
| `external/data/skill` | `GET data/skills/{id}` | per-level effect `x` | `DATA` |

All via `requests.RootUrl(domain)` (`libs/atlas-rest/requests/url.go:14`) with the `BASE_SERVICE_URL` fallback — no hard-coded URLs, no manifest edits (known footgun: hard-coded `*_SERVICE_URL` breaks env overlays).

**Caching:** skill-effect data (`Effects[level-1].X`) is immutable per tenant → in-process per-tenant cache (`cache.go`, `sync.Once`/`RWMutex` per guidelines), fetched once for `skill.DarkKnightBerserkId` on first use. Skill level and character level are cached in the registry entry, refreshed by events/re-evaluations. HP and effective max HP are never cached — read fresh per re-evaluation (2 REST calls, tracked characters only).

**Effective-stats staleness (the one real race):** when Hyper Body applies/expires, atlas-buffs *is the producer* of the buff event that atlas-effective-stats consumes to recompute max HP — an immediate re-evaluation would be guaranteed to read the stale value. Mitigation: buff-origin triggers set `dirtyAt = now + reevalGrace` (named constant, 2s) instead of `now`, giving effective-stats a consume-and-recompute window; any residual staleness self-heals at the next HP change (Cosmic-equivalent "recompute on next trigger" semantics, PRD FR-5). Character-origin triggers (HP change, login, transfer) set `dirtyAt = now` — no race, since HP changes don't move max HP, and max-HP-bearing `STAT_CHANGED` events (level-up, AP into HP) also get the grace (they race the same effective-stats recompute of `MAX_HP`).

### D6 — Event contract: new `BERSERK` type on `EVENT_TOPIC_CHARACTER_BUFF_STATUS`

```go
// kafka/message/character/kafka.go (atlas-buffs), mirrored in atlas-channel's kafka/message/buff
EventStatusTypeBerserk = "BERSERK"

type BerserkStatusEventBody struct {
    TransactionId  uuid.UUID  `json:"transactionId"`
    ChannelId      channel.Id `json:"channelId"`
    SkillId        uint32     `json:"skillId"`        // skill.DarkKnightBerserkId; explicit for forward generality
    CharacterLevel byte       `json:"characterLevel"` // saves the channel handler a character REST read per tick
    SkillLevel     byte       `json:"skillLevel"`
    Active         bool       `json:"active"`
}
```

Envelope: the topic's existing `StatusEvent[E]{WorldId, CharacterId, Type, Body}`; key `producer.CreateKey(int(characterId))` → per-character ordering alongside the character's other buff events. The existing envelope has no transaction id, so it rides in the body (PRD §5 minimum satisfied).

- **Alternative — dedicated `EVENT_TOPIC_CHARACTER_BERSERK_STATUS`:** rejected. Buys no isolation (consumers type-guard anyway; existing consumers of this topic ignore unknown types by design), and costs configmap + `.env.example` + overlay wiring — the exact "new topic env var missing from live config" failure family this repo has been bitten by.
- ChannelId in the body lets atlas-channel use the precise `sc.Is(tenant, worldId, channelId)` guard (like the character consumer, `kafka/consumer/character/consumer.go:443`) instead of the buff consumer's world-only guard.

### D7 — Death: `hp > 0` is part of the formula

`active := skillLevel > 0 && hp > 0 && hp*100/effectiveMaxHp < x`. This handles the DIED→STAT_CHANGED emission order with zero special-casing: the death-accompanying `STAT_CHANGED(HP)` re-evaluates to `active=false` (without the guard, `0*100/max = 0 < x` would light the aura on a corpse). No `DIED` consumer needed. The entry keeps ticking `active=false` while dead — which *clears* the aura for all observers, strictly safer than Cosmic (which keeps broadcasting the stale pre-death state until revive) and satisfying FR-2's intent that a dead character never shows the aura. Revive flows through `SetHP` → `STAT_CHANGED` → normal re-evaluation.

### D8 — Channel edge: entries created without a known channel skip broadcast until resolved

The skill `UPDATED` envelope has no channel, and atlas-character REST doesn't expose one (§2). For the SP-allocation 0→1 path the entry is created with `channelKnown=false`; **every consumed character-status event that carries a `ChannelId` (`STAT_CHANGED`, `MAP_CHANGED`, `LOGIN`) refreshes the tracked entry's channel**, so the very next such event — in practice the SP-change `STAT_CHANGED` emitted by the same allocation flow, or the next HP tick — fills it in. Until then the ticker re-evaluates but does not emit (can't route). This same refresh rule *is* the channel-transfer handling: no separate `CHANNEL_CHANGED` bookkeeping is required, though the consumer handles it explicitly for completeness.

## 4. Components

### 4.1 atlas-buffs — new `berserk/` package

| File | Responsibility |
|---|---|
| `berserk/model.go` + `builder.go` | Immutable entry model (§D1), builder enforcing invariants |
| `berserk/registry.go` | `TenantRegistry[uint32, Model]` wrapper: `Track`, `Untrack`, `MarkDirty(at)`, `UpdateChannel`, `UpdateSkillLevel`, `ClaimReeval`, `ClaimBroadcast`, `GetTenants` integration |
| `berserk/processor.go` | `Evaluate` (pure FR-1 computation given inputs), `Reevaluate(mb)` (fetch → evaluate → store), `ProcessTicks` (ticker entry point per tenant), `TrackOnLogin`, trigger handlers' domain logic; `AndEmit` variants per convention |
| `berserk/producer.go` | `berserkStatusEventProvider(...)` on `EnvEventStatusTopic` |
| `berserk/cache.go` | per-tenant effect-`x` cache |
| `external/{character,skills,effectivestats,data/skill}/` | REST clients (requests.go + rest.go each, per guidelines) |
| `tasks/berserk.go` | 1s ticker calling `berserk.ProcessBerserkTicks` (poison.go shape) |
| `kafka/consumer/character_status/consumer.go` | new consumer: `EVENT_TOPIC_CHARACTER_STATUS` → LOGIN / LOGOUT / STAT_CHANGED / MAP_CHANGED / CHANNEL_CHANGED handlers |
| `kafka/consumer/skill_status/consumer.go` | new consumer: `EVENT_TOPIC_SKILL_STATUS` → UPDATED (and DELETED→untrack-if-berserk) handlers |
| `kafka/message/character/kafka.go` | BERSERK event type + body; local copies of the character/skill status envelopes it consumes |

Trigger → registry action mapping (consumers, no REST):

| Event | Action |
|---|---|
| `LOGIN` | *(only place a skills REST call happens outside the ticker — one `GET characters/{id}/skills/1320006`; acceptable at login rates and unavoidable: no event carries the level at login)* level==0 → ignore; level>0 → `Track` with world/channel/skillLevel, `dirtyAt=now` |
| `LOGOUT` | `Untrack` |
| `STAT_CHANGED` | untracked → ignore. Tracked: refresh channel; `Updates` ∋ `stat.TypeHp` → `dirtyAt=now`; ∋ `stat.TypeMaxHp` → `dirtyAt=now+grace` |
| `MAP_CHANGED` / `CHANNEL_CHANGED` | tracked → refresh world/channel, `dirtyAt=now` (Cosmic re-checks on transfer) |
| skill `UPDATED`, `SkillId==skill.DarkKnightBerserkId` | level>0 → upsert (`Track` if new, `channelKnown=false`; else `UpdateSkillLevel`), `dirtyAt=now`; level==0 → `Untrack` |
| buff `Apply`/`Cancel`/`CancelAll`/`CancelByStatTypes`/`ExpireBuffs` (in-process hook) | affected character tracked && changes intersect max-HP stat types → `dirtyAt=now+grace` |

The buff hook is a small helper `affectsMaxHp(changes)` (shape of `isDiseaseChange`, `character/immunity.go`) called inside the existing processor methods' `message.Emit` closures (`character/processor.go:43-143`). The max-HP stat-type set mirrors what atlas-effective-stats maps to `TypeMaxHp` — today `character.TemporaryStatTypeHyperBodyHP` (`libs/atlas-constants/character/temporary_stat.go:19`); the plan phase greps effective-stats' buff→stat mapping (`stat/model.go:428-429` region) and mirrors the full set, with a comment pinning the source.

Re-evaluation (`Reevaluate`, run by ticker under a claimed entry):

```
skillLevel := entry.skillLevel                     // event-maintained
x          := effectCache.X(tenant, skillLevel)     // cached, atlas-data on first use
char       := external/character GET                // hp, level
maxHp      := external/effectivestats GET           // needs channelKnown for the route
active     := skillLevel > 0 && char.hp > 0 && uint32(char.hp)*100/maxHp < uint32(x)
store: active, characterLevel=char.level, nextBroadcastAt=now+initialDelay
```

Any lookup failure: log at warn, leave the entry as it was **with `dirtyAt` re-armed** (`now + 1s`) so the re-evaluation retries on a later pass rather than silently freezing on stale state; the existing schedule keeps broadcasting the last-known state meanwhile (FR-5 semantics). Integer math throughout; `maxHp==0` guard → skip with warn.

### 4.2 atlas-channel — one handler + two helpers

- `socket/handler/effects.go`: add `AnnounceBerserkEffect(l)(ctx)(wp)(skillId uint32, characterLevel byte, skillLevel byte, active bool)` and `AnnounceForeignBerserkEffect(...)(characterId uint32, ...)` — identical to `AnnounceSkillUse`/`AnnounceForeignSkillUse` (`effects.go:19-39`) but passing `active` as the `darkForceEffect` argument of `CharacterSkillUseEffectBody`/`...ForeignBody`. The packet lib derives the isBerserk gate from the skill id itself (§2); no registry, writer, or template work.
- `kafka/consumer/buff/consumer.go`: add `handleStatusEventBerserk` registered in the existing `InitHandlers` (`consumer.go:34-55`) decoding `buff2.StatusEvent[buff2.BerserkStatusEventBody]`:
  1. guard `e.Type != EventStatusTypeBerserk` → return;
  2. guard `!sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.Body.ChannelId)` → return;
  3. `session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, …)` → own `AnnounceBerserkEffect`, then `ForOtherSessionsInMap(s.Field(), s.CharacterId(), AnnounceForeignBerserkEffect(...))` — the `character_skill_use.go:108-110` shape. No session → no-op (character transferred/logged out between emit and consume; next tick self-corrects).
- `kafka/message/buff/kafka.go`: add the type constant + body struct.
- No `main.go` changes: the buff consumer is already registered (`main.go:200`, `main.go:500`).

### 4.3 Flow walkthroughs

- **HP drops below threshold:** atlas-character emits `STAT_CHANGED(HP)` → buffs marks dirty → ≤1s later ticker re-evaluates (2 REST reads) → `active=true`, first broadcast 5s later → channel writes own+foreign packets with the flag on; repeats every 3s. Heal above threshold: same path, flag off (strict `<`: equality is off).
- **Hyper Body expires (HP constant):** buffs' expiration ticker emits EXPIRED; the in-process hook marks dirty at `now+grace`; re-evaluation reads the recomputed (lower) maxHp → HP% may now cross the threshold → state flips with no HP event (AC-3).
- **Observer enters the map:** next ≤3s tick's foreign broadcast covers them — no map-enter hook anywhere (AC-6).
- **Login / logout / transfer / death / SP allocation:** per the trigger table; all end in either `Untrack` or a re-evaluation + fresh 5s schedule.

## 5. Concurrency and ordering

- **Cross-replica:** every entry mutation is a single atomic registry operation; ticker claims are atomic (D2) — at most one replica emits per deadline, at most one runs a given re-evaluation.
- **Cross-topic races** (character-status vs skill-status vs buff hook for one character): consumers only write fields (`dirtyAt`, `channelId`, `skillLevel`) via atomic read-modify-write; the ticker's claimed re-evaluation reads a consistent snapshot and computes the full state fresh. Last-writer-wins on `dirtyAt` is correct — a re-evaluation is idempotent and always computes from current data, so *which* trigger fires it is immaterial (this is why FR-2's "cancel-then-reschedule must be atomic" is satisfied by claim atomicity rather than a per-character lock).
- **Emit-vs-consume staleness** (channel handler races a map transfer): foreign broadcast targets are resolved from the *live session* at consume time (`s.Field()`), not from the event, so a stale `ChannelId` at worst drops one tick (guard fails or no session); the next tick routes correctly.

## 6. Testing

Per project rules: Builder-pattern setup only, no `*_testhelpers.go`; table-driven.

- **`berserk.Evaluate` (pure):** threshold boundary table — below/at/above `x`% (equality must be **inactive**), `hp=0` (dead → inactive regardless of ratio), `skillLevel=0`, `maxHp` inflated/deflated by buff (Hyper Body flip with constant HP), integer-division edges (hp\*100 overflow bounds with uint32 math), per-level `x` resolution.
- **Registry/claims:** track/untrack lifecycle; channel-unknown → no broadcast claim; concurrent `ClaimBroadcast`/`ClaimReeval` from two goroutines → exactly one winner (the cancel-reschedule race the PRD's AC names); dirty coalescing (N marks → 1 re-evaluation); grace-deferred dirty not claimable early.
- **Consumers (atlas-buffs):** each trigger row of the table drives the expected registry mutation; wrong-type/wrong-skill events are no-ops; untracked STAT_CHANGED does zero work; tenant isolation (two tenants, same character id).
- **Ticker/processor with mocked externals:** re-evaluation happy path emits nothing until deadline; broadcast tick emits the correct envelope/body (golden JSON); lookup failure re-arms dirty and keeps last state; `maxHp=0` guarded.
- **atlas-channel handler:** type/world/channel guards; own+foreign announced with `darkForceEffect=active` threaded (assert against the packet lib body constructors — the existing byte fixtures already pin the wire format); no-session no-op.
- **Verification suite (PRD AC):** `go test -race ./...`, `go vet ./...`, `go build ./...` in atlas-buffs + atlas-channel; `docker buildx bake atlas-buffs atlas-channel`; `tools/redis-key-guard.sh` (all Redis access goes through `libs/atlas-redis` types — the new registry namespace is `TenantRegistry`-based, so the guard stays clean).

## 7. PRD open questions — resolutions

1. **GM-hide:** unchanged — follow-up when hide lands; foreign broadcast is unconditional here.
2. **Tick emission split:** per-tick events from atlas-buffs; stateless atlas-channel (D4).
3. **STAT_CHANGED payload:** carries no HP value (verified §2); HP read via atlas-character REST inside the ticker's re-evaluation (D3/D5).
4. **v83 `x` values:** verified from local WZ — levels 1–30, `x` 21→50 (§2); runtime-resolved from atlas-data regardless.

## 8. Out of scope (unchanged from PRD)

Server-side damage amplification, GM-hide, Beholder/Dragon Blood, Evan Dragon Fury, packet/writer/template changes, persistence, UI.
