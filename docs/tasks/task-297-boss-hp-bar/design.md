# Boss HP Bar (FIELD_EFFECT / BOSS_HP) — Design

Task: task-297-boss-hp-bar
PRD: `docs/tasks/task-297-boss-hp-bar/prd.md` (approved)
Status: Draft
Created: 2026-09-04

---

## 1. Summary

All wire-level machinery exists. This design covers only atlas-channel: widen the
channel's `data/monster` projection to carry the WZ tag colors, add a small shared
package that owns the FR-1 qualification rule and the announcement, and hook it into
three call sites — damage, death, and field entry.

Two findings change the PRD's stated shape and are called out up front:

- **F1 (blocking correction).** A boss killed by damage does **not** emit `DESTROYED`.
  `services/atlas-monsters/atlas.com/monsters/monster/processor.go` `Damage` emits
  `DAMAGED` then `KILLED` and removes the monster from the registry
  (`processor_test.go:88-100`: `expected 2 events (damaged+killed)`; `:137-141`:
  "Monster must be gone"). FR-8 hooks `handleStatusEventDestroyed` only, which fires on
  despawn/removal, not on a kill. Implemented as literally written, the gauge would
  **never** empty when a boss dies — the acceptance criterion "empties on kill" would
  fail. This design hooks **both** `handleStatusEventDestroyed` and
  `handleStatusEventKilled`; both already evict the live mirror
  (`kafka/consumer/monster/consumer.go:212`, `:322`), and both event envelopes carry
  `Boss` and `MonsterId`.

- **F2.** The damage path's own re-fetch is `monster.NewProcessor(...).GetById(e.UniqueId)`
  against atlas-monsters (`consumer.go:268`). At kill time the monster is already gone from
  that registry, so the death gauge cannot be resolved through it. Death-path state comes
  from the event envelope (`e.MonsterId`) plus the live mirror (max HP), read before eviction.

---

## 2. Component map

| Change | File |
|---|---|
| D1 — projection widening | `services/atlas-channel/atlas.com/channel/data/monster/{rest.go,model.go}` |
| D1 — processor mock | `.../data/monster/mock/processor.go` (new) |
| D2 — template cache | `.../data/monster/cache.go` (new) |
| D3 — shared qualification + announce | `.../monster/bosshp/bosshp.go` (new package) |
| D4 — mirror max HP | `.../monster/live_mirror.go` |
| D5 — damage hook | `.../kafka/consumer/monster/consumer.go` `handleStatusEventDamaged` |
| D6 — death hooks | `.../kafka/consumer/monster/consumer.go` `handleStatusEventKilled`, `handleStatusEventDestroyed` |
| D7 — field-entry hook | `.../kafka/consumer/map/consumer.go` `spawnMonsterForSession` |

No change in `libs/atlas-packet`, `atlas-data`, `atlas-configurations`, or `atlas-ui`.

---

## 3. Design decisions

### D1 — Widen the channel `data/monster` projection (FR-15)

`RestModel` today declares only `Boss` and `FixedDamage`
(`data/monster/rest.go:8-12`); atlas-data already serves `tag_color` /
`tag_background_color` (`services/atlas-data/atlas.com/data/monster/rest.go:36-37`).

Add symmetrically:

```go
type RestModel struct {
    Id                 uint32 `json:"-"`
    Boss               bool   `json:"boss"`
    FixedDamage        uint32 `json:"fixed_damage"`
    TagColor           byte   `json:"tag_color"`
    TagBackgroundColor byte   `json:"tag_background_color"`
}
```

with `tagColor` / `tagBackgroundColor` on the immutable `Model`, accessors
`TagColor()` / `TagBackgroundColor()`, and both `Extract` and `Transform` updated.
The existing `TestTransformRoundTrip` (`rest_test.go:29`) is a `reflect.DeepEqual`
round trip and fails automatically if either direction is missed — that is the
symmetry guard, no new mechanism needed.

Also add `data/monster/mock/processor.go` mirroring the existing shape in
`data/{cash,npc,portal,skill,tradeability}/mock/processor.go`. It does not exist today
and the FR-17 failure tests need it.

*Rejected:* adding the tag colors to `StatusEventDamagedBody`. FR-16 forbids it, and
it would couple a Kafka contract to a rendering detail.

### D2 — Cache the template lookup (OQ2)

`data/monster.ProcessorImpl.GetById` is an uncached REST call (`processor.go:26`).
Two of the three call sites make it hot: a boss fight produces a high `DAMAGED` rate
for the same template, and field entry enumerates every monster in the map.

