# Buff Domain

## Responsibility

The buff domain manages temporary stat modifications for game characters, including application, cancellation, automatic expiration of buffs, disease immunity checks, stat-value mutation on existing buffs, correlation-based bulk cancellation, and periodic HP effect processing.

## Core Models

### buff.Model

Immutable representation of a buff.

| Field | Type | Description |
|-------|------|-------------|
| id | uuid.UUID | Unique buff identifier |
| sourceId | int32 | Source identifier (skill/item ID) |
| level | byte | Buff level |
| duration | int32 | Duration in milliseconds |
| changes | []stat.Model | Stat modifications |
| createdAt | time.Time | Creation timestamp |
| expiresAt | time.Time | Expiration timestamp |
| noExpiry | bool | True for a buff that never expires on its own; removed only by explicit cancel |
| correlationId | string | Opaque identifier of what granted this buff (for example, an event occurrence id), used for correlation-based bulk cancellation |

### stat.Model

Immutable representation of a stat modification.

| Field | Type | Description |
|-------|------|-------------|
| statType | string | Stat type identifier |
| amount | int32 | Modification amount |

### character.Model

Represents a character with active buffs.

| Field | Type | Description |
|-------|------|-------------|
| tenant | tenant.Model | Tenant context |
| worldId | world.Id | World identifier |
| channelId | channel.Id | Channel identifier |
| characterId | uint32 | Character identifier |
| buffs | map[string]buff.Model | Active buffs keyed by a composite string: "<sourceId>" for a normal whole-source buff, or "<sourceId>:<statType>" for an accumulate-mode buff |

### periodic.Effect

Immutable row of the declarative periodic-effect table. Keyed by temporary-stat type; resolves a buff's stat type to its tick schedule and behavior.

| Field | Type | Description |
|-------|------|-------------|
| statType | character.TemporaryStatType | Temporary-stat type stored on the buff change this row keys off |
| interval | time.Duration | Cadence between ticks for this effect |
| resource | Resource | Character resource the tick moves (only `ResourceHP` exists) |
| direction | Direction | Sign applied to the per-tick magnitude (`Drain` = -1, `Restore` = 1) |
| floor | bool | Whether the tick must clamp the resource at 1 rather than let it reach 0 |
| specialEffect | bool | Whether each tick should also broadcast the source skill's special user effect |

The table currently defines three rows: POISON (1s interval, drain, no floor), DRAGON_BLOOD (4s interval, drain, floor, special effect), RECOVERY (5s interval, restore, no floor).

### character.PeriodicEntry

One due-able periodic effect found by a tick-pass scan, produced by `Registry.GetPeriodicEntries`.

| Field | Type | Description |
|-------|------|-------------|
| Tenant | tenant.Model | Tenant context |
| WorldId | world.Id | World identifier |
| ChannelId | channel.Id | Channel identifier |
| CharacterId | uint32 | Character identifier |
| StatType | string | Periodic temporary-stat type |
| Amount | int32 | Per-tick magnitude stored on the owning buff's stat change |
| SourceId | int32 | Source skill id of the owning buff, carried so a special-effect row can name the skill whose animation to pulse |

### character.TickKey

Identifies one periodic effect on one character for throttle-state keying (character + stat type, so two periodic effects on the same character throttle independently).

| Field | Type | Description |
|-------|------|-------------|
| CharacterId | uint32 | Character identifier |
| StatType | string | Periodic temporary-stat type |

### character.StatValueUpdate

One stat-value mutation request against an existing (or, with `CreateIfMissing`, to-be-created) buff.

| Field | Type | Description |
|-------|------|-------------|
| SourceId | int32 | Source identifier of the target buff |
| StatType | string | Stat type to mutate |
| Operation | string | `INCREMENT` (add, clamped to Cap) or `SET` (replace outright) |
| Amount | int32 | Value to add (INCREMENT) or set (SET) |
| Cap | int32 | Upper bound for INCREMENT; ignored for SET |
| CreateIfMissing | bool | When true, an INCREMENT against a missing buff creates a no-expiry buff instead of no-op |
| Level | byte | Source skill level stamped on a buff created by CreateIfMissing |

## Invariants

