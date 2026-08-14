# Player-cast mists II — Design

Task: task-218-player-cast-mists
PRD: [`prd.md`](./prd.md) (approved)
Status: Draft for review
Created: 2026-08-12

---

## 0. Evidence gathered during design

Everything in §1 was read from live `atlas-data` (namespace `atlas-main`,
`GET /api/data/skills/{id}` per tenant) or decompiled from the checked-in
IDBs. Nothing here is recalled from general MapleStory knowledge. Where a
value could not be established, it is labelled **unverified** and carries the
check needed to settle it.

Live tenant ids used (from `GET /api/tenants`, re-listed at design time — they
change on reprovision):

| version | tenant id |
|---|---|
| GMS 48 | `e1f06ae2-80c1-47f7-bb6f-38a9f50d23dd` |
| GMS 61 | `0d250dc9-64c4-45ae-8bc2-fc0a9cdb5578` |
| GMS 72 | `48d415ca-59de-4953-9aed-0c4156a09bc9` |
| GMS 79 | `92adbe47-5ada-4f3b-8224-f58c80a4a2d5` |
| GMS 83 | `ec876921-c363-4cc6-9c51-5bb8d57f9553` |
| GMS 84 | `4936dff2-7121-4f46-b9eb-1ae541f4a85f` |
| GMS 87 | `86da65d2-b9fa-4176-985a-6a5df586220c` |
| GMS 92 | `db1dbfb3-4345-4731-9223-c40b0c7f6457` |
| GMS 95 | `c794c706-aea3-4882-90a6-a3b7ee314f52` |
| JMS 185 | `abedf3b4-1d7c-4b3b-bc52-70f62ab09418` |

GMS 12 has no live baseline tenant, so no version-12 statement below is a live
observation; the v12 column is driven purely by the checked-in binding tables.

---

## 1. Skill data findings (FR-1)

### 1.1 Availability — live sweep, all 10 tenants × 4 skills

`GET /api/data/skills/{id}` HTTP status:

| skill | 48 | 61 | 72 | 79 | 83 | 84 | 87 | 92 | 95 | jms185 |
|---|---|---|---|---|---|---|---|---|---|---|
| Smokescreen 4221006 | 200 | 200 | 200 | 200 | 200 | 200 | 200 | 200 | 200 | 200 |
| Flame Gear 12111005 | 404 | 404 | 200 | 200 | 200 | 200 | 200 | 200 | 200 | 200 |
| Poison Bomb 14111006 | 404 | 404 | 200 | 200 | 200 | 200 | 200 | 200 | 200 | 200 |
| Recovery Aura 22161003 | 404 | 404 | 404 | 404 | 404 | 200 | 200 | 200 | 200 | 200 |

This **confirms the PRD §7 availability table**, including its "expected"
Recovery Aura row: gms 84/87/92/95 and jms 185, and nowhere else. PRD FR-0.5's
correction branch does not fire.

`maxLevel`: Smokescreen 30, Flame Gear 30, Poison Bomb 30, Recovery Aura 15.

### 1.2 Per-skill effect fields (FR-1.1)

Values below are `effects[0]` (level 1) and `effects[-1]` (max level) as served.

**Smokescreen 4221006** — identical on gms 48/61/72/79/83/84/87/92 and jms 185:
`duration` 31000→60000 ms, `lt(-110,-82)` → `rb(110,83)` at L1 widening to
`lt(-200,-150)`/`rb(200,150)` at L30, `cooldown` 600, `MPConsume` 16→45.
GMS 95 differs: same 31000 ms but `lt(-250,-150)`/`rb(250,150)` and
`cooldown` 360. Non-zero lifetime and non-degenerate rectangle at every level
on every version. **FR-1.1 satisfied with no reader work.**

**Flame Gear 12111005** — gms 72/79/83/84/87/92 and jms 185: `duration`
4000→40000 ms, `lt(-200,-250)`/`rb(200,30)`, `prop` 0.51→0.80,
`MPConsume` 21→50. GMS 95 diverges: `duration` 11000, `mobCount` 8,
`damage` 124, `dot` 42, `dotInterval` 1, `dotTime` 3.

**Poison Bomb 14111006** — gms 72/79/83/84/87/92 and jms 185: `duration`
4000→40000 ms, `lt(-100,-82)`/`rb(100,83)`, `damage` 104→220, `mobCount` 6,
`prop` 0.51→0.80. GMS 95: `damage` 83, `dot` 62, `dotInterval` 1, `dotTime` 4.

**Recovery Aura 22161003** — identical on gms 84/87/92/95 and jms 185:
`duration` 30000 ms at every level, `lt(-200,-125)`/`rb(200,30)`,
`cooldown` 60, `HPConsume` = `MPConsume` = 18→34, `hp`/`mp`/`hpR`/`mpR` all 0,
`dot`/`dotInterval`/`dotTime` all 0, `y` 0 at every level, and **`x` 38 at L1
rising to 80 at L15** — the only per-level-varying magnitude node on the skill.
Its in-game description, served verbatim by atlas-data, is:

> "Creates a recovery aura around you for 30 seconds. The **MP** of all party
> members inside the aura will be continuously restored. [Cooldown: 1 min]"

### 1.3 Answer to open question 5 — Recovery Aura's magnitude node (FR-1.2)

`x` is the magnitude; the effect it drives is **MP only**, not HP. Evidence:
the description names MP explicitly; `hp`, `mp`, `hpR`, `mpR` are all zero at
every level on every version; `x` is the only node that varies with level.

