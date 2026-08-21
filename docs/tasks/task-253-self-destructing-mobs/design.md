# Self-Destructing Mobs — Design

Task: `task-253-self-destructing-mobs` (renumbered from 252 — number collided with
`task-252-jukebox-cash-item`).
Input: [`prd.md`](./prd.md) (approved).
Status: Draft for review.

---

## 1. What this document decides

The PRD leaves seven open questions and one command-shape choice. This design closes all of
them against evidence, then fixes the component boundaries, the wire contracts, and the
concurrency model for the implementation plan.

Nothing here is implemented. No code has been changed.

---

## 2. Derivations

### 2.1 `CMobPool::OnMobLeaveField` — trailing `int32` on dead-type 4

Swept **all ten** IDBs (not spot-checked). For each, the function was located by symbol and
its `CInPacket::Decode*` call sites enumerated over the function's own address range.

| Version | Session | `OnMobLeaveField` | Decodes | Trailing `int32` when deadType == 4 |
|---|---|---|---|---|
| GMS v48 | `12a398ce` | `0x55957b` | `Decode4`, `Decode1` | **no** |
| GMS v61 | `921fdbb5` | `0x5d4b87` | `Decode4`, `Decode1` | **no** |
| GMS v72 | `99e435d8` | `0x6258a1` | `Decode4`, `Decode1` | **no** |
| GMS v79 | `5a1cd4f3` | `0x646ff6` | `Decode4`, `Decode1` | **no** |
| GMS v83 | `754107bf` | `0x67961d` | `Decode4`, `Decode1` | **no** |
| GMS v84 | `46c2a2eb` | `0x6901b3` | `Decode4`, `Decode1` | **no** |
| GMS v87 | `c0829805` | `0x6b5169` | `Decode4`, `Decode1` | **no** |
| GMS v92 | `019cd393` | `0x64bb90` | `Decode4`, `Decode1`, `Decode4` | **yes** |
| GMS v95 | `ecc757f4` | `0x658b90` | `Decode4`, `Decode1`, `Decode4` | **yes** |
| JMS v185 | `a977912e` | `0x6f8a1f` | `Decode4`, `Decode1`, `Decode4` | **yes** |

v95 pseudocode (symbolised, `0x658b90`):

```c
dwMobID = CInPacket::Decode4(iPacket);
v4      = CInPacket::Decode1(v3);
v5      = 0;
if ( v4 == 4 )
    v5 = CInPacket::Decode4(v3);          // -> v6->m_dwSwallowCharacterID
```

**Resolves PRD OQ3.** The gate boundary is GMS **v92**, not v95. The read is
*unconditional on dead-type 4* — it is not predicated on a swallow character being set — so a
`destroyType == 4` encode **must** carry the trailing `int32` on v92/v95/JMS and **must not**
on v48–v87. Today `destroy.go` writes it unconditionally on every version, which desyncs
v48–v87 whenever dead-type 4 is sent.

**Resolves PRD OQ2.** There is no "swallow only when set" branch to discover: on v92+ the
byte is the discriminator and the trailing field always follows it. A self-destruct that wants
dead-type 4 on v92+ therefore writes a trailing `int32(0)`; the client stores `0` into
`m_dwSwallowCharacterID` and `CMob::OnSwallowed` runs with no swallower — see §2.2 for why
that is acceptable and §4.6 for the fallback.

### 2.2 Client-side meaning of each dead-type

`CMobPool::Update` drains `m_lMobDelayedDead` and switches on `m_nDeadType`. The switch is
**version-dependent**, so the same byte does not mean the same animation everywhere.

v95 (`0x658610`), a full 6-arm switch:

| dead-type | handler | effect |
|---|---|---|
| 0 | `CMob::OnDie` (`0x64e4b0`) | unreachable from this path — `OnMobLeaveField` removes a type-0 mob immediately instead of queueing it |
| 1 | `CMob::OnDie` | ordinary death / fade-out |
| 2 | `CMob::OnBomb` (`0x650ec0`) | bomb animation |
| 3 | `CMob::OnDestructByMiss` (`0x64ea30`) | `play_mob_sound(SE_MOB_DIE)`, `m_nOneTimeAction = 40`, `m_nSuspended = 2` |
| 4 | `CMob::OnSwallowed` (`0x641810`) | swallow; consumes `m_dwSwallowCharacterID` |
| 5 | `CMob::OnDie` | ordinary death / fade-out |
| other | *(no handler)* | mob is dequeued with no animation |

v87 (`0x6b4c78`) collapses the same space into two arms:

```c
if ( deadType <= 1 || deadType == 3 )  CMob::OnDie(mob);
else                                   sub_69E44A(mob);   // m_nOneTimeAction = 12, mob sound
```

So on v87: `{0,1,3}` → `OnDie`; `{2,4,5}` → the action-12 "bomb" animation. On v95: `3` gets a
dedicated destruct animation and `5` renders as a plain die. The mapping genuinely inverts
between eras.