- Each buff has a unique UUID generated at creation
- In default (non-accumulate) mode, a buff is keyed by sourceId within a character's buff map; applying a buff with an existing sourceId replaces the previous buff
- In accumulate mode, each change is stored as its own buff keyed by (sourceId, statType); a re-apply of the same stat replaces only that key, other stats of the same source are left intact
- Expiration time is calculated as createdAt + duration milliseconds
- A buff is expired when expiresAt is before current time, unless noExpiry is set, in which case it is never expired
- Duration must be positive for an expiring buff (ErrInvalidDuration); a no-expiry buff has duration 0
- Changes must not be empty (ErrEmptyChanges)
- Disease buffs (STUN, POISON, SEAL, DARKNESS, WEAKEN, CURSE, SEDUCE, CONFUSE, UNDEAD, SLOW, STOP_PORTION) are blocked on Apply if the character has a HOLY_SHIELD buff active
- The HOLY_SHIELD immunity check applies only to Apply; Cancel, CancelAll, and CancelByStatTypes are not gated by it
- A periodic effect ticks only when its own row interval has elapsed since its last tick for that (character, statType)
- Periodic tick magnitude is clamped to 32767 before conversion to the resource-change command's int16 amount
- A non-positive stored periodic magnitude is skipped (no tick emitted)
- A floor-enforcing periodic drain never reduces HP below 1
- When two live buffs on one character carry the same periodic stat type, the entry with the largest Amount is the one that ticks
- CorrelationId-based cancellation matches on exact string equality; an empty correlationId is not honored by the processor's CancelByCorrelation
- UpdateStatValue INCREMENT is a no-op when the current amount is already at or above Cap, or when Amount is non-positive; SET is a no-op when Amount is less than 1
- UpdateStatValue never changes a buff's stored createdAt/expiresAt

## State Transitions

- A buff is created by Apply (fresh or replacing an existing whole-source/per-stat entry)
- A buff's stat amount may be mutated in place by UpdateStatValue without altering its identity, level, duration, or timestamps
- A buff is removed by expiry (ExpireBuffs / ExpireForCharacter, driven by elapsed time), by explicit Cancel/CancelAll/CancelByStatTypes/CancelByCorrelation, or as a side effect of UpdateStatValue's CreateIfMissing path creating a new buff in place of a missing one
- Removing a buff whose changes include a periodic stat type clears that periodic tick's throttle state
- Removing a buff whose changes affect max HP schedules a grace-deferred berserk re-evaluation for the character (see Berserk Domain)

## Processors

### Processor

Primary domain processor for buff operations.

| Method | Description |
|--------|-------------|
| GetById | Retrieve character with buffs by character ID |
| Apply | Apply buff to character with disease immunity check; `accumulate` selects whole-source replace (false) or per-stat accumulate (true); `noExpiry` marks the buff as never expiring; emits one APPLIED event per stored buff |
| Cancel | Cancel buff(s) by sourceId and emit one EXPIRED event per removed buff |
| CancelAll | Cancel all buffs for character and emit expired events |
| CancelByStatTypes | Cancel any buff whose Changes() intersects a stat-type set; emits one EXPIRED event per cancelled buff |
| CancelByCorrelation | Cancel every buff across the tenant carrying a given correlationId; emits one EXPIRED event per removed buff |
| UpdateStatValue | Apply an INCREMENT or SET mutation to an existing buff's stat (or create one with CreateIfMissing); emits APPLIED for a created buff, STAT_UPDATED for a mutated one |
| ExpireBuffs | Process and emit events for all expired buffs across every character in the tenant |
| ExpireForCharacter | Process and emit events for one character's expired buffs |
| ProcessPeriodicTicks | Find due periodic effects across the tenant and emit resource-change commands (and, for rows flagged specialEffect, a visual-pulse event) |

### Registry

Redis-backed buff storage (singleton). Per-tenant key isolation via TenantRegistry.

| Method | Description |
|--------|-------------|
| Apply | Add buff for character; `accumulate=false` replaces the whole-source buff keyed by sourceId, `accumulate=true` stores one buff per stat change keyed by (sourceId, statType); `noExpiry` builds a non-expiring buff; returns the buff(s) created |
| Get | Retrieve character by ID |
| GetTenants | Get all tenants with registered characters |
| GetCharacters | Get all characters for a tenant |
| Cancel | Remove all buffs matching sourceId (may be more than one in accumulate mode); returns ErrNotFound if none matched |
| CancelAll | Remove all buffs for character |
| CancelByStatTypes | Filter and remove buffs whose Changes() intersects a stat-type set; returns the cancelled buffs |
| CancelByCorrelation | Filter and remove buffs on one character whose CorrelationId matches; returns the cancelled buffs |
| UpdateStatValue | Mutate (or, with CreateIfMissing, create) a stat value on a character's buff; returns the resulting buff plus changed/created flags |
| GetExpired | Remove and return expired buffs for character |
| HasImmunity | Check if character has HOLY_SHIELD buff active |
| GetPeriodicEntries | Scan the tenant's stored characters and return one entry per (character, statType) whose stat type is periodic and whose owning buff has not expired |
| GetPeriodicTick | Get last tick timestamp for a (character, statType) key |
| UpdatePeriodicTick | Record a tick timestamp for a (character, statType) key with a bounded TTL |
| ClearPeriodicTick | Remove the throttle entry for a (character, statType) key |
| ClearPeriodicTicksFor | Clear the throttle entry for every periodic stat type carried by a set of removed buffs' changes |

## Background Tasks

### Expiration Task

Runs on configurable interval (default 10000ms) to:
1. Iterate all tenants with active buffs
2. For each tenant, process all characters
3. Remove expired buffs and emit expired events

### PeriodicTick Task