`x` is **already exposed** on the atlas-channel effect model
(`data/skill/effect/model.go`, `X()`), read unscaled at
`services/atlas-data/atlas.com/data/skill/reader.go:266`
(`SetX(int16(node.GetIntegerWithDefault("x", 0)))`). **No new reader work and
no new REST field are required for FR-1.2.** The design's one deviation from
the PRD here is scope-reducing: FR-1.2 anticipated plumbing a new field; none
is needed, and the "normalise units at the reader" clause is vacuous because
`x` carries no unit conversion — it is a raw amount, exactly as
`mprecovery`'s handler already consumes it
(`skill/handler/mprecovery/mprecovery.go`, `e.X()`/`e.Y()`).

**Unverified:** whether `x` is an absolute MP amount or a percentage of max MP.
WZ alone cannot settle this — the node is a bare integer with no sibling rate
node, and no version exposes a second candidate. This design treats it as
**absolute MP**, consistent with every other `x` consumer in the repo, and
FR-5.1's acceptance test asserts the absolute reading. Settling it definitively
requires an in-client observation; that is called out in §9 as the single
open in-game check.

### 1.4 Answer to open question 4 — Flame Gear's monster status (FR-1.3)

**Flame Gear applies `POISON`, with the target-derived magnitude, on every
version — same as Poison Mist and Poison Bomb.** This is a WZ-derived
conclusion, not an analogy:

1. `monsterStatus` is `{}` (empty) for 12111005 on **every** version served.
   The WZ names no monster status at all, so there is no other status to pick.
2. `atlas-monsters` defines exactly two DoT statuses —
   `StatusPoison` and `StatusVenom`
   (`services/atlas-monsters/atlas.com/monsters/monster/status.go:20,23`).
   `VENOM` carries a caster-supplied magnitude and is the Night Lord Venom
   stat; using it for Flame Gear would put the wrong monster temporary-stat on
   the wire. No third status exists, and inventing one would require a
   client-verified MobStat bit that no version's WZ asks for.
3. The only caster-side magnitude candidate is `dot`, and **`dot` is 0 for
   12111005 on gms 72/79/83/84/87/92 and jms 185** — six of the seven versions
   that bind the skill. A `dot`-driven magnitude is therefore not portable, and
   a per-version split (POISON below v95, `dot` at v95) would mean the skill
   behaves differently on one version for no player-visible reason.

**Consequence for FR-6.3:** Flame Gear's magnitude is target-derived, so it is
sent as `0` and `atlas-monsters` resolves it per monster at apply time via
`monster.ResolvePoisonDamage` — identical to Poison Mist and Poison Bomb. No
`atlas-monsters` change is required. PRD §7's "may require a status type that
does not yet exist" row resolves to **no change**.

The v95 `dot`/`dotTime` data is recorded here as deliberately unused, not
overlooked. If a later task wants caster-derived burn damage, it must first add
a status `atlas-monsters` and the client both understand; that is out of scope.

### 1.5 Data defect to report (FR-1.4)

Live v95 rows serve `dotInterval: 1` and `dotTime: 3`. The reader has scaled
both to milliseconds since task-200
(`skill/reader.go:263-264`, `* 1000`). The live rows are therefore **stale,
pre-task-200 ingests** — this is the known
"effects ingested, not re-parsed" trap: atlas-data computes effect documents
once in the ingest worker and the REST pods only serve stored rows, so
redeploying the REST image never refreshes them.

This does not block task-218 — the chosen design reads none of `dot`,
`dotInterval`, or `dotTime` — but it is a real defect and is reported here
rather than worked around. **No fallback is hard-coded anywhere in this
design.** Fixing it means a re-ingest plus an atlas-data REST pod restart
(the `RegStorage` in-memory cache has no TTL and no invalidation), and belongs
to whichever task next depends on those fields.

A second, smaller observation for the plan phase: `damage` defaults to 100,
`attackCount` to 1, and `mobCount` to 1 when the WZ node is absent
(`skill/reader.go:197,268,270`). So `damage:100, attackCount:1, mobCount:1` in
a REST response means "no attack nodes present", not "a 100% attack". This
matters in §4.

---

## 2. Client evidence

### 2.1 All four skills are AFFECTEDAREA skills

`CAffectedAreaPool::AffectedAreaAnimationCreated` (GMS v95 @0x4372c0)
dispatches purely on `nSkillID`. Decompiled, its arms are:

| nSkillID | arm |
|---|---|
| 130, 131 | mob-skill arms; write `pa->nDamage = a[nSLV-1].nX` |
| 2111003 (Poison Mist) | `GetTileUOL` → `MakeLayer_Fog` |
| **4221006 (Smokescreen)** | same arm as 2111003 (`MakeLayer_Fog`) |
| **14111006 (Poison Bomb)** | same arm as 2111003 (`MakeLayer_Fog`) |
| **12111005 (Flame Gear)** | `GetTileUOL` → `MakeLayer_Fog_OneTile` |
| **22161003 (Recovery Aura)** | same arm as Flame Gear |
| 32121006 | its own arm (out of scope) |

(IDA renders three of these immediates as pseudo-symbols — `loc_40684E` =
0x40684E = 4221006 and `unk_B8CC9D` = 0xB8CC9D = 12111005. Both were converted
by hand and cross-checked against the skill ids.)

Every arm resets `pa->tEnd = pa->tStart + 1000 * SKILLLEVELDATA::tTime` when
`bResetEndTime` — the client computes lifetime from **its own** WZ. This is
the standing reason FR-8.2 rejects rather than clamps.

Note also that none of the four arms writes `nDamage`. Only the mob-skill arms
(130/131) do. That is the same uninitialised-`nDamage` hazard task-200
diagnosed, and it is why §3.2's `nType` derivation must never emit `0` for a
character-owned mist.

### 2.2 Answer to open question 2 — Smokescreen is party-scoped (FR-4.6)