**Resolves PRD OQ4.** Every value in the WZ data set (1, 3, 4, 5) is *legal* on every version
in the sense that no client reads past the byte or faults on it — the worst case is a client
that renders an ordinary death instead of a bespoke one, which is a cosmetic difference, not a
desync. The only wire-level hazard in the whole space is the dead-type-4 trailing field
(§2.1). **Therefore the server passes the WZ `action` byte through verbatim and does not
remap per version.** Remapping would mean inventing an animation the data did not ask for;
per-version rendering is the client's business.

**Post-implementation update (fix-dom25 brief):** the byte-stability finding above is not
disputed — it stays true and the IDA evidence stays as written. What changed is *how* the
still-unremapped byte reaches the wire: the backend-guidelines audit (DOM-25) flagged the
atlas-monsters → atlas-channel Kafka seam carrying a raw `CMob::m_nDeadType` byte and the
channel bare-casting it straight onto the wire, instead of going through the tenant
`operations` writer-options table every other resolved code in the service uses (task-102/
task-103 precedent). atlas-monsters now maps the WZ `selfDestruction.action` byte to a
`DeathType*` semantic key (`DISAPPEAR`/`FADE_OUT`/`BOMB`/`DESTRUCT_BY_MISS`/`SWALLOW`/
`SELF_DESTRUCT`) at the point it reads the byte; the Kafka event, the channel consumer, and
the `DestroyMonster` writer's `operations` table (added to all 11 seed templates, all six
entries 0..5, identical across versions) carry only that key. The resolved byte the table
produces is the same byte this section derived — the table is adopted for uniformity with
the rest of the service, not because the per-version rendering facts above changed.

### 2.3 `CMob::TryFirstSelfDestruction` — what the client actually reports

v95 `0x640ee0`:

```c
if ( m_pTemplate->bSelfDestruction && m_pTemplate->bFirstSelfDestruction ) {
    if ( localUser is not in state 0x12/0x13 ) {
        CAvatar::GetBodyRect(localUser, &rcLocalUser, 0);
        for ( i = 0; i < GetCurTemplate(this)->nAttackCount; ++i ) {
            info = GetAttackInfo(GetCurTemplate(this), i);
            if ( !info->nType ) {
                OffsetRect(&rcRange, mobX, mobY);
                if ( IntersectRect(&rcIntersect, &rcRange, &rcLocalUser) ) {
                    COutPacket(232 /* 0xE8 */);
                    Encode4(GetMobID(this));   // <- the mob OBJECT id
                    SendPacket(...); return;
                }
            }
        }
    }
}
```

Two facts the handler design depends on:

- The `mobId` on the wire is `m_dwMobID`, the **object id** — the same identifier
  `MonsterDestroy` carries as `uniqueId` and the same one Atlas calls `uniqueId`. It is **not**
  the template id. The existing codec comment already says "secured mob id"; this pins the
  meaning.
- The client only sends it for a mob whose template carries the self-destruction flags, and
  only when the *local* user's body rect intersects. It is nonetheless a client-controlled
  packet naming an arbitrary id, so the server re-derives every predicate (§4.5).

### 2.4 `selfDestruction.removeAfter` unit

No client-side answer exists: neither `selfDestruction` nor `removeAfter` appears as a string
in the v95 image, and the timer is not a client mechanic — the client learns about a
self-destruct only through `OnMobLeaveField`. The unit is purely a server-side reading of WZ.

**Resolves PRD OQ1 as proposed:** adopt **seconds**, matching Cosmic
(`MapleMap.java:1868`, `SECONDS.toMillis(...)`), and document the inconsistency. Both in-scope
timer mobs (`9300166`, `9300329`) carry `removeAfter = 0`, where the unit is unobservable; the
only mobs where it matters are the two out-of-scope Monster Carnival mobs and `9400566`, whose
HP predicate wins. The reading is therefore a documented convention, not a load-bearing
derivation. If Monster Carnival is ever implemented, the unit is re-opened there.

### 2.5 Do DoT ticks reach `damageCore`?

**Resolves PRD OQ5: no, and they deliberately cannot kill.**
`StatusExpirationTask.processDoTTick` (`status_task.go`) calls
`GetMonsterRegistry().ApplyDamage` directly and caps the tick at `currentHp - 1`:

```go
if totalDamage >= current.Hp() { totalDamage = current.Hp() - 1 }
```

It never enters `damageCore` and never produces a kill. That cap does **not** prevent the
self-destruct threshold from being crossed, because the threshold is above zero: a Boomer at
1801 HP poisoned for 50 lands on 1751 ≤ 1800 and must detonate. So the threshold check has to
be applied at the DoT tick site as well as inside `damageCore` (§4.2), and the kill-prevention
cap stays exactly as it is — poison still cannot reduce a mob to 0 HP.

### 2.6 `atlas-data`'s absent-block sentinel is not what the PRD assumed

`getSelfDestruction` (`services/atlas-data/atlas.com/data/monster/reader.go:206`) returns
`selfDestruction{}` — that is `{action: 0, removeAfter: 0, hp: 0}` — when the node is missing.
The `-1` defaults apply only when the block *exists* but omits a field:

```go
c, err := node.ChildByName("selfDestruction")
if err != nil { return selfDestruction{} }          // {0, 0, 0}
action      := byte(c.GetIntegerWithDefault("action", 0))
removeAfter := c.GetIntegerWithDefault("removeAfter", -1)
hp          := c.GetIntegerWithDefault("hp", -1)
```

PRD FR-1.4 states the absent value is `{0, -1, -1}`. It is not. This matters: under the actual
sentinel, "absent" has `hp == 0`, and `hp > -1` — the PRD's HP-threshold predicate — is **true**
for every ordinary monster in the game. Taken literally, FR-2.1 would detonate every mob at
0 HP through the self-destruct path. See D2 for the resolution.

---

## 3. Architecture

```
atlas-data  ──REST──▶  atlas-monsters                            atlas-channel
  monster info          information.Model.SelfDestruction()        MONSTER_BOMB (0xE8/…)
  (self_destruction)      │                                          │  alive? in-field?
                          ├── damageCore  ── threshold ──┐           ▼
                          ├── DoT tick    ── threshold ──┤   COMMAND_TOPIC_MONSTER
                          ├── SelfDestructTimerTask ─────┤   SELF_DESTRUCT{uniqueId,characterId}
                          └── SELF_DESTRUCT handler ─────┤           │
                                                         ▼           │
                                       Registry.SelfDestruct(uniqueId)  ◀───┘
                                            (atomic, exactly-once)
                                                         │
                                                         ▼
                                       KILLED{..., deathType: <wz action>}
                                                         │
                          ┌──────────────────────────────┴───────────────┐
                          ▼                                              ▼
                 atlas-monster-death                              atlas-channel
                 drops + EXP (tolerates ActorId=0)         MonsterDestroy(uniqueId, deathType)
                                                             version-gated trailing int32
```

Four trigger paths converge on **one** kill primitive. That convergence is the core of the
design: every trigger resolves the animation from server-side WZ data, calls the same atomic
registry transition, and reuses the same post-kill bookkeeping as an ordinary death. There is
no second death path to keep in sync.

### 3.1 Components

| Unit | Purpose | Depends on |
|---|---|---|
| `information.SelfDestruction` (atlas-monsters) | immutable value type + presence predicate | REST DTO |
| `Registry.SelfDestruct` (atlas-monsters) | atomic HP→0 transition, exactly-once | Redis registry |
| `ProcessorImpl.selfDestruct` (atlas-monsters) | resolve action, transition, run the shared kill epilogue | registry, information, producer |
| `SelfDestructTimerRegistry` + `SelfDestructTimerTask` | timer arm/cancel/fire | Redis `TenantRegistry`, processor |
| `SELF_DESTRUCT` command arm | channel → monsters detonation request | command topic |
| `MonsterBombHandleFunc` (atlas-channel) | cheap client-side guards, emit command | session, `LiveMirror`, monster processor |
| `Destroy` codec gate (`libs/atlas-packet`) | version-correct dead-type-4 encoding | tenant |

---

## 4. Decisions

### D1 — `SelfDestruction` is a value type on `information.Model`, populated in `Extract`

```go
type SelfDestruction struct {
    present     bool
    action      byte
    removeAfter int32
    hp          int32
}

func (s SelfDestruction) Present() bool     { return s.present }
func (s SelfDestruction) Action() byte      { return s.action }
func (s SelfDestruction) RemoveAfter() int32{ return s.removeAfter }
func (s SelfDestruction) Hp() int32         { return s.hp }

// OnHpThreshold reports the HP-driven mechanic; OnTimer reports the timer-driven one.
func (s SelfDestruction) OnHpThreshold() bool { return s.present && s.hp > -1 }
func (s SelfDestruction) OnTimer() bool       { return s.present && s.hp <= -1 }
```

`Model` gains `selfDestruction SelfDestruction` + `SelfDestruction()` accessor;
`ModelBuilder` gains `SetSelfDestruction`. Both follow the existing package conventions
exactly (unexported fields, value receivers, builder only exposing what tests need).

Exposing the two predicates as methods rather than leaving callers to write `Present() && Hp > -1`
is deliberate: four call sites need them, and the sentinel arithmetic is the single easiest
thing in this task to get wrong (§2.6).

**Alternative considered — put the check on `Model` directly** (`m.SelfDestructsAt() (uint32, bool)`).
Rejected: the timer path needs `RemoveAfter` and every path needs `Action`, so the value type
has to exist anyway; a second flattened accessor is redundant surface.

### D2 — Fix the absent-block sentinel in `atlas-data` rather than pattern-match the zero value downstream

`getSelfDestruction`'s missing-node return becomes explicit:

```go
if err != nil { return selfDestruction{Action: 0, RemoveAfter: -1, Hp: -1} }
```

and `reader_test.go:1267`'s expectation moves from `{0,0,0}` to `{0,-1,-1}`. `Extract` in
atlas-monsters then derives presence unambiguously:

```go
present := rm.SelfDestruction.Hp > -1 || rm.SelfDestruction.RemoveAfter > -1
```