**Decision: port the existing in-process TTL cache to `data/monster`.**
`data/skill/cache.go` is the template — positive/negative TTL, `ErrNotFound`-only
negative caching, env kill-switch, lazy expiry, no singleflight, tenant-keyed. Monster
template data is immutable WZ data, so the cache is sound. Env names follow the
precedent: `MONSTER_DATA_CACHE_ENABLED` (default true), `MONSTER_DATA_CACHE_TTL`
(default 5m), `MONSTER_DATA_CACHE_NEGATIVE_TTL` (default 30s). The cache sits inside
`ProcessorImpl.GetById`; the `Processor` interface is unchanged, so every existing
caller benefits and no call site learns about caching.

Metrics (`data/skill/metrics.go`) are **not** ported — that is observability scope
belonging to a metrics task, and NFR-5 asks only for error logging.

*Rejected (a) — rely on the `e.Body.Boss` pre-filter alone.* It satisfies NFR-1's
letter (no lookup for ordinary mobs on the damage path) but leaves a Zakum fight
issuing one atlas-data round trip per damage event, and does nothing for the
field-entry path, which has no boss flag to pre-filter on.

*Rejected (c) — carry tag colors on the live mirror.* The mirror is seeded from the
atlas-monsters REST shape (`LiveEntryFromModel`, `live_mirror.go:73`), which has no
tag colors; sourcing them would mean an atlas-data lookup on every `CREATED` — i.e.
per mob spawn for *ordinary* mobs, strictly worse than the cache. The field-entry path
also reads the REST monster list, not the mirror, so it would not be served.

The `e.Body.Boss` pre-filter is still applied on the damage path (D5) — it is free and
keeps ordinary mobs out of the cache entirely.

### D3 — `monster/bosshp`, one home for the rule (FR-1, AC-2)

New package `services/atlas-channel/atlas.com/channel/monster/bosshp`. Neither
`session` nor `monster` imports the other today, so this creates no cycle.

```go
// Gauge is a resolved, qualifying boss HP gauge. Constructed only by Resolve,
// so a Gauge value is proof that FR-1 held.
type Gauge struct {
    monsterId          uint32 // template id, not unique id
    currentHp          uint32
    maxHp              uint32
    tagColor           byte
    tagBackgroundColor byte
}

type Resolver struct{ p monsterdata.Processor }

func NewResolver(l logrus.FieldLogger, ctx context.Context) *Resolver

// Resolve reports whether monsterTemplateId qualifies for the gauge and, if so,
// returns it. (Gauge{}, false, nil) means "does not qualify"; a non-nil error
// means the atlas-data lookup failed and the caller must log and skip (FR-17).
func (r *Resolver) Resolve(monsterTemplateId, currentHp, maxHp uint32) (Gauge, bool, error)

// AnnounceOperator sends one BOSS_HP field effect to a session.
func AnnounceOperator(l logrus.FieldLogger) func(context.Context) func(writer.Producer) func(Gauge) model.Operator[session.Model]
```

`Resolve` implements FR-1 as `m.Boss() && m.TagColor() != 0`. The **atlas-data** boss
flag is the authority; `e.Body.Boss` on the event is only a pre-filter (D5). FR-3 holds
by construction — `tagBackgroundColor` is copied, never tested.

`AnnounceOperator` wraps
`session.Announce(l)(ctx)(wp)(fieldcb.FieldEffectWriter)(fieldpkt.FieldEffectBossHpBody(...))`,
matching the two existing `FieldEffect` call sites
(`kafka/consumer/party_quest/consumer.go:93`, `kafka/consumer/event/consumer.go:86`).

NFR-2 needs no work: `FieldEffectBossHpBody` already goes through
`atlas_packet.WithResolvedCode("operations", "BOSS_HP", ...)`
(`libs/atlas-packet/field/field_effect_body.go:57`).

NFR-3 also needs no work: on a template with no `FieldEffect` writer,
`session.Announce` fails at `writerProducer(writerName)` and returns the error before
encoding (`session/processor.go:265-270`). Every call site logs and continues.

### D4 — Max HP on the live mirror (FR-9, F2)

`LiveEntry` carries `MonsterId`, `Mp`, `MaxMp` but no HP (`live_mirror.go:26-35`).
The death path needs `maxHp` for the gauge denominator and cannot get it from
atlas-monsters (F2).

Add `MaxHp uint32` to `LiveEntry`, seeded by `LiveEntryFromModel` (the source
`monster.Model` already exposes `MaxHp()`, `monster/model.go:120`) and refreshed
nowhere else — max HP does not change over a monster's life, so no per-damage mirror
write is added to the hot path.

Death-path resolution order, before the existing eviction calls:

1. `monster.GetLiveMirror().Lookup(t, e.UniqueId)` → `MaxHp`.
2. Template id from the event envelope `e.MonsterId` (populated by
   `services/atlas-monsters/.../producer.go:32` and `:158`), not from the mirror.
3. `Resolve(e.MonsterId, 0, maxHp)`.
4. Announce.
5. Then `GetNextSkillInbox().Evict` / `GetStatusMirror().OnMonsterGone` /
   `GetLiveMirror().Remove`, unchanged.

A mirror miss is FR-10: log at error, skip the gauge, leave the rest of the handler
untouched.

*Rejected:* using atlas-data's `hp` (base max HP) as the denominator. Spawned monsters
can carry an overridden max HP, and the mirror already knows the real one.

### D5 — Damage hook (FR-4 … FR-7)

Inside `handleStatusEventDamaged`, after the existing
`m, err := monsterGetByIdFn(...)` fetch and alongside — not inside — the existing
`e.Body.Boss` HP-bar branch:

```go
if e.Body.Boss {
    bossHpBroadcaster(l, ctx, sc, wp, f, e.MonsterId, m.Hp(), m.MaxHp())
}
```

- `e.Body.Boss` is the cheap pre-filter (NFR-1); `bossHpBroadcaster` still calls
  `Resolve`, so the atlas-data `boss` flag remains the authority for FR-1/FR-2.
- Template id is `e.MonsterId`; `m.Hp()`/`m.MaxHp()` are post-damage (FR-4).
- Placed **outside** the `shouldEchoDamagePacket` guard, next to the HP-bar broadcast
  the guard's own comment already exempts (FR-6).
- The broadcaster body runs under `routine.Go` and does not touch the existing
  `announcer` goroutine or the `MonsterDamage` goroutine (FR-5, FR-7).
- Change `consumer.go:268` from `monster.NewProcessor(l, ctx).GetById(e.UniqueId)` to
  the existing `monsterGetByIdFn` seam (`consumer.go:436`) so this handler is testable
  the way `handleStatusEventCreated` already is. Behaviour-identical.

### D6 — Death hooks (FR-8 … FR-10, corrected per F1)

Both `handleStatusEventKilled` and `handleStatusEventDestroyed` gain the D4 sequence
with `currentHp = 0`. `KILLED` is the path a real boss kill takes; `DESTROYED` covers
despawn and is kept so a boss removed without dying does not leave a stale gauge.

Both bodies carry `Boss bool`
(`kafka/message/monster/kafka.go` `StatusEventKilledBody`, and the `DESTROYED` body does
not — for `DESTROYED` the mirror lookup plus `Resolve` alone gates it, which is correct
and only costs a cached lookup per despawn).

Double-send risk: a kill emits `KILLED` only, a despawn emits `DESTROYED` only, so the
two hooks cannot both fire for one monster. If they ever did, the second is a no-op on
screen — an identical zero-HP gauge.

### D7 — Field-entry hook (FR-11 … FR-14)

Hook **inside** `spawnMonsterForSession` (`kafka/consumer/map/consumer.go:772`), at the
end of the operator, after the `MonsterSpawn` announce and after the conditional
`MonsterControl` grant.

This is the choice OQ4 asks for, and it buys FR-12 and FR-14 for free: the operator only
runs per monster already spawned, so Spawn-before-gauge is structural rather than a
convention a later edit can break; a field with no monsters never invokes the operator, so
no lookup happens; and FR-13's "one per monster, in enumeration order" is the operator's
natural semantics. The gauge is announced to `s` only.

Placing it after `MonsterControl` rather than between Spawn and Control keeps the
heavily-documented Spawn→Control invariant (`consumer.go:750-771`) physically
uninterrupted.

A `Resolve` error is logged and returns `nil` — a tag-color lookup failure must not abort
the spawn enumeration for the remaining monsters (FR-17).

*Rejected:* a separate pass after the enumeration completes. It would need its own monster
list fetch, would race the spawn goroutine (FR-12), and would double the REST traffic.

---

## 4. Test strategy

The repo's established seam for consumer-handler tests is a package-level function
variable swapped by the test (`monsterStatSetBroadcaster`, `consumer.go:499`;
`monsterGetByIdFn`, `consumer.go:436`). This design uses the same shape rather than
inventing a new injection mechanism.

- `kafka/consumer/monster`: `var bossHpBroadcaster = func(l, ctx, sc, wp, f, monsterTemplateId, currentHp, maxHp) {...}`
- `kafka/consumer/map`: `var bossHpSenderFn = func(l, ctx, wp, s, m monster.Model) error {...}`