`CAffectedAreaPool::IsSmokeAreaByPoint(dwCharacterID, adwPartyMemberID, ptUser,
nPhase)` (GMS v95 @0x434f40), decompiled, accepts an affected area iff **all**
of:

1. `p->nType == 2`;
2. `tCur - p->tStart >= 0`;
3. `p->nPhase == nPhase`;
4. `p->dwOwnerID` appears in `adwPartyMemberID`, **or**
   `p->dwOwnerID == dwCharacterID`;
5. `PtInRect(&p->rcArea, ptUser)`.

Condition 4 settles FR-4.6: the client protects **the caster and the caster's
online party members only**, not everyone standing in the cloud. The server
must match, or a non-party player will render as unharmed on their own client
while the server keeps killing them (or vice versa).

Its only two callers are `CUserLocal::SetDamaged` (@0x9345aa) and
`CUserLocal::Update` (@0x938b21). The party array passed at the `SetDamaged`
call site is filled immediately before by
`CWvsContext::GetOnlinePartyMemberID` (@0x93455c) — so it is **online party
membership evaluated at hit time**, not a snapshot from cast time.

### 2.3 What the client does with a smoke hit

At `SetDamaged+0x1ef` (0x9345af): `test eax, eax; jnz loc_93651F` — a positive
smoke result jumps straight to the function epilogue, **before** the character
data fetch, the miss roll, Power Guard, Meso Guard, Achilles, Magic Guard, and
before the damage packet is built. The client takes zero damage and sends
nothing.

Two consequences:

- The server-side protection is a **short-circuit**, not another term in the
  mitigation chain. Modelling it as a percentage reduction inside
  `computeMitigation` would diverge from the client.
- Because the honest client sends nothing, the server-side check exists purely
  to stop a **crafted** client from claiming damage while standing in smoke, or
  from being damaged by a server-initiated source. FR-4.2 is therefore not
  optional garnish — it is the whole point of the server-side path.

### 2.4 `nPhase` — a bounded, recorded risk

Condition 3 above compares `p->nPhase` against the value `SetDamaged` reads
from `CUserLocal+0x2E18` (`mov edi, [esi+2E18h]`, @0x934573) — the local user's
field phase. Atlas sends `nPhase = 0` on every `AffectedAreaCreated`
(`kafka/consumer/mist/consumer.go`, `mistPhase`), and the legacy versions omit
the field entirely.

So client-side smoke recognition works **iff the local user's phase is 0**,
which is the case for every map Atlas serves today (Atlas models no field
phase at all). This is not a new risk introduced by task-218 — it applies to
every mist Atlas creates — but it is newly load-bearing, because Smokescreen is
the first mist whose client behaviour depends on the `nPhase` comparison.

Recorded rather than mitigated: mitigating would mean modelling field phase,
which is out of scope. **The server-side protection in §5 does not depend on
`nPhase` at all**, so a phase mismatch degrades to "client renders a hit the
server refuses to apply", not to "player takes damage they shouldn't".

---

## 3. Contract and model changes

### 3.1 `EffectKind` gains two values (FR-2.1, FR-2.2)

In `services/atlas-maps/atlas.com/maps/kafka/message/mist/kafka.go` **and its
atlas-channel mirror**:

```go
EffectKindDisease        = "DISEASE"
EffectKindDamageOverTime = "DAMAGE_OVER_TIME"
EffectKindProtection     = "PROTECTION"   // new — Smokescreen
EffectKindRecovery       = "RECOVERY"     // new — Recovery Aura
```

The empty-string default keeps meaning `DISEASE` + `CHARACTER` targeting, so
the pre-task-200 `atlas-monsters` `AREA_POISON` producer is untouched
(FR-2.3).

### 3.2 New magnitude field on `CreateCommandBody` (FR-2.4)

```go
RecoveryMp int32 `json:"recoveryMp"`   // per-tick MP restored; RECOVERY only
```

One field, named for what it is. `DiseaseValue` is **not** reused — the PRD
prohibits it and the two have different resolution rules (`DiseaseValue` is
target-derived and overwritten downstream; `RecoveryMp` is caster-derived and
authoritative). No `RecoveryHp` field is added: §1.3 established the skill
restores MP only, and an always-zero field would invite a future reader to
assume HP recovery is wired when it is not.

Party scoping for RECOVERY travels as:

```go
PartyMemberIds []uint32 `json:"partyMemberIds"`   // RECOVERY only
```

See §6 for why this is a cast-time snapshot rather than a live lookup.

`mist.Mist` gains `recoveryMp` and `partyMemberIds` private fields with getters,
set through the existing builder via one grouped setter
(`SetRecovery(mp int32, partyMemberIds []uint32)`), matching the existing
`SetKinds` / `SetDisease` grouping style. `PartyMemberIds()` returns a
defensive copy — the tick fans out across goroutines (NFR-3) and a shared
backing array is exactly the kind of shared mutable state `ForEachInMap`'s
parallelism punishes.

### 3.3 `effectKind` is added to the `MIST_CREATED` event body

```go
type CreatedBody struct {
    ...
    EffectKind string `json:"effectKind"`
}
```

atlas-channel needs to recognise a protection mist to populate its registry
(§5). It could infer this from the existing `Type` field (`nType == 2`), but
that would couple channel-side logic to a client wire value — precisely the
coupling `AffectedAreaTypeFor`'s doc comment exists to prevent. Carrying the
domain concept is one additive string and keeps `nType` a pure render detail.

### 3.4 `nType` derivation (FR-3.1, FR-3.2, FR-3.4)