This makes PRD FR-1.4 true as written and gives one sentinel, used identically by the
producer and every consumer.

**Alternative A — derive presence from the all-zero struct** (`action != 0 || hp != 0 || removeAfter != 0`).
Works today (all twelve real mobs have a non-zero `action`), needs no `atlas-data` change,
and is what the PRD proposed. Rejected: it encodes "no real mob has an all-default block" as
an invisible assumption in a consumer, and it leaves two different absent-shapes in the system
(`{0,0,0}` from a missing node, `{a,-1,-1}` from a present-but-empty one) that only differ by
where they came from.

**Alternative B — add an explicit `present` boolean to the `atlas-data` payload.** Cleanest
semantically, but it widens the REST contract for a fact that a sentinel already carries, and
every consumer would have to keep both in sync. Rejected as YAGNI.

Cost of D2: the `self_destruction` object in `GET /monsters/{id}` changes from
`{"action":0,"remove_after":0,"hp":0}` to `{"action":0,"remove_after":-1,"hp":-1}` for every
non-self-destructing monster. The only consumers are atlas-monsters (new code, expects the new
sentinel) and `atlas-monster-death`'s `information/rest.go:37`, which mirrors the field and
never reads it. Rolling-deploy safe in the recommended order (atlas-data first, atlas-monsters
second), but not for the reason a first read of the predicate suggests: `Hp > -1 || RemoveAfter
> -1` is **not** false under the old pre-D2 `{0,0,0}` sentinel — `0 > -1` is true in Go, so a
new atlas-monsters paired with an old atlas-data reports `Present() == true` for every ordinary
monster (every monster's Hp defaults to 0, not -1, under the pre-D2 contract). The traced blast
radius during that window is narrower than "every monster self-destructs," though: `OnTimer()`
is `present && hp <= -1`, which stays false at `hp == 0`, so there is no timer storm. Only
`OnHpThreshold()` (`present && hp > -1`) goes true, with a threshold of 0 — coinciding with,
not preceding, the ordinary kill point. `Registry.SelfDestruct` is exactly-once (D3) regardless
of which path (ordinary kill or the coincident false threshold) reaches it, so the mob still
dies exactly once. The visible defect in the deploy window is confined to a wrong death-animation
byte (`action` defaults to 0, so the mob's ordinary kill would carry a self-destruct `action` of
0 instead of `DeathTypeFadeOut`), which is why the deploy order (atlas-data before atlas-monsters)
is the mitigation, not a claim that the predicate is false under the old sentinel.

### D3 — One atomic registry primitive, `Registry.SelfDestruct`, owns exactly-once

```go
// SelfDestruct atomically drives the monster to 0 HP. Killed is true for the
// caller that performed the transition and false for every caller that finds
// it already at 0 — which is what makes concurrent triggers (two damage lines,
// a timer racing a bomb report) emit exactly one KILLED.
func (r *Registry) SelfDestruct(t tenant.Model, uniqueId uint32) (DamageSummary, error)
```

Implemented with the same `r.reg.Update` compare-and-set as `ApplyDamage`, capturing
`transitioned := cur.Hp > 0` inside the mutator and returning it as `Killed`. Damage entries
are left untouched: a detonation is not damage and must not rewrite the damage leader.

**Alternative A — reuse `ApplyDamage(characterId, math.MaxUint32, …)`,** which is what
`Kill` (Mortal Blow) does today. Rejected on two counts. First, `ApplyDamage` returns
`Killed: m.Hp() == 0`, which is true for *every* concurrent caller once HP has reached zero —
the second caller clamps `actual` to 0 but still reports `Killed`. That is a latent double-kill
in the Mortal Blow path already; the self-destruct paths (timer sweep on a leader that just
failed over, two clients reporting the same contact) hit it far more often, and FR "exactly one
KILLED" is explicit. Second, it credits the full clamped remainder to a `characterId` that on
the timer and contact paths does not exist.

**Alternative B — a processor-level mutex or a Redis lock per mob.** Rejected: the registry
already provides the atomic primitive; adding a second synchronisation layer for one call site
is the wrong trade.

Note: this design does **not** change `ApplyDamage`'s `Killed` semantics. Doing so would alter
the ordinary damage path and Mortal Blow, which is outside this task's scope. It is called out
in §8 as a known adjacent defect.

### D4 — Threshold evaluation rides the existing information lookup in `damageCore`

`damageCore` already fetches `information.Model` once per damage event for the boss flag and
revives (`processor.go:552-559`). The self-destruct data comes off that same `ma`, so the hot
path takes **zero** additional lookups per hit, and the information cache
(`information/cache.go`) already absorbs the read. The check is a struct-field comparison
against the post-damage HP.

Placement: after the damage loop, after `hasLast`, and **before** the `if killed` block:

```go
sd := ma.SelfDestruction()
if !killed && sd.OnHpThreshold() && int64(last.Monster.Hp()) <= int64(sd.Hp()) {
    p.selfDestructFrom(last.Monster, last.CharacterId, sd.Action(), TriggerThreshold)
    return
}
```

Consequences that fall out of the placement, matching FR-2.2/2.5/2.6:

- Only reached when the attack did **not** already kill — one death, never two.
- Runs once per attack, after all lines are applied — a multi-line attack that crosses on
  line 2 produces one detonation (the loop's `break`-on-kill overkill discard is untouched).
- A mob spawned below its threshold detonates on the next damage event, not at spawn.
- The DAMAGED event has already been emitted, so the channel still writes the final HP-bar
  packet before the destroy — same ordering the ordinary kill path relies on.

### D5 — `selfDestructFrom` is the single kill epilogue, shared with `damageCore`

The post-kill bookkeeping in `damageCore` (cooldown clears, attack-cooldown clears, drop-timer
unregister, status-cancel emits, KILLED emit, registry removal, revive spawn) is extracted into
an unexported helper that takes the monster, the killer id, the boss flag, the revives, and the
death type. `damageCore`'s existing `if killed` block calls it with
`deathType = DeathTypeFadeOut`; `selfDestructFrom` calls it with the WZ action. FR-6.5 is then
satisfied by construction rather than by a parallel copy that can drift.

`selfDestructFrom` itself:

1. `Registry.SelfDestruct` — bail silently if `Killed` is false (already dead / already
   detonated).
2. Unregister the self-destruct timer (idempotent; the timer may be what called us).
3. Resolve the killer: the supplied `characterId` when there is one, else the monster's damage
   leader, else `0` (FR-6.3).
4. Run the shared epilogue with `deathType = action`.
5. `Debugf` with mob id, unique id, trigger (`threshold` / `timer` / `contact`), and action
   (NFR observability).

### D6 — Timer state lives in a new Redis-backed `SelfDestructTimerRegistry` swept by a 1s task

Modeled directly on `DropTimerRegistry` / `DropTimerTask`:

```go
type SelfDestructTimerEntry struct {
    monsterId uint32
    field     field.Model
    action    byte
    fireAt    time.Time
}
```

- `atlasredis.NewTenantRegistry[uint32, storedSelfDestructTimer](rc, "self-destruct-timer", …)`,
  keyed `atlas:self-destruct-timer:<tenantId>:<region>:<major>.<minor>:<uniqueId>` — tenant
  scoping is structural, not a convention (NFR multi-tenancy).
- Armed in `Create` next to the existing friendly/drop-period arm, when `sd.OnTimer()`:
  `fireAt = now.Add(time.Duration(max(removeAfter, 0)) * time.Second)`. `removeAfter <= 0`
  yields `fireAt = now`, which the next sweep tick fires (FR-3.5).
- Unregistered in `Destroy`, in the shared kill epilogue, and by the sweep itself when the mob
  is gone or already dead (FR-3.3).
- `SelfDestructTimerTask` at `time.Second`, registered in `registerSweepTasks` alongside
  `NewDropTimerTask`, therefore leader-gated when leader election is enabled and idempotent
  when it is not (D3 makes a double-fire a no-op).
- Teardown: `DestroyAll` already walks every monster and calls `Destroy`, which unregisters —
  so FR-3.6 needs a test, not a new mechanism.

**Alternative — a `time.AfterFunc` per mob in process memory.** Cheaper to write, but it dies
with the pod, is not tenant-keyed by construction, and is a mechanism the service does not
otherwise use. Rejected: FR-3.4 asks explicitly for the sibling pattern.

**Alternative — reuse `DropTimerRegistry` with a mode flag.** Rejected: two unrelated
lifetimes in one entry, and the drop timer's `RecordHit`/`UpdateLastDrop` mutators are
meaningless here.

### D7 — A new `SELF_DESTRUCT` command, not a reason-carrying `KILL`

`CommandTypeSelfDestruct = "SELF_DESTRUCT"` with
`selfDestructCommandBody{ CharacterId uint32 }`, keyed on the monster's unique id like every
other monster command, added to both the channel's message-contract file
(`kafka/message/monster/kafka.go`) and atlas-monsters' consumer contract
(`kafka/consumer/monster/kafka.go`). The `Command` envelope already carries world / channel /
map / instance / monsterId, so FR-5.6 is satisfied by the envelope.

Adopting the PRD's recommendation, for the reasons it gives plus one more:

- `Kill` fail-closed **drops bosses** (`processor.go:1773`). The Papulatus bombs and
  `9300266`/`9300267` are boss-fight summons; whether *they* are flagged boss is a data
  question we should not have to answer to make the mechanic work.
- `Kill` routes through `damageCore` with `MaxUint32`, which is the wrong primitive (D3).
- The animation must come from server-side WZ data, never from a client-supplied
  discriminator. A `KILL` variant that carried a reason byte would put a
  client-influenceable field one refactor away from the animation.

**Alternative — extend `KILL` with a `reason` field and branch inside `Kill`.** Rejected: it
would need a boss-guard exemption, a different damage primitive, and a different animation
source — i.e. a second implementation wearing the first one's name.

The handler is a thin arm mirroring `handleKillCommand`:

```go
func handleSelfDestructCommand(l, ctx, c command[selfDestructCommandBody]) {
    if c.Type != CommandTypeSelfDestruct { return }
    monster.NewProcessor(l, ctx).SelfDestruct(c.MonsterId, c.Body.CharacterId)
}
```

`Processor.SelfDestruct` performs the **authoritative** checks — monster exists, alive, and its
information carries a `selfDestruction` block — then delegates to `selfDestructFrom` with
`TriggerContact`. All three are silent debug-level rejections (FR-5.3, FR-5.5).

### D8 — The channel validates cheaply and locally; atlas-monsters owns the `selfDestruction` predicate

`MonsterBombHandleFunc` becomes:

1. Decode `mobId` (§2.3: an object id).
2. Fetch the character; reject if `Hp() == 0` — the `character_skill_use.go:59` idiom.
3. `monster.GetLiveMirror().Lookup(tenant, mobId)`; reject on miss, and reject when
   `entry.Field` is not the session's field.
4. `monster.NewProcessor(l, ctx).SelfDestruct(entry.Field, mobId, s.CharacterId())` →
   `SelfDestructCommandProvider`.

No `enableActions` and no failure packet: `CMob::TryFirstSelfDestruction` is fire-and-forget
with no client-side response state (§2.3), so a rejection is a log line and nothing else.

The `selfDestruction`-block check deliberately does **not** happen in the channel, even though
FR-5.3 lists it among the rejections. The channel's monster-information client
(`monster/information/rest.go`) currently carries only `attacks`; widening it plus its cache to
carry `self_destruction` would add a REST dependency to a packet handler for a check
atlas-monsters must repeat anyway — it is the authority on monster lifecycle and it already
holds the data with a cache in front of it. The requirement is met, one hop later, by the
authority. The channel keeps only the two checks it can answer from state it already has.

**Alternative — check everything channel-side and emit only valid commands.** Rejected: it
duplicates the authoritative check, adds a hot-path REST hop, and still cannot close the race
between the check and the command landing.

**Alternative — have the channel write `MonsterDestroy` directly.** Rejected outright by
FR-5.4; the channel is not the authority on monster life-cycle, and drops/EXP would not happen.

### D9 — `deathType` on `KILLED` and `DESTROYED`, with 0 meaning "fade out"

```go
type statusEventKilledBody struct {
    X, Y          int16
    ActorId       uint32
    Boss          bool
    DamageEntries []damageEntry
    DeathType     byte `json:"deathType"`
}
```

Same field on `statusEventDestroyedBody`. `atlas-channel`'s `killForSession` /
`destroyForSession` map the value:

```go
dt := monsterpkt.DestroyTypeFadeOut
if e.Body.DeathType != 0 { dt = monsterpkt.DestroyType(e.Body.DeathType) }
```

An omitting producer (old atlas-monsters, rolling deploy) sends no field → `0` → fade-out,
byte-identical to today (FR-4.2). An old channel reading a new event ignores the field and
renders fade-out — degraded, not broken. Both deploy orders are safe.

The cost is that wire dead-type `0` ("disappear, no animation") becomes inexpressible through
this event. Nothing emits it today, no WZ `action` uses it, and `OnMobLeaveField` treats it as
"remove immediately" rather than as a death — so the collision is with a value this event has
no reason to carry. Documented in the body's comment rather than paid for with a
`*byte`/`omitempty` pointer, which would put a nil-check in two consumers to preserve a value
neither wants.

### D10 — Gate the trailing `int32` in `destroy.go`; pass the byte through unmapped

```go
// hasSwallowCharacterId reports whether CMobPool::OnMobLeaveField reads the
// trailing int32 that follows dead-type 4. Swept across all ten IDBs: absent on
// GMS v48 0x55957b, v61 0x5d4b87, v72 0x6258a1, v79 0x646ff6, v83 0x67961d,
// v84 0x6901b3, v87 0x6b5169; present on GMS v92 0x64bb90, v95 0x658b90 and
// JMS v185 0x6f8a1f. The JMS arm is derived independently, not left to fall out
// of MajorAtLeast(185 >= 92).
func hasSwallowCharacterId(t tenant.Model) bool {
    return (t.IsRegion("GMS") && t.MajorAtLeast(92)) || t.Region() == "JMS"
}
```

`Encode` and `Decode` both consult it; `Encode` takes the tenant from `ctx` the way
`reactor/clientbound/spawn.go:62` does. New constants, each carrying its §2.2 citation:

```go
DestroyTypeDisappear    DestroyType = 0
DestroyTypeFadeOut      DestroyType = 1
DestroyTypeBomb         DestroyType = 2   // v95 CMob::OnBomb
DestroyTypeDestructByMiss DestroyType = 3 // v95 CMob::OnDestructByMiss
DestroyTypeSwallow      DestroyType = 4   // v95 CMob::OnSwallowed
DestroyTypeSelfDestruct DestroyType = 5   // v95 -> CMob::OnDie; v87 -> action-12 bomb
```

The server does **not** remap `action` per version (§2.2). A self-destruct with `action = 4`
on v92+ writes `swallowCharacterId = 0`; on v48–v87 the gate suppresses the field entirely and
the client renders its own dead-type-4 arm (the action-12 bomb on v87). Both are correct wire
shapes; neither desyncs.

**Alternative — refuse to send dead-type 4 and substitute 1 on versions that lack a swallow
concept.** Rejected: it invents a death the data did not specify, and §2.1 shows there is no
version where 4 is unreadable — only versions where it is read differently.

---

## 5. Change surface

### `atlas-data`
- `monster/reader.go:206` — absent-block sentinel `{0,0,0}` → `{0,-1,-1}` (D2).
- `monster/reader_test.go:1267` — expectation updated.

### `atlas-monsters` (the bulk)
- `monster/information/model.go` — `SelfDestruction` value type, field, accessor.
- `monster/information/builder.go` — `SetSelfDestruction`.
- `monster/information/rest.go` — `Extract` maps the DTO, derives `present`.
- `monster/registry.go` — `SelfDestruct` atomic transition (D3).
- `monster/processor.go` — extract the kill epilogue from `damageCore` (D5); threshold check
  in `damageCore` (D4); `SelfDestruct` on the `Processor` interface + impl (D7); arm the timer
  in `Create`; unregister in `Destroy`.
- `monster/status_task.go` — threshold check after a DoT tick (§2.5).
- `monster/self_destruct_timer_registry.go`, `monster/self_destruct_timer_task.go` — new (D6).
- `monster/kafka.go` — `DeathType` on the killed/destroyed bodies; `CommandTypeSelfDestruct`.
- `monster/producer.go` — providers take and emit the death type.
- `kafka/consumer/monster/kafka.go`, `consumer.go` — `SELF_DESTRUCT` arm + registration.
- `main.go` — register `NewSelfDestructTimerTask`.
- `monster/mock/processor.go` — interface extension follow-through.

### `atlas-channel`
- `socket/handler/monster_bomb.go` — validate-and-command (D8).
- `monster/processor.go` + `monster/producer.go` — `SelfDestruct` +
  `SelfDestructCommandProvider`; `monster/mock` follow-through.
- `kafka/message/monster/kafka.go` — `CommandTypeSelfDestruct`, `SelfDestructCommandBody`,
  `DeathType` on the killed/destroyed body mirrors.
- `kafka/consumer/monster/consumer.go:216,314` — pass `DeathType` through (D9).

### `atlas-monster-death`
- No production change expected. `filterByQuestState` already excludes quest drops when the
  quest lookup fails (`processor.go:104-110`), `party.GetByMemberId` failure already yields
  `ownerPartyId = 0` (`:61-64`), `rates.GetForCharacter` already falls back to
  `Default()` on error (`rates/provider.go:27-30`), and `DistributeExperience` over an empty
  entry list iterates nothing. The work here is **tests that pin those behaviours for
  `ActorId = 0`**, per FR-6.4 — "verified, not assumed".

### `libs/atlas-packet`
- `monster/clientbound/destroy.go` — constants + version gate (D10).
- Byte-fixture tests per affected version.

### `atlas-ui`
- None.

---

## 6. Wire contracts

| Contract | Change | Compatibility |
|---|---|---|
| `GET /monsters/{id}` `self_destruction` | absent sentinel `{0,0,0}` → `{0,-1,-1}` | both deploy orders safe (D2) |
| `EVENT_TOPIC_MONSTER_STATUS` `KILLED` / `DESTROYED` | `+ deathType byte` | additive; `0` = fade-out (D9) |
| `COMMAND_TOPIC_MONSTER` | `+ SELF_DESTRUCT{characterId}` | new type; unknown types are already ignored by every arm's type gate |
| `MonsterDestroy` (clientbound) | trailing `int32` gated to GMS ≥ 92 / JMS | fixes a v48–v87 desync that exists today |
| `MonsterBomb` (serverbound) | codec unchanged | behavior only |

---

## 7. Concurrency and ordering

- **Exactly-once death**: `Registry.SelfDestruct`'s compare-and-set is the only place a
  detonation is decided. Two damage events crossing the threshold, a timer firing while a
  contact report is in flight, or two channels each reporting the same bomb all funnel through
  it; the loser sees `Killed == false` and returns.
- **Ordering with the triggering attack**: `SelfDestructCommandProvider` keys on the monster's
  unique id, the same key `DamageCommandProvider` uses, so a contact report lands on the same
  partition as the damage that preceded it and is processed after it.
- **Timer vs. leader failover**: the sweep is leader-gated when leader election is on; if two
  pods ever sweep simultaneously, D3 makes the second a no-op.
- **Event ordering to the client**: unchanged. DAMAGED is emitted before the threshold check,
  so the HP bar still precedes the destroy packet.

---

## 8. Known adjacent defects (not fixed here)

1. `ApplyDamage` reports `Killed: m.Hp() == 0`, which is true for a caller that applied zero
   damage to an already-dead monster. `Damage` and `Kill` guard with an `Alive()` pre-check, so
   the window is narrow, but it is a real double-kill race on the ordinary damage path.
   Out of scope; `SelfDestruct` avoids it by construction (D3) rather than by fixing it.
2. `calculateExperienceStandardDeviationThreshold` divides by `totalEntries` and by
   `len(entryExperienceRatio)`, both zero for an empty damage list, producing `NaN`. It is
   harmless today because the caller then iterates an empty map, and it will stay harmless on
   the `ActorId = 0` path for the same reason. Pinned by a test rather than changed.

---

## 9. Testing strategy

Unit tests, following the repo's Builder pattern for setup (no `*_testhelpers.go`).

**atlas-monsters**
- `information`: `Extract` maps a present block; absent block reports `!Present()`;
  `OnHpThreshold` / `OnTimer` truth table across `{absent, hp-only, timer-only, both}`.
- `damageCore`: Boomer (`hp = 1800`) damaged from full to ≤ 1800 detonates with action 1;
  exactly one KILLED; multi-line attack crossing mid-attack produces one death; a mob with no
  block dies normally at 0 HP (regression); a mob already below threshold detonates on the
  next hit.
- `Registry.SelfDestruct`: second call returns `Killed == false`; damage entries unchanged.
- DoT: a poison tick crossing the threshold detonates; a poison tick that does not cross still
  cannot reach 0 HP (regression on the kill-prevention cap).
- Timer: `hp < 0, removeAfter = 0` fires on the next sweep; a mob killed first leaves no timer
  and the sweep is a no-op; `Destroy` unregisters; `DestroyAll` leaves the registry empty.
- `SelfDestruct` processor: rejects an unknown mob, a dead mob, and a mob with no block.
- Producers: `KILLED`/`DESTROYED` carry `deathType`; ordinary kills carry `1`.

**atlas-channel**
- `monster_bomb.go`: accepts a live in-field mob and emits `SELF_DESTRUCT`; rejects a dead
  character, a mirror miss, and a mob in another field — each with no command emitted.
- Consumer: a `KILLED` with `deathType = 3` writes `MonsterDestroy(uniqueId, 3)`; a `KILLED`
  with the field absent writes fade-out.

**libs/atlas-packet**
- `destroy_test.go`: dead-type 4 encodes 5 bytes on v83 and 9 bytes on v95/v92/JMS185;
  dead-types 1/3/5 encode 5 bytes on every version; decode round-trips symmetrically under
  the same gate.

**atlas-monster-death**
- `CreateDrops` with `killerId = 0`: no error, quest-specific drops excluded, `ownerPartyId = 0`.
- `DistributeExperience` with an empty entry list: no error, no award calls.

---

## 10. Packet-coverage plan

`KILL_MONSTER` / `CMobPool::OnMobLeaveField` current row: v48 ❌, v61–v87 ✅, v92 🟡ᶠ,
v95 ✅, JMS185 ✅.

The gate changes emitted bytes **only** for `destroyType == 4`. That still counts as a wire
change for FR-4.7, so every currently-verified cell whose encoder moved must be re-verified and
its evidence re-pinned: **v61, v72, v79, v83, v84, v87** (trailing field newly suppressed) and
**v92, v95, JMS185** (behaviour unchanged, but the encode path is now conditional). v92 is
`🟡ᶠ` and needs its tier-1 byte fixture regardless. v48 is `❌` and out of scope to promote
here.

`packet-audit matrix` plus the `--check` passes must exit 0 before the PR, per
`docs/packets/PROCESS.md`.

---

## 11. Risks

| Risk | Mitigation |
|---|---|
| PRD FR-2.1's literal predicate (`hp > -1`) detonates every mob under the real sentinel | D2 fixes the sentinel; `OnHpThreshold()` is the only predicate any caller uses; regression test on a no-block mob |
| Rolling deploy renders every death as "disappear" | `0` is mapped to fade-out at the consumer, not passed through (D9); test covers the absent field |
| A v83 client desyncs on dead-type 4 | Already broken today; D10's gate is the fix, with per-version byte fixtures |
| The Papulatus bombs never spawn, so the mechanic is unobservable (PRD OQ7) | Generic summon exists (`executeSummon`, `processor.go:1306`) and calls `Create` per `summons` entry; whether Papulatus's mob-skill data lists `8500003`/`8500004` is unverified. Live-channel check is an explicit acceptance item — if they do not spawn, that is reported, not silently skipped. A Boomer (`5100002`) is the fallback live target and needs no boss content. |
| Timer sweep fires against a stale entry after a pod restart | Sweep re-reads the monster registry and unregisters when the mob is gone or dead, exactly as `DropTimerTask.processEntry` does |

---

## 12. Explicitly out of scope

Per the PRD: `MOB_TIME_BOMB_END`; Papulatus boss scripting beyond the bomb mobs; Monster
Carnival as content (`9400547` / `9400550` work if they spawn, but are not test targets); the
top-level `info/removeAfter` despawn field; any change to how ordinary monsters die. Added by
this design: fixing `ApplyDamage`'s `Killed` semantics (§8.1), and re-opening the
`removeAfter` unit question (§2.4), which only Monster Carnival can settle.