| AC | Test | Location |
|---|---|---|
| Projection + symmetry | extend `TestExtract` / `TestTransformRoundTrip` with both tag colors; add an unmarshal test over a real atlas-data payload | `data/monster/rest_test.go` |
| Cache | port the `data/skill/cache_test.go` cases (hit, negative hit, expiry, kill-switch) | `data/monster/cache_test.go` |
| FR-1 / FR-2 / FR-3 gating | `Resolve` table test over (boss, tagColor) with `data/monster/mock` | `monster/bosshp/bosshp_test.go` |
| FR-17 lookup failure | `Resolve` returns error, callers log-and-continue | `bosshp_test.go` + each consumer test |
| FR-4 damage broadcast | swap `bossHpBroadcaster`, assert template id / post-damage HP / max HP / colors, and that it reached the map-wide seam | `kafka/consumer/monster/consumer_test.go` |
| FR-5 non-regression | same test asserts the existing `MonsterHealth` announcement still fires | same |
| FR-6 | `DamageSource = CHARACTER_ATTACK` (echo false) still broadcasts the gauge | same |
| FR-8 death | `KILLED` **and** `DESTROYED` each produce exactly one gauge with `currentHp == 0`, resolved before `Remove` (assert by seeding the mirror and checking the recorded `maxHp` is the mirror's) | same |
| FR-11 / FR-12 / FR-13 | swap `bossHpSenderFn`, assert order relative to `MonsterSpawn` and that it is addressed to the entering session only | `kafka/consumer/map/consumer_test.go` |
| FR-14 | empty field → zero sender invocations, zero lookups (mock counts calls) | same |
| Byte layout + resolved mode | encode `FieldEffectBossHpBody` with an `operations` map and assert the exact bytes, including that the mode byte comes from the table and is not `99` | `monster/bosshp/bosshp_test.go` |

`libs/atlas-packet` already carries the byte-fixture proof for `EffectBossHp` on
gms_v79/v83/v87/v95 and jms_v185 (`field/clientbound/effect_test.go:10-18`); the test above
proves the *body helper wiring*, not the struct, and does not duplicate that coverage.

Live smoke (OQ5): Zakum (`8800002`) on the gms_v83 tenant — first hit, damage tracking,
empty-on-kill, mid-fight entry — plus an ordinary mob (e.g. `100100`) confirming no
top-of-screen gauge and an unchanged over-head bar.

---

## 5. Resolved open questions

| OQ | Resolution |
|---|---|
| 1 — `gms_12` / `gms_92` lack the `FieldEffect` writer | **Flag and defer.** Confirmed: 9 of 11 seed templates carry `"writer": "FieldEffect"` with `"BOSS_HP"`; `template_gms_12_1.json` and `template_gms_92_1.json` carry neither. Adding `CField::OnFieldEffect` to those versions is a family-wide bring-up, not a boss-HP change, and would need its own IDB derivation and packet-audit verification. Write it up as a follow-up with this evidence (AC-15). NFR-3 already makes the feature degrade silently and safely on those tenants. |
| 2 — caching | D2: port the `data/skill` TTL cache into `data/monster`, keep the `e.Body.Boss` pre-filter on the damage path. |
| 3 — destroy-time resolution | D4/D6: template id from the event envelope `e.MonsterId`; max HP from a new `LiveEntry.MaxHp`, read before eviction. Not `monster.NewProcessor(...).GetById` — the monster is gone from the atlas-monsters registry by then (F2). |
| 4 — field-entry call site | D7: inside `spawnMonsterForSession`, after Spawn and after Control. |
| 5 — live smoke | Zakum `8800002` on gms_v83, plus an ordinary mob control. |

## 6. Deviations from the PRD

- **FR-8 widened to `KILLED`** (F1). Without this, no boss kill empties the gauge.
  The `DESTROYED` hook FR-8 names is kept as well.
- **`LiveEntry` gains `MaxHp`** — a file the PRD's §7 Service Impact does not list. It is
  the only source of true max HP at death time.
- **`data/monster` gains a cache and a mock** — also outside §7's list, required by NFR-1
  and by the FR-17 acceptance tests respectively.

## 7. Risks

- **R1.** Adding `MaxHp` to `LiveEntry` touches a struct read by the movement path.
  Mitigation: it is a pure addition with a default zero value; no existing reader is
  changed, and `live_mirror_test.go` covers the seed path.
- **R2.** The `data/monster` cache changes behaviour for existing callers (the damage
  path's `FixedDamage` read). Mitigation: template data is immutable WZ data, and the
  env kill-switch is the rollback.
- **R3.** Field entry into a dense map now does one cached lookup per monster.
  Mitigation: cache hit after the first monster of each template; the enumeration already
  runs in its own `routine.Go`.