```go
// AffectedAreaTypeSmoke is 2 because the client's smoke lookup keys on it:
// CAffectedAreaPool::IsSmokeAreaByPoint (v95 @0x434f40) rejects any area whose
// nType != 2, and v83 CAffectedAreaPool::Update (@0x43109f) gates the fade-out
// animation on the same value.
AffectedAreaTypeSmoke = int32(2)

func AffectedAreaTypeFor(ownerType string, effectKind string) int32 {
    if ownerType != OwnerTypeCharacter {
        return AffectedAreaTypeMobSkill // 0 — unchanged, load-bearing
    }
    if effectKind == mist.EffectKindProtection {
        return AffectedAreaTypeSmoke // 2
    }
    return AffectedAreaTypeUserSkill // 1
}
```

Four outcomes, all covered by one table-driven regression test (FR-3.4):

| ownerType | effectKind | nType |
|---|---|---|
| MONSTER | (any, incl. empty) | 0 |
| CHARACTER | PROTECTION | 2 |
| CHARACTER | DAMAGE_OVER_TIME | 1 |
| CHARACTER | RECOVERY / DISEASE / empty | 1 |

`nType` stays derived inside atlas-maps and stays off `COMMAND_TOPIC_MIST`
(FR-3.3). The signature change is the *only* churn: every existing caller
passes the mist's effect kind, which the create path already has in hand.

### 3.5 Answer to open question 8 — add a mist-contract mirror guard

**Yes.** `tools/mist-contract-mirror-guard.sh`, modelled directly on
`tools/trade-contract-mirror-guard.sh`: diff
`services/atlas-maps/atlas.com/maps/kafka/message/mist/kafka.go` against
`services/atlas-channel/atlas.com/channel/kafka/message/mist/kafka.go` from the
`package` clause onward, allowing only the leading doc comment (which names the
mirror direction) to differ.

The two files live in separate Go modules, so a json tag changed in one and not
the other compiles clean and decodes into a zero-valued body at runtime. This
task adds three keys to that contract in one change — exactly the moment the
divergence hazard is highest. Add a corresponding numbered entry to CLAUDE.md's
Build & Verification list.