Runs on configurable interval (default 1000ms) to:
1. Iterate all tenants with active buffs
2. For each tenant, scan for periodic effect entries due since their own row interval
3. Emit a resource-change command per due entry, clamped and floored per the effect's table row
4. Emit a visual-pulse event alongside the resource-change command for rows flagged specialEffect

## Berserk Domain

### Responsibility

Tracks Dark Knight Berserk aura state per character: whether the character's current HP ratio is under the skill's threshold, re-evaluating on relevant character/skill events and broadcasting the resulting state on a schedule.

### Core Models

#### berserk.Model

Immutable representation of one tracked Dark Knight.

| Field | Type | Description |
|-------|------|-------------|
| worldId | world.Id | World identifier |
| channelId | channel.Id | Channel identifier; meaningless until channelKnown |
| channelKnown | bool | Whether a channel-bearing event has populated worldId/channelId |
| characterId | uint32 | Character identifier |
| characterLevel | byte | Character level, refreshed on each re-evaluation |
| skillLevel | byte | Berserk skill level |
| active | bool | Last-evaluated aura active state |
| dirtyAt | time.Time | Time at/after which a re-evaluation is due; zero means none scheduled |
| nextBroadcastAt | time.Time | Time at/after which the next broadcast tick is due; zero means no evaluation has completed yet |

### Invariants

- A re-evaluation is due only when channelKnown is true, dirtyAt is non-zero, and dirtyAt is not after the current time
- A broadcast is due only when channelKnown is true, nextBroadcastAt is non-zero, and nextBroadcastAt is not after the current time
- Active state: `active := skillLevel > 0 && hp > 0 && hp*100/effectiveMaxHp < x` (strict less-than; at exactly x% the aura is off; hp == 0 folds death handling into the formula)
- Level 0 (not learned) is never tracked: TrackOnLogin and HandleSkillUpdated skip registry entry creation for skill level 0, and a skill-level update to 0 untracks the character
- A skill-update-created entry has no channel until the next channel-bearing character event
- A max-HP-affecting buff change (apply/cancel/expiry) schedules a grace-deferred re-evaluation (ReevalGrace, 2s) so atlas-effective-stats can recompute max HP first
- An HP-only stat change schedules an immediate re-evaluation; a max-HP stat change schedules a grace-deferred one even if HP also moved
- A map or channel transfer schedules an immediate re-evaluation and updates the tracked channel
- A failed re-evaluation lookup re-arms dirtyAt after ReevalRetryDelay (1s) rather than leaving the schedule stalled
- On a state transition (or the first evaluation), the new state is emitted inline in the same pass the flip is detected, and the broadcast schedule resets to one BroadcastPeriod (3s) out
- An unchanged state on re-evaluation preserves the running broadcast cadence and emits nothing inline; the periodic broadcast still refreshes late-joining observers
- Effect x values (per skill level) are cached per tenant; a failed fetch is not cached

### Processors

#### Processor

Primary domain processor for Berserk tracking.

| Method | Description |
|--------|-------------|
| TrackOnLogin | Look up the character's Berserk skill level and register a tracked entry if learned (level > 0) |
| Untrack | Remove a character's tracked entry |
| HandleStatChanged | Refresh the tracked channel and mark the entry dirty when the stat update includes HP or max HP |
| HandleTransfer | Refresh the tracked channel and mark the entry dirty on map/channel transfer |
| HandleSkillUpdated | Update the tracked skill level, track a new entry if none exists, or untrack when the level drops to 0 |
| MarkMaxHpDirty | Schedule a grace-deferred re-evaluation for a character whose max HP was affected by a buff change |
| ProcessTicks | One scan pass: claim and process due re-evaluations, then claim and emit due broadcasts |

#### Registry

Redis-backed Berserk tracking storage (singleton, namespace `buffs-berserk`). Per-tenant key isolation via TenantRegistry; shares the buff registry's tenant set.

| Method | Description |
|--------|-------------|
| Track | Register a tracked entry for a character |
| Untrack | Remove a character's tracked entry |
| Get | Retrieve a character's tracked entry |
| GetAll | Retrieve all tracked entries for a tenant |
| GetTenants | Get all tenants with tracked entries |
| MarkDirty | Schedule a re-evaluation at/after a given time; no-op for an untracked character |
| UpdateChannel | Update the tracked world/channel; no-op for an untracked character |
| UpdateSkillLevel | Update the tracked skill level; returns ErrNotFound for an untracked character |
| ClaimReeval | Atomically claim a due re-evaluation, clearing dirtyAt; single-winner across replicas |
| ClaimBroadcast | Atomically claim a due broadcast tick, advancing the deadline by BroadcastPeriod; single-winner across replicas |
| StoreEvaluation | Record a re-evaluation outcome: active state, refreshed character level, next broadcast deadline |

### Background Tasks

#### BerserkTick Task

Runs on configurable interval (default 1000ms) to fan out one tick-processing pass per tenant with a tracked Dark Knight.