Caveat for the plan phase: the two files are **not** byte-identical today
(atlas-channel's copy is a partial mirror). The guard task must first bring the
channel copy to a full mirror, or the guard must be written to compare only the
declarations present in both. Prefer the former — a partial mirror is how the
divergence starts.

---

## 4. Answer to open question 6 — which packet delivers each cast (FR-7.2)

| skill | registry | evidence |
|---|---|---|
| Flame Gear 12111005 | `RegisterAttackCast` | `prop` 0.51→0.80 on gms 72–92/jms; `mobCount` 8 and `damage` 124 at v95 — real attack nodes, not reader defaults |
| Poison Bomb 14111006 | `RegisterAttackCast` | `mobCount` 6 and `damage` 104→220 on every version; `prop` 0.51→0.80 pre-v95 |
| Smokescreen 4221006 | `Register` (USE_SKILL) | `damage` 100, `attackCount` 1, `mobCount` 1, `prop` 0 on **every** version — i.e. every attack node absent, per §1.5's default table |
| Recovery Aura 22161003 | `Register` (USE_SKILL) | same all-defaults signature on every version that binds it |

The discriminator is the same one task-200 used, applied with §1.5's default
table so that "absent" is not misread as "present and equal to the default" —
the trap that makes `damage:100` look like a real attack multiplier. Note
Poison Mist itself serves `damage:100, mobCount:1` on gms 83/87/92 yet is an
attack skill; its `prop` 0.41 is what distinguishes it. The rule applied is:
**any** of `prop != 0`, `mobCount != 1`, or `damage != 100` on **any**
provisioned version ⇒ attack-delivered. Flame Gear, Poison Bomb, and Poison
Mist each trip it; Smokescreen and Recovery Aura trip none of the three on any
of the ten live tenants.

This also lands the right cost behaviour for free, which is the failure mode
FR-7.1 warns about:

- Flame Gear / Poison Bomb on `RegisterAttackCast`: `processAttack` charges the
  generic MP cost, exactly as for Poison Mist.
- Smokescreen / Recovery Aura on `Register`: `UseSkill` applies MP consume, HP
  consume, and cooldown *before* the handler lookup
  (`skill/handler/common.go:140-164` then `:201`). Recovery Aura's
  `HPConsume` 18→34 and 60 s cooldown, and Smokescreen's 600 s (360 s at v95)
  cooldown, are all handled by the generic path — the handlers charge nothing
  themselves.

**Confidence.** This is a WZ-shape argument, and it is the same argument the
repo already relies on. It is strong for Flame Gear and Poison Bomb (positive
evidence of attack nodes) and is an argument from total absence for Smokescreen
and Recovery Aura. The absence is uniform across ten independently-ingested
tenants, which is what makes it more than a single-row observation. The
low-cost confirmation, listed in §9, is to cast each once against a live
channel and read the log line — a mis-registered handler is loudly silent
(never fires) rather than subtly wrong.

---

## 5. Answer to open question 1 — Smokescreen mechanism (FR-4.4)

**Chosen: (c) a channel-local mist registry consulted on the damage path.**
This is a variant of the PRD's option (b) with the network hop removed. The
PRD names (a) buff-mediated as the recommended default and explicitly leaves
the decision to this phase; the recommendation is declined, with reasons.

### 5.1 The mechanism

`atlas-channel` already consumes `EVENT_TOPIC_MIST`
(`kafka/consumer/mist/consumer.go`) and already filters to its own
`(world, channel)` via `sc.Is(...)`. Today it translates each event into an
`AffectedAreaCreated` / `AffectedAreaRemoved` broadcast and keeps nothing.

Add a small in-process registry — same shape as the existing atlas-maps mist
registry, tenant-keyed, `sync.RWMutex`-guarded, singleton via `sync.Once`:

- `handleMistCreated` inserts protection mists (`EffectKind == PROTECTION`)
  keyed by mist id: field, owner id, absolute rect, expiry.
- `handleMistDestroyed` removes by mist id.
- Entries past their expiry are treated as absent on read and pruned lazily on
  the next write, so a dropped `MIST_DESTROYED` cannot leave a permanently
  protective rectangle.

`processDamageTaken` gains one check, placed **before** `computeMitigation`:

```go
if deps.inProtectiveMist(f, c.Id(), c.X(), c.Y()) {
    l.Debugf("Character [%d] shielded by a protection mist in map [%d]; damage [%d] dropped.",
        c.Id(), f.MapId(), raw)
    return
}
```

`inProtectiveMist` returns true iff some live protection mist on the character's
field contains `(x, y)` **and** the mist's owner is the character or one of the
character's online party members — the exact conjunction §2.2 read out of
`IsSmokeAreaByPoint`. Party membership comes from the existing
`atlas-channel/party` processor, which the channel already uses for party HP
sync, so no new cross-service dependency is introduced.

### 5.2 Why not (a), buff-mediated

Three concrete defects, each verified rather than speculative:

1. **Wrong scope, unfixably.** The atlas-maps character tick
   (`tasks/mist_tick.go`, `tickCharacters`) iterates every character in the
   field and has no party knowledge — atlas-maps has no party client at all
   (`grep -ril party services/atlas-maps/...` returns only unrelated hits in
   `data/map/monster` and `mist/model.go`). Buff-mediated protection would
   shield **everyone** inside the cloud, contradicting §2.2's client behaviour.
   Fixing that means giving atlas-maps a party REST client — a new
   service-to-service edge added on the losing branch.
2. **Tick-granularity lag on both edges.** FR-4.3 requires protection to end
   when a character walks out. A buff applied on a 3 s re-apply cadence and
   expiring on its own duration leaves a window in which a character outside
   the rectangle is still protected — the exact inverse of the requirement, and
   an exploitable one (step out, get hit, take nothing).
3. **It puts a stat on the wire.** Every `COMMAND_TOPIC_CHARACTER_BUFF` apply
   becomes a client `SecondaryStat`. The closest existing candidate,
   `TemporaryStatTypeNotDamaged` (`NOT_DAMAGED`,
   `libs/atlas-constants/character/temporary_stat.go:93`), has client-side
   semantics on every version that Atlas has never exercised. §2.3 shows the
   client already handles smoke locally from its own affected-area list — it
   needs **no** buff to render or behave correctly. Sending one is unverified
   client surface for zero client benefit.

This also answers **open question 3**: no protective temporary-stat type needs
to be added, because no temporary stat is used. `NOT_DAMAGED` exists and was
considered; it is deliberately not used, for reason 3.

### 5.3 Why not (b) as written, and what (c) costs

Option (b)'s synchronous `atlas-channel → atlas-maps` REST call on the hot
damage path is the right *semantics* with the wrong *plumbing*: a per-hit
network round trip on the most latency-sensitive path in the channel. (c)
keeps the semantics and drops the call to a map read under an `RWMutex`.

Costs, stated plainly:

- **Restart gap.** The mist consumer uses
  `consumer.SetStartOffset(kafka.LastOffset)`, so a channel that restarts
  mid-mist never learns about mists created before it came up and will not
  protect against them. This is not a regression: the same restart already
  loses the `AffectedAreaCreated` broadcast, so those mists are invisible to
  every client on that channel too. Worst case is bounded by the longest
  Smokescreen lifetime (60 s at L30).
- **Duplicated containment logic.** The rect test now exists in both
  atlas-maps (`Mist.Contains`) and atlas-channel. Mitigated by using the same
  inclusive-edge convention and asserting it in both services' tests; the
  alternative (one authority, queried remotely) is option (b)'s round trip.

### 5.4 Worst-case lag (FR-4.3) and composition (FR-4.5)

**Worst-case lag on entering or leaving the cloud is one damage event** — the
check reads the character's current position and the live registry at the
moment the hit is processed. There is no tick quantisation. The only latency is
the Kafka propagation of `MIST_CREATED` / `MIST_DESTROYED` from atlas-maps to
the channel, which bounds how quickly protection *starts* and *stops*, not how
quickly a moving character enters or leaves it.

FR-4.5 is satisfied structurally rather than by careful arithmetic: the check
returns before `computeMitigation` is ever called, so Power Guard, Mana
Reflection, and Meso Guard amounts are not computed from zeroed damage — they
are not computed at all, matching §2.3's client short-circuit.

FR-4.2 holds because nothing in the check reads the wire: the position comes
from the server's character model, the rectangle from the mist event, the party
from the party processor. A client claiming a hit while inside a protection
mist gets it dropped; a client claiming protection it does not have has no
field to claim it in.

---

## 6. Recovery Aura (FR-5)

### 6.1 Where the tick lives

The tick stays in `atlas-maps`, in the existing `tickCharacters` structure,
dispatched on `EffectKind`:

```
tickOneMist
 ├─ TargetKind MONSTER  → tickMonsters      (unchanged)
 └─ TargetKind CHARACTER
     ├─ EffectKind RECOVERY   → tickRecovery       (new)
     ├─ EffectKind PROTECTION → no-op (see §6.4)
     ├─ EffectKind DISEASE / "" → tickCharacters   (unchanged)
     └─ anything else → warn, no effect            (FR-2.5)
```

`tickRecovery` reuses `charsInField` + `posLookup` + `Mist.Contains` exactly as
`tickCharacters` does, then emits one `COMMAND_TOPIC_CHARACTER` `CHANGE_MP`
command per eligible character
(`ChangeMPCommandBody{ChannelId, Amount: int16(m.RecoveryMp())}`), keyed on the
character id, through the same `message.Buffer` / `message.Emit` batching. This
satisfies FR-5.4: atlas-maps emits a command to the owning service and mutates
nothing itself, the same discipline `tickCharacters` and `tickMonsters` follow.

`atlas-maps` already mirrors two command envelopes locally
(`buffCommand`, `monsterCommand`); the `CHANGE_MP` envelope is a third mirror
of the same shape. `COMMAND_TOPIC_CHARACTER` is not a shared-handler topic in
the way `COMMAND_TOPIC_MONSTER` is, but the mirror must still be key-for-key
faithful to
`services/atlas-channel/atlas.com/channel/kafka/message/character/kafka.go:70`.

### 6.2 Party scoping (FR-5.2) — snapshot at cast time

`PartyMemberIds` is resolved by the **handler in atlas-channel**, which has the
party processor, and carried on the CREATE command. `tickRecovery` heals a
character only if their id is in that set (the set always includes the caster).

The alternative — a live party lookup per tick — would need a party client in
atlas-maps, the same new edge §5.2 rejected. The cost of the snapshot is a
30 s staleness window: someone who joins the party mid-aura is not healed, and
someone who leaves still is. This is acceptable here and **not** acceptable for
Smokescreen, and the difference is principled: the client independently
evaluates live party membership for smoke (§2.2), so a server snapshot would
visibly disagree with the player's own screen. Nothing client-side evaluates
Recovery Aura membership, so the snapshot is unobservable except as a 30 s
edge case.

### 6.3 Cadence, and FR-5.3's caps

No WZ node carries a recovery cadence: `dot`, `dotInterval`, and `dotTime` are
all 0 for 22161003 on all five versions that serve it (§1.2). The cadence is
therefore a documented server constant, the same class of decision as
`PlayerMistTickIntervalMs`, and the handler sets `TickIntervalMs` to the shared
**3000 ms** player-mist cadence. Over the fixed 30 000 ms lifetime that is
10 ticks, i.e. 380 MP at L1 (`x` 38) and 800 MP at L15 (`x` 80).

FR-5.3's "never exceed max MP, never affect a dead character" is satisfied by
the owning service, not re-implemented in the tick: `atlas-character`'s
`ChangeMP` already clamps to max MP, and a dead character is at 0 HP with the
existing death path owning the transition. Re-clamping in atlas-maps would need
the character's max MP — a second REST call per character per tick — and would
be a second authority for a rule that already has one. The plan phase must
**verify** the clamp exists in `atlas-character`'s `ChangeMP` before relying on
it; if it does not, the clamp belongs there, not here.

### 6.4 Protection mists do not tick

A Smokescreen mist is created with `TickIntervalMs: 0`. `Mist.ShouldTick`
returns false for a non-positive interval (`mist/model.go`), so `tickOneMist`
returns before the effect-kind switch and the mist still expires normally via
the `Expired()` branch. The `PROTECTION` arm in the switch above therefore
exists only as an explicit, commented no-op so that a future non-zero interval
does not silently fall through to the `DISEASE` default and start diseasing
everyone in a smoke cloud.

---

## 7. Handler factoring (FR-7.6) and validation (FR-8)

### 7.1 Shared validation and emit helper

All five mist skills (the four new ones plus Poison Mist) share the same
cast-time shape. Extract into a new `skill/handler/mistcast` package:

```go
// Params is everything a mist cast differs by.
type Params struct {
    SkillName   string   // for log lines
    TargetKind  string
    EffectKind  string
    Disease     string   // "" for non-disease kinds
    TickMs      int64    // 0 for PROTECTION
    RecoveryMp  int32    // 0 unless RECOVERY
    PartyIds    []uint32 // nil unless RECOVERY
}

// Cast validates the effect, loads the caster, and emits CREATE.
// Returns nil on every rejection path — there is no MP/cooldown rollback.
func Cast(l, ctx, f, characterId, skillId, skillLevel, e, p Params) error
```

`Cast` runs the four rejections in `poisonmist.go`'s order — non-positive
lifetime; lifetime shorter than one tick; degenerate rectangle
(`rb.X <= lt.X || rb.Y <= lt.Y`); lifetime above `MaxPlayerMistDurationMs` —
each logging a warning naming the skill and the reason (FR-8.1), rejecting
rather than truncating the ceiling (FR-8.2), emitting nothing and returning nil
(FR-8.3). Caster-load and emit failures log at error and return nil (FR-8.4).

The sub-tick check is skipped when `TickMs == 0` (protection mists, which never
tick); the other three apply to all five skills.

`MaxPlayerMistDurationMs` (300 000 ms) moves to `mistcast` and stays 300 000:
the largest legitimate lifetime across all five skills and all versions is
Smokescreen's 60 000 ms at L30, so the ceiling remains 5× the largest real
value and can still only fire on corrupt data.

`PlayerMistTickIntervalMs` (3000 ms) also moves to `mistcast`. Its doc comment
— the one explaining why it must exceed atlas-maps' `monsterDotTickIntervalMs`
— moves with it verbatim; it is the most expensively-learned comment in this
subsystem.

**`poisonmist` is refactored onto `mistcast` in this task.** Leaving it as the
fifth copy would mean the shared helper is a fork of the real implementation
rather than the implementation, and the PRD's regression bar ("Poison Mist
behaviour unchanged; its existing tests pass untouched") is exactly the check
that makes this refactor safe to do now.

### 7.2 The five handler packages

Each new subpackage is ~40 lines: an `init()` registering on the correct
registry, and an `Apply` that builds `Params` and delegates to
`mistcast.Cast`.

| package | registry | identity |
|---|---|---|
| `skill/handler/poisonmist` | `RegisterAttackCast` | `skill2.FirePoisonMagicianPoisonMist` |
| `skill/handler/flamegear` | `RegisterAttackCast` | `skill2.BlazeWizardStage3FlameGear` |
| `skill/handler/poisonbomb` | `RegisterAttackCast` | `skill2.NightWalkerStage3PoisonBomb` |
| `skill/handler/smokescreen` | `Register` | `skill2.ShadowerSmokescreen` |
| `skill/handler/recoveryaura` | `Register` | `skill2.EvanStage8RecoveryAura` |

All five register by `skill2.Identity` (FR-7.3), never a wire id, so one
registration covers every version that binds the skill. All four new packages
get a blank import in `skill/handler/registrations/registrations.go` (FR-7.5) —
a handler package that is not imported never runs its `init()` and is silently
absent.

`SourceSkillId` on the CREATE command is the **wire** `skillId` the handler was
called with, not the Identity (FR-7.4): the client compares it against its own
WZ to select the rendering arm (§2.1's `AffectedAreaAnimationCreated`
dispatch), so it must be the id that version binds.

The two USE_SKILL handlers take a `packetmodel.SkillUsageInfo` rather than
`(skillId, skillLevel)`; both read only the skill id and level from it, so the
`Params` construction is the only difference from the attack-cast three.

### 7.3 Per-skill constants, with the FR-6.4 window

| skill | targetKind | effectKind | disease | re-apply P | emitted DoT T | window P−T |
|---|---|---|---|---|---|---|
| Poison Mist | MONSTER | DAMAGE_OVER_TIME | POISON | 3000 | 1000 | 2000 ms |
| Flame Gear | MONSTER | DAMAGE_OVER_TIME | POISON | 3000 | 1000 | 2000 ms |
| Poison Bomb | MONSTER | DAMAGE_OVER_TIME | POISON | 3000 | 1000 | 2000 ms |
| Smokescreen | CHARACTER | PROTECTION | — | 0 (no tick) | — | — |
| Recovery Aura | CHARACTER | RECOVERY | — | 3000 | — | — |

`T` is atlas-maps' `monsterDotTickIntervalMs`, unchanged at 1000 ms.
`P > T` strictly, so the eligible damage window is 2000 ms per cycle rather
than zero — the failure mode documented at `poisonmist.go:33-63`, where
`P == T` makes the mist deal no damage at any tuning. The window is asserted in
a test (FR-6.4), not merely stated.

Shortest lifetimes clear the sub-tick gate: Flame Gear and Poison Bomb are
4000 ms at L1 on gms 72–92/jms (11 000 ms and 4000 ms at v95), all ≥ 3000 ms.
Recovery Aura is a flat 30 000 ms. Smokescreen's 31 000 ms is exempt from the
gate (no tick).

Poison Bomb sends `POISON` with `DiseaseValue: 0` (FR-6.2) — the magnitude is
target-derived and `atlas-monsters` overwrites anything sent. Flame Gear does
the same, for the reasons in §1.4 (FR-6.3).

---

## 8. FR-0 — the Evan snapshot re-drain

### 8.1 Root cause is already fixed upstream; only the snapshot is stale

The PRD's diagnosis is confirmed: `grep -cE "^\t22[0-9]{6}:"` returns **0** for
both `version_gms_95_1_gen.go` and `version_gms_84_1_gen.go`.

The reason, however, has expired. The 2026-07-30 drain used the jobs-union
method (`GET /api/data/jobs?page[size]=200`, union of each row's
`attributes.skills`) because the skills list endpoint returned 400 — and at
that time the Evan job documents were blank. Probed live at design time:

- `GET /api/data/skills` **still returns 400** on v95. The list-endpoint
  fallback is still required; no method change is possible there.
- `GET /api/data/jobs?page[size]=200` on v95 now returns populated Evan rows:
  `2200` n=2, `2210` n=2, `2211` n=2, `2212` n=3, `2213` n=2, `2214` n=4,
  `2215` n=4, `2216` n=4, `2217` n=5, `2218` n=4, plus the `220`/`221`/`222`
  branch rows.
- `GET /api/data/jobs/2216/skills` returns
  `[22160000, 22161001, 22161002, 22161003]` on GMS 84, 92, 95 and JMS 185, and
  **empty on GMS 48 and GMS 83** — so the endpoint is version-gated in this
  baseline and its per-version answers agree exactly with §1.1's per-skill
  sweep.

So FR-0 is **a re-run of the existing drain, not a new method**:

- **FR-0.1** Re-run the jobs-union drain against all 10 live tenants,
  overwrite `libs/atlas-constants/gen/wzsnapshot/<version>.json`, and update
  `PROVENANCE.md` with the new timestamp and a short note that the 2026-07-30
  drain predated the Evan job documents being populated (the `Skill.wz/Dragon/`
  subdirectory defect), which is why the `22xxxxxx` range was absent. The
  gms_12 mirror-from-gms_48 policy is unchanged.
- **FR-0.2** Regenerate; `22161003 → EvanStage8RecoveryAura` must appear on
  gms 84/87/92/95 and jms 185 and **nowhere else**, derived from the drain, not
  from this document.
- **FR-0.3** `go run . -check` in `libs/atlas-constants/gen` exits 0.
- **FR-0.4** The regeneration diff must be **additive only**. Any line where a
  previously-bound wire id changes its Identity is a defect to investigate, not
  a result to accept: task-187's divergence semantics and the generated ban
  list behind `tools/skill-job-id-guard.sh` both depend on those bindings. The
  plan phase should make this a mechanical check on the diff (no `-` lines in
  the binding maps other than pure reorderings), not an eyeball pass — the
  re-drain will add hundreds of lines and a single altered binding is easy to
  miss.
- **FR-0.5** does not fire: §1.1's live sweep already confirms the expected
  version set.

### 8.2 Scope discipline

The re-drain will pull in **every** skill the jobs-union now surfaces that it
did not on 2026-07-30, not only Evan's. That is unavoidable — the snapshot is
regenerated wholesale — and is why FR-0.4's additive-only check is the real
acceptance gate rather than "the diff is small". PRD non-goal "broadening the
wzsnapshot drain beyond what FR-0 requires" is respected in the sense that the
*method* and the *tenant set* are unchanged; only the data is fresh.

---

## 9. Open items for the plan phase

Genuinely unresolved, each with the check that settles it:

1. **Is Recovery Aura's `x` absolute MP or a percentage of max MP?** (§1.3)
   Not determinable from WZ. Design assumes absolute. Settle by casting at a
   known level and reading the MP delta against `x`. If it turns out to be a
   percentage, the change is confined to `tickRecovery`'s command amount and
   the `RecoveryMp` field's doc comment.
2. **Does `atlas-character`'s `ChangeMP` clamp to max MP?** (§6.3) FR-5.3
   depends on it. Read the processor; if it does not clamp, the clamp belongs
   in `atlas-character`, not in the mist tick.
3. **Registry confirmation for the two USE_SKILL handlers.** (§4) The WZ-shape
   argument is an argument from absence for Smokescreen and Recovery Aura. One
   live cast each, checking for the handler's `Infof` line, confirms it. A
   mis-registration fails loudly (handler never fires), not subtly.

Reported defects that this task does **not** fix, and does not work around:

4. **Live v95 `dotInterval`/`dotTime` are stale seconds.** (§1.5) Needs a
   re-ingest plus an atlas-data REST pod restart. Nothing in this design reads
   those fields.

Explicitly closed by this design: PRD open questions 1 (§5), 2 (§2.2),
3 (§5.2 — no new stat needed), 4 (§1.4), 5 (§1.3), 6 (§4), 7 (§1.1 / §8),
8 (§3.5).

---

## 10. Service impact, revised

| service | change | vs PRD §7 |
|---|---|---|
| `libs/atlas-constants` | re-drain 10 wzsnapshots, regenerate binding tables, update PROVENANCE | as specified |
| `atlas-data` | **none** — `x` is already exposed; no new WZ field is needed | reduced (PRD expected reader work) |
| `atlas-channel` | 4 new handler packages + `mistcast` shared package + `poisonmist` refactored onto it; registrations blank imports; mist contract mirror brought to full parity; **protection-mist registry in the mist consumer**; smoke short-circuit in `processDamageTaken`; party snapshot on the Recovery Aura cast | as specified, plus the registry (the mechanism choice in §5) |
| `atlas-maps` | 2 new effect kinds; `RecoveryMp` + `PartyMemberIds` on command/model/builder; `effectKind` on `CreatedBody`; `AffectedAreaTypeFor` extended with `AffectedAreaTypeSmoke`; `tickRecovery`; unknown-kind rejection | as specified |
| `atlas-buffs` | **none** — no temporary stat is used | reduced |
| `atlas-monsters` | **none** — Flame Gear applies existing `POISON` | as PRD's "none expected" |
| seed templates | **none** — `AffectedAreaCreated` / `AffectedAreaRemoved` already registered everywhere (`ae3341511`, #1226); to be re-verified, not assumed | as specified |
| `tools/` | new `mist-contract-mirror-guard.sh` + CLAUDE.md entry | new (answers open question 8) |

## 11. Testing

Unit, per FR:

- `AffectedAreaTypeFor` table test, all four outcomes (FR-3.4).
- `mistcast.Cast` rejection table: non-positive lifetime, sub-tick lifetime,
  degenerate rectangle, over-ceiling lifetime — each asserting *nothing* was
  emitted and nil was returned (FR-8.1–8.3); plus caster-load failure and
  emit failure (FR-8.4).
- Per-skill `Params` assertions: correct registry, correct target/effect kind,
  `SourceSkillId` is the wire id not the Identity (FR-7.4), `DiseaseValue` 0
  for both POISON mists (FR-6.2, FR-6.3).
- FR-6.4 window: assert `PlayerMistTickIntervalMs > monsterDotTickIntervalMs`
  and that the difference is the non-zero eligible window.
- `inProtectiveMist`: inside+party → true; inside+non-party → false (FR-4.6);
  outside+party → false (FR-4.3); expired mist → false (FR-4.3);
  destroyed mist → false (FR-4.3); monster-owned mist → false.
- `processDamageTaken` with a protection hit: zero HP change, **no** reflect,
  **no** meso spend, `computeMitigation` not reached (FR-4.1, FR-4.5).
- `tickRecovery`: heals party members inside; skips a non-party character
  inside (FR-5.2); skips characters outside; emits `CHANGE_MP` with `x` as the
  amount; emits nothing for an empty field.
- `tickOneMist` unknown-kind arm: warns naming the value, emits nothing
  (FR-2.5).
- Regression: existing Poison Mist tests pass **unmodified** after the
  `mistcast` refactor; the monster `AREA_POISON` path still derives `nType` 0;
  `applyStatusBody`'s key set is byte-identical.

Verification gates are CLAUDE.md's full list. Two are newly load-bearing here:
`tools/buff-duration-guard.sh` (the recovery path deliberately does **not**
touch `COMMAND_TOPIC_CHARACTER_BUFF`, so it should stay trivially clean) and
`tools/skill-job-id-guard.sh` (the FR-0 regeneration reshapes the tables its
ban list derives from).
