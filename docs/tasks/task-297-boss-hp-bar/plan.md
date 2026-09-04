# Boss HP Bar (FIELD_EFFECT / BOSS_HP) — Implementation Plan

Task: task-297-boss-hp-bar
Design: `docs/tasks/task-297-boss-hp-bar/design.md`
PRD: `docs/tasks/task-297-boss-hp-bar/prd.md`

All work is in one service: **atlas-channel**. Module root for every `go build` /
`go test` below is `services/atlas-channel/atlas.com/channel` (module `atlas-channel`).

`libs/atlas-packet` is NOT touched: `EffectBossHp`
(`libs/atlas-packet/field/clientbound/effect.go:126`) and `FieldEffectBossHpBody`
(`libs/atlas-packet/field/field_effect_body.go:57`) already exist and carry
`packet-audit:verify` markers for gms_v48/v61/v72/v79/v83/v84/v87/v95 and jms_v185.

Task order is sequential — Tasks 3–7 depend on the projection (Task 1) and the
`bosshp` package (Task 3).

---

## Task 1: Widen the channel `data/monster` projection and add its mock

Add `tag_color` / `tag_background_color` to atlas-channel's projection of atlas-data's
monster resource (FR-15), and add the package mock the later tests need.

atlas-data already serves both fields
(`services/atlas-data/atlas.com/data/monster/rest.go:37-38`, parsed from `hpTagColor` /
`hpTagBgcolor` at `reader.go:85-89`). No atlas-data change.

### Files

- `services/atlas-channel/atlas.com/channel/data/monster/rest.go` — add `TagColor byte \`json:"tag_color"\`` and `TagBackgroundColor byte \`json:"tag_background_color"\`` to `RestModel`; update `Extract` and `Transform` symmetrically
- `services/atlas-channel/atlas.com/channel/data/monster/model.go` — add unexported `tagColor` / `tagBackgroundColor` fields and `TagColor()` / `TagBackgroundColor()` accessors
- `services/atlas-channel/atlas.com/channel/data/monster/rest_test.go` — extend `TestExtract` and `TestTransformRoundTrip`; add a JSON-unmarshal case
- `services/atlas-channel/atlas.com/channel/data/monster/mock/processor.go` — **new file**; `ProcessorMock` for `monster.Processor`

Patterns to copy: `services/atlas-channel/atlas.com/channel/data/npc/mock/processor.go:1-24`
(mock shape: exported `*Func` fields, `var _ npc.Processor = (*ProcessorMock)(nil)`,
nil-guarded delegation).

Module root: `services/atlas-channel/atlas.com/channel`.

- [ ] **Step 1: Write the failing test**

Extend `data/monster/rest_test.go`. `TestExtract` and `TestTransformRoundTrip` already
exist (`rest_test.go:7`, `:29`) — widen them in place rather than adding new functions.
Add one new function `TestUnmarshalDataPayload`.

| test | change |
|---|---|
| `TestExtract` | build `RestModel{Boss: true, FixedDamage: 5, TagColor: 6, TagBackgroundColor: 1}`; add assertions `m.TagColor() == 6` and `m.TagBackgroundColor() == 1` |
| `TestTransformRoundTrip` | seed `Model{id: 11, boss: true, fixedDamage: 33, tagColor: 6, tagBackgroundColor: 1}`; the existing `reflect.DeepEqual` is the symmetry guard — no new assertion needed |
| `TestUnmarshalDataPayload` (new) | `json.Unmarshal` the attributes object below into a `RestModel`, then `Extract`, then assert all four fields |

The `TestUnmarshalDataPayload` fixture — an attributes object in atlas-data's real
shape, including fields the channel projection deliberately drops:

```json
{"boss":true,"fixed_damage":0,"tag_color":6,"tag_background_color":1,"hp":42000000,"level":80}
```

Expected after `Extract`: `Boss() == true`, `FixedDamage() == 0`, `TagColor() == 6`,
`TagBackgroundColor() == 1`. The dropped keys must not cause an unmarshal error.

- [ ] **Step 2: Implement**

Add the two fields to `RestModel` and to `Model`, wire both directions of
`Extract`/`Transform`, add the two accessors. Then write `data/monster/mock/processor.go`
in package `mock`, importing `atlas-channel/data/monster`:

```go
type ProcessorMock struct {
	GetByIdFunc func(monsterId uint32) (monster.Model, error)
}

var _ monster.Processor = (*ProcessorMock)(nil)
```

with `GetById` delegating when the func is non-nil and returning `(monster.Model{}, nil)`
otherwise.

- [ ] **Step 3: Verify**

`go build ./... && go test ./data/monster/...` from
`services/atlas-channel/atlas.com/channel`.

---

## Task 2: TTL cache for `data/monster.GetById`

`data/monster.ProcessorImpl.GetById` (`data/monster/processor.go:26`) is an uncached REST
call. Two of the three new call sites make it hot (per-damage-event on a boss fight;
per-monster on field entry). Port the `data/skill` in-process TTL cache (NFR-1, design D2).

Metrics are deliberately NOT ported (design D2) — `data/skill/cache.go` calls
`recordCache` from `data/skill/metrics.go`; drop those three calls and do not create a
`data/monster/metrics.go`. Drop the `outcomeCache*` constants with them.

Monster template data is immutable WZ data, so caching is sound; the env kill-switch is
the rollback (design R2).

### Files

- `services/atlas-channel/atlas.com/channel/data/monster/cache.go` — **new file**; port of `data/skill/cache.go`
- `services/atlas-channel/atlas.com/channel/data/monster/cache_test.go` — **new file**; port of `data/skill/cache_test.go`
- `services/atlas-channel/atlas.com/channel/data/monster/processor.go` — `GetById` delegates to `getByIdCached`; change `NewProcessor` to return `Processor` and add `var _ Processor = (*ProcessorImpl)(nil)`
- `services/atlas-channel/atlas.com/channel/main.go` — register `EvictTenant` in the existing `listener.RegisterEvictor` block

Patterns to copy: `services/atlas-channel/atlas.com/channel/data/skill/cache.go:1-201`
(whole file), `services/atlas-channel/atlas.com/channel/data/skill/processor.go:22-35`
(`NewProcessor` returning the interface plus the compile-time assertion),
`services/atlas-channel/atlas.com/channel/main.go:317-318` (evictor call sites).

Read-only reference: `services/atlas-channel/atlas.com/channel/socket/handler/character_damage.go:41`
— the one existing `data/monster` consumer; it goes through `GetById` and needs no change.

Module root: `services/atlas-channel/atlas.com/channel`.

Env names (design D2): `MONSTER_DATA_CACHE_ENABLED` (default `true`),
`MONSTER_DATA_CACHE_TTL` (default `5m`), `MONSTER_DATA_CACHE_NEGATIVE_TTL` (default `30s`).
Bounds as in `data/skill`: TTL `[1s, 24h]`, negative TTL `[0s, 5m]`.

- [ ] **Step 1: Write the failing test**

`TestMonsterDataCache` — table-driven, one `run func(t *testing.T)` per case, setup and
helper shape copied verbatim from `data/skill/cache_test.go:1-52` (`resetSkillCache`,
`newTestTenant`, `testSkillCtx`, `testSkillModel` → rename to `resetMonsterCache`,
`newTestTenant`, `testMonsterCtx`, `testMonsterModel`). `testMonsterModel` builds via
`Extract(RestModel{Id: id, Boss: true, TagColor: 6})`.

Each case swaps the `upstreamFn` package var for a counting stub and restores it with
`defer`, exactly as the skill cases do.

| case | setup | assertion |
|---|---|---|
| `PositiveHitAvoidsSecondFetch` | `upstreamFn` returns a model, counts calls; call `getByIdCached` twice for id `8800002` | fetch count `== 1`; both results equal the model |
| `ExpiredEntryRefetches` | set `MONSTER_DATA_CACHE_TTL=1s` via `t.Setenv` before reset; put an entry whose `expiresAt` is in the past via `c.put`; call `getByIdCached` | fetch count `== 1` (the expired entry was a miss) |
| `NegativeCachesNotFound` | `upstreamFn` returns `fmt.Errorf("x: %w", requests.ErrNotFound)`; call twice | fetch count `== 1`; both calls return an error satisfying `errors.Is(err, requests.ErrNotFound)` |
| `TransientErrorNotCached` | `upstreamFn` returns `errors.New("boom")`; call twice | fetch count `== 2` |
| `DisabledBypasses` | `t.Setenv("MONSTER_DATA_CACHE_ENABLED", "false")` before reset; call twice | fetch count `== 2` |
| `TenantIsolation` | two distinct tenants via `newTestTenant`; call once per tenant for the same id | fetch count `== 2`; each tenant's entry is its own |
| `ConcurrentAccess` | 50 goroutines calling `getByIdCached` for the same id under a `sync.WaitGroup` | no panic; every returned model equals the seeded one (run under `-race`) |

- [ ] **Step 2: Implement**

Copy `data/skill/cache.go` into `data/monster/cache.go`, package `monster`. Rename:
`skillCache` → `monsterCache`, `skillCacheOnce`/`skillCachePtr` → `monsterCacheOnce`/
`monsterCachePtr`, `getSkillCache` → `getMonsterCache`, parameter `skillId` → `monsterId`,
env consts to the three names above. `upstreamFn` becomes:

```go
var upstreamFn = func(l logrus.FieldLogger, ctx context.Context, monsterId uint32) (Model, error) {
	return requests.Provider[RestModel, Model](l, ctx)(requestById(ctx, monsterId), Extract)()
}
```

Keep `SeedForTest` and `EvictTenant` exported with the same semantics. Remove every
`recordCache(...)` line and the `outcomeCache*` constants.

Then in `processor.go`: `func (p *ProcessorImpl) GetById(monsterId uint32) (Model, error)
{ return getByIdCached(p.l, p.ctx, monsterId) }`, change `NewProcessor` to return
`Processor`, and add `var _ Processor = (*ProcessorImpl)(nil)`.

In `main.go`, add the import `datamonster "atlas-channel/data/monster"` (alias required —
`monsterDomain` and `monsterinfo` already occupy the obvious names) and add
`datamonster.EvictTenant(tid)` next to `dataskill.EvictTenant(tid)` at `main.go:318`.

- [ ] **Step 3: Verify**

`go build ./... && go test -race ./data/monster/... ./socket/handler/...` from the module
root. The `socket/handler` run guards the one pre-existing `data/monster` consumer against
the `NewProcessor` return-type change.

---

## Task 3: The `monster/bosshp` package — qualification rule and announcement

One home for FR-1 so the three call sites do not each re-derive it (AC-2), plus the
announce operator.

No import cycle: `atlas-channel/monster/bosshp` imports `atlas-channel/session`,
`atlas-channel/socket/writer` and `atlas-channel/data/monster`; none of those imports
`monster/bosshp`.

### Files

- `services/atlas-channel/atlas.com/channel/monster/bosshp/bosshp.go` — **new file**
- `services/atlas-channel/atlas.com/channel/monster/bosshp/bosshp_test.go` — **new file**
- `services/atlas-channel/atlas.com/channel/data/monster/mock/processor.go` — new file, already created in Task 1; read-only here

Patterns to copy:
`services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go:185-193`
(`spawnForSession` — the curried `func(l) func(ctx) func(wp) func(arg) model.Operator[session.Model]`
shape), and `services/atlas-channel/atlas.com/channel/kafka/consumer/event/consumer.go:86`
(the existing `FieldEffectWriter` announce).

Module root: `services/atlas-channel/atlas.com/channel`.

- [ ] **Step 1: Write the failing test**

`TestResolve` — table-driven over `(boss, tagColor, tagBackgroundColor, lookupErr)`, using
`mock.ProcessorMock` from Task 1 and `bosshp.NewResolverFrom(p)`.

| case | mock returns | expect `ok` | expect `err` | expect gauge |
|---|---|---|---|---|
| qualifying | `Boss=true, TagColor=6, TagBackgroundColor=1` | `true` | `nil` | `monsterId=8800002, currentHp=50000, maxHp=100000, tagColor=6, tagBackgroundColor=1` |
| boss, zero tag colour (FR-2) | `Boss=true, TagColor=0, TagBackgroundColor=1` | `false` | `nil` | zero value |
| non-boss, non-zero tag colour (FR-2) | `Boss=false, TagColor=6` | `false` | `nil` | zero value |
| zero background colour still qualifies (FR-3) | `Boss=true, TagColor=6, TagBackgroundColor=0` | `true` | `nil` | `tagBackgroundColor=0` |
| lookup failure (FR-17) | `(monsterdata.Model{}, errors.New("boom"))` | `false` | non-nil, `err.Error()` contains `"boom"` | zero value |

All cases call `Resolve(8800002, 50000, 100000)`. Assert gauge fields through the exported
accessors (`MonsterId()`, `CurrentHp()`, `MaxHp()`, `TagColor()`, `TagBackgroundColor()`).

`TestBossHpBodyBytes` — asserts the body helper's wire output with the mode byte resolved
from an `operations` table (NFR-2, AC-11), not hard-coded.

```go
options := map[string]interface{}{
	"operations": map[string]interface{}{"BOSS_HP": float64(5)},
}
```

Call `fieldpkt.FieldEffectBossHpBody(8800002, 50000, 100000, 6, 1)(logrus.New(), context.Background())(options)`.

Expected bytes, exactly 15, in order — mode, then `monsterId`/`currentHp`/`maxHp` as
little-endian uint32, then the two colour bytes (matches
`libs/atlas-packet/field/clientbound/effect.go:144-152`):

```
0x05 0x02 0x47 0x86 0x00 0x50 0xc3 0x00 0x00 0xa0 0x86 0x01 0x00 0x06 0x01
```

Second sub-case in the same function: with `options` set to
`map[string]interface{}{"operations": map[string]interface{}{}}` (BOSS_HP absent), the
first byte is the `99` sentinel from `atlas_packet.ResolveCode` — assert `b[0] == 99` to
prove the mode is table-driven and never a literal.

- [ ] **Step 2: Implement**

`bosshp.go`, package `bosshp`:

```go
// Gauge is a resolved, qualifying boss HP gauge. Constructed only by Resolve,
// so a Gauge value is proof that FR-1 held.
type Gauge struct {
	monsterId          uint32
	currentHp          uint32
	maxHp              uint32
	tagColor           byte
	tagBackgroundColor byte
}
```

with the five accessors named above.

```go
type Resolver struct{ p monsterdata.Processor }

func NewResolver(l logrus.FieldLogger, ctx context.Context) *Resolver
func NewResolverFrom(p monsterdata.Processor) *Resolver

// Resolve reports whether monsterTemplateId qualifies for the gauge. A false
// ok with a nil error means "does not qualify"; a non-nil error means the
// atlas-data lookup failed and the caller must log and skip (FR-17).
func (r *Resolver) Resolve(monsterTemplateId uint32, currentHp uint32, maxHp uint32) (Gauge, bool, error)
```

`NewResolver` builds `monsterdata.NewProcessor(l, ctx)`. `Resolve` calls
`r.p.GetById(monsterTemplateId)`, returns `(Gauge{}, false, err)` on error, and qualifies
on `m.Boss() && m.TagColor() != 0`. `tagBackgroundColor` is copied, never tested (FR-3).

```go
// AnnounceOperator sends one BOSS_HP field effect to a session.
func AnnounceOperator(l logrus.FieldLogger) func(context.Context) func(writer.Producer) func(Gauge) model.Operator[session.Model]
```

wrapping
`session.Announce(l)(ctx)(wp)(fieldcb.FieldEffectWriter)(fieldpkt.FieldEffectBossHpBody(g.monsterId, g.currentHp, g.maxHp, g.tagColor, g.tagBackgroundColor))`.

NFR-3 needs no code: on a template with no `FieldEffect` writer (`gms_12`, `gms_92`),
`session.Announce` fails at `writerProducer(writerName)`
(`session/processor.go:265-270`) and returns the error before encoding; every call site
below logs and continues.

- [ ] **Step 3: Verify**

`go build ./... && go test ./monster/bosshp/...` from the module root.

---

## Task 4: Carry max HP on the live mirror

The death path needs the gauge's denominator and cannot get it from atlas-monsters — at
`KILLED` the monster is already gone from that registry (design F2,
`services/atlas-monsters/atlas.com/monsters/monster/processor.go` `Damage`). The mirror is
the only source of true max HP at death time (design D4).

Pure addition: a new field with a zero default; no existing reader changes (design R1).

### Files

- `services/atlas-channel/atlas.com/channel/monster/live_mirror.go` — add `MaxHp uint32` to `LiveEntry` (`live_mirror.go:23-35`) and seed it in `LiveEntryFromModel` (`:73-83`)
- `services/atlas-channel/atlas.com/channel/monster/live_mirror_test.go` — assert the seed
- `services/atlas-channel/atlas.com/channel/monster/model.go` — read-only; `MaxHp()` at `model.go:120` is the source

Module root: `services/atlas-channel/atlas.com/channel`.

- [ ] **Step 1: Write the failing test**

`TestLiveEntryFromModel_SeedsMaxHp` — build a `monster.Model` with the package builder
(`monster/builder.go:30` `NewBuilder(uniqueId, field, monsterId)`, then `.SetHp(...)`,
`.SetMaxHp(...)`, `.SetMp(...)`, `.SetMaxMp(...)`, `.MustBuild()`), project it, assert.

| input | assertion |
|---|---|
| `NewBuilder(7001, field.NewBuilder(0,1,100000000).Build(), 8800002).SetHp(50000).SetMaxHp(100000).SetMp(60).SetMaxMp(90).MustBuild()` | `LiveEntryFromModel(m).MaxHp == 100000`, and the pre-existing fields are unchanged: `MonsterId == 8800002`, `Mp == 60`, `MaxMp == 90` |

Setup shape copied from the existing seed assertion in
`services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer_test.go:291-325`.

- [ ] **Step 2: Implement**

Add `MaxHp uint32` to `LiveEntry` next to `MaxMp`, and `MaxHp: mo.MaxHp(),` to
`LiveEntryFromModel`. Nothing else writes it — max HP does not change over a monster's
life, so no per-damage mirror write is added to the hot path.

- [ ] **Step 3: Verify**

`go build ./... && go test ./monster/...` from the module root.

---

## Task 5: Damage hook — broadcast the gauge on `DAMAGED`

FR-4 … FR-7. Add the broadcast seam and call it from `handleStatusEventDamaged`
alongside — not inside — the existing `e.Body.Boss` `MonsterHealth` branch.

### Files

- `services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go` — add the `bossHpBroadcaster` package var near the other seams (after `monsterStatResetBroadcaster`, `consumer.go:507`); call it from `handleStatusEventDamaged` (`consumer.go:257-305`); change `consumer.go:268` to the `monsterGetByIdFn` seam
- `services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer_test.go` — new cases

Patterns to copy:
`services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go:499-506`
(`monsterStatSetBroadcaster` — the broadcaster-seam shape) and
`consumer_test.go:31-46` (`withRecordingBroadcasters` — the swap-and-restore helper).

Module root: `services/atlas-channel/atlas.com/channel`.

- [ ] **Step 1: Write the failing test**

Add `withRecordingBossHp(t)` next to `withRecordingBroadcasters` (`consumer_test.go:31`),
returning a restore func and a `*[]bossHpRecord` where

```go
type bossHpRecord struct {
	monsterTemplateId uint32
	currentHp         uint32
	maxHp             uint32
}
```

`TestHandleStatusEventDamaged_BossHpGauge` — table-driven. Every case swaps
`monsterGetByIdFn` to return
`monster.NewBuilder(e.UniqueId, f, 8800002).SetHp(50000).SetMaxHp(100000).MustBuild()`
and swaps `bossHpBroadcaster` for the recorder. Tenant/server/field setup copied from
`consumer_test.go:291-300`. Event envelope: `WorldId: 0, ChannelId: 1, MapId: 100000000,
UniqueId: 7101, MonsterId: 8800002, Type: monster2.EventStatusDamaged`.

| case | `Body.Boss` | `Body.DamageSource` | expect records |
|---|---|---|---|
| boss damage broadcasts (FR-4) | `true` | `monster2.DamageSourceMonsterAttack` | exactly 1: `{8800002, 50000, 100000}` — note the **template** id, not `UniqueId` 7101 |
| echo-suppressed source still broadcasts (FR-6) | `true` | `monster2.DamageSourceCharacterAttack` | exactly 1: `{8800002, 50000, 100000}` |
| non-boss event does not broadcast (NFR-1 pre-filter) | `false` | `monster2.DamageSourceMonsterAttack` | 0 records |
| monster fetch failure aborts before the gauge | `true` | `monster2.DamageSourceMonsterAttack` | 0 records; `monsterGetByIdFn` returns `errors.New("boom")` |

`TestHandleStatusEventDamaged_GaugeDoesNotDisturbHealthPath` (FR-5) — the `MonsterHealth`
broadcast goes through `session.Announce` inside `_map.ForSessionsInMap` with no seam, and
the sessions this package's tests can register carry a nil connection, so a wire-level
assertion is not available here without inventing an injection point the design does not
call for. Assert the two things that are observable and that a regression would break:

| assertion | why it is the FR-5 guard |
|---|---|
| `monsterGetByIdFn` is invoked **exactly once** per `DAMAGED` event with `Body.Boss: true` | the gauge reuses the health path's single fetch; a second call would mean the gauge forked its own fetch and the shared `m` is no longer what the health branch sees |
| the gauge recorder fires **and** the handler returns without panicking, with `Body.Boss: true` and `DamageSource: monster2.DamageSourceMonsterAttack` | the health goroutine and the `MonsterDamage` goroutine are both still launched after the gauge call |

The pre-existing `TestShouldEchoDamagePacket` (`consumer_test.go:617`) is unchanged and
must still pass. Wire-level FR-5 coverage is the live smoke (AC-13), which is where the
"both bars coexist" claim is actually observable.

- [ ] **Step 2: Implement**

Add the seam. `routine.Go` lives **inside** the default body so the handler's call is
synchronous (and therefore assertable) while production stays non-blocking (FR-7):

```go
// bossHpBroadcaster resolves the FR-1 gauge for a monster template and fans the
// BOSS_HP field effect out to every session in the field. Package-level var so
// tests can record the broadcast without standing up ForSessionsInMap; the
// routine.Go is inside so the caller never blocks on the atlas-data lookup.
var bossHpBroadcaster = func(l logrus.FieldLogger, ctx context.Context, sc server.Model, wp writer.Producer, f field.Model, monsterTemplateId uint32, currentHp uint32, maxHp uint32) {
	routine.Go(l, ctx, func(_ context.Context) {
		g, ok, err := bosshp.NewResolver(l, ctx).Resolve(monsterTemplateId, currentHp, maxHp)
		if err != nil {
			l.WithError(err).Errorf("Unable to resolve boss HP gauge for monster template [%d] in field map [%d].", monsterTemplateId, f.MapId())
			return
		}
		if !ok {
			return
		}
		if err := _map.NewProcessor(l, ctx).ForSessionsInMap(f, bosshp.AnnounceOperator(l)(ctx)(wp)(g)); err != nil {
			l.WithError(err).Errorf("Unable to broadcast boss HP gauge for monster template [%d] in field map [%d].", monsterTemplateId, f.MapId())
		}
	})
}
```

`field.Model` has `MapId()` (`libs/atlas-constants/field/model.go:37`) and no `String()`.

In `handleStatusEventDamaged`: change `consumer.go:268` from
`monster.NewProcessor(l, ctx).GetById(e.UniqueId)` to `monsterGetByIdFn(l, ctx, e.UniqueId)`
(behaviour-identical; `monsterGetByIdFn` at `consumer.go:436` is exactly that call). Then
after `f := sc.Field(e.MapId, e.Instance)` and outside the `shouldEchoDamagePacket` guard:

```go
if e.Body.Boss {
	bossHpBroadcaster(l, ctx, sc, wp, f, e.MonsterId, m.Hp(), m.MaxHp())
}
```

Do not touch the existing `announcer` goroutine or the `MonsterDamage` goroutine.

- [ ] **Step 3: Verify**

`go build ./... && go test -race ./kafka/consumer/monster/...` from the module root.

---

## Task 6: Death hooks — empty the gauge on `KILLED` and `DESTROYED`

FR-8 … FR-10, corrected per design F1: a boss killed by damage emits `KILLED`, not
`DESTROYED` (`services/atlas-monsters/atlas.com/monsters/monster/producer.go`
`killedStatusEventProvider`). Hooking only `DESTROYED`, as the PRD literally states, would
mean the gauge never empties on a kill. Both handlers get the hook; `DESTROYED` covers
despawn so a boss removed without dying leaves no stale gauge.

Both envelopes carry `MonsterId` (`kafka/message/monster/kafka.go:214`; populated by
`statusEventProvider` for both event types).

### Files

- `services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go` — add the death-gauge sequence to `handleStatusEventDestroyed` (`consumer.go:195-219`) and `handleStatusEventKilled` (`consumer.go:307-324`)
- `services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer_test.go` — new cases
- `services/atlas-channel/atlas.com/channel/monster/live_mirror.go` — read-only; `Lookup` supplies `MaxHp` from Task 4

Module root: `services/atlas-channel/atlas.com/channel`.

- [ ] **Step 1: Write the failing test**

`TestHandleStatusEventDeath_BossHpGaugeEmpties` — table-driven, reusing
`withRecordingBossHp` from Task 5 and the mirror-seeding shape at
`consumer_test.go:413-439`.

| case | setup | expect records |
|---|---|---|
| `KILLED` empties the gauge (FR-8) | `GetLiveMirror().Put(tm, 7201, monster.LiveEntry{Field: f, MonsterId: 8800002, MaxHp: 100000})`; fire `handleStatusEventKilled` with `UniqueId: 7201, MonsterId: 8800002, Type: monster2.EventStatusKilled` | exactly 1: `{8800002, 0, 100000}` |
| `DESTROYED` empties the gauge (FR-8) | same seed at `UniqueId: 7202`; fire `handleStatusEventDestroyed` with `Type: monster2.EventStatusDestroyed` | exactly 1: `{8800002, 0, 100000}` |
| mirror miss logs and skips (FR-10) | no `Put`; fire `handleStatusEventKilled` for `UniqueId: 7203` | 0 records; the handler still completes (assert with the existing `Lookup(tm, 7203)` miss and no panic) |
| resolved before eviction (FR-9) | seed `UniqueId: 7204` with `MaxHp: 100000`; fire `handleStatusEventKilled` | the record's `maxHp == 100000` — a value only obtainable before `Remove`; and `GetLiveMirror().Lookup(tm, 7204)` is a miss afterwards |

The existing `TestHandleStatusEventDestroyedAndKilled_RemoveMirrorEntry`
(`consumer_test.go:413`) must still pass unchanged.

- [ ] **Step 2: Implement**

In both handlers, insert the sequence **before** the existing eviction calls
(`GetNextSkillInbox().Evict` / `GetStatusMirror().OnMonsterGone` / `GetLiveMirror().Remove`),
after the existing `ForSessionsInMap` destroy/kill announcement:

```go
t := tenant.MustFromContext(ctx)
if le, ok := monster.GetLiveMirror().Lookup(t, e.UniqueId); ok {
	bossHpBroadcaster(l, ctx, sc, wp, sc.Field(e.MapId, e.Instance), e.MonsterId, 0, le.MaxHp)
} else {
	l.Errorf("Unable to resolve monster [%d] from the live mirror; skipping the boss HP gauge clear.", e.UniqueId)
}
```

`handleStatusEventDestroyed` already binds `t := tenant.MustFromContext(ctx)`
(`consumer.go:216`) — move that binding above the new block rather than shadowing it.
`handleStatusEventKilled` calls `tenant.MustFromContext(ctx)` twice inline
(`consumer.go:321-322`); hoist it to a single `t` and reuse.

The FR-1 gate still runs inside `bossHpBroadcaster`, so a non-qualifying monster costs one
cached lookup and sends nothing. `DESTROYED`'s body carries no `Boss` flag, which is fine —
`Resolve` is the authority (design D6).

Double-send is not possible: a kill emits `KILLED` only, a despawn `DESTROYED` only.

- [ ] **Step 3: Verify**

`go build ./... && go test -race ./kafka/consumer/monster/...` from the module root.

---

## Task 7: Field-entry hook — show the current gauge to an entering character

FR-11 … FR-14. Hook inside `spawnMonsterForSession`, at the end of the operator, after the
`MonsterSpawn` announce and after the conditional `MonsterControl` grant (design D7).

That placement buys three requirements structurally: the operator only runs per already-
spawned monster, so Spawn-before-gauge cannot be broken by a later edit (FR-12); a field
with no monsters never invokes it, so no lookup happens (FR-14); and one gauge per monster
in enumeration order is the operator's natural semantics (FR-13). Placing it after
`MonsterControl` rather than between Spawn and Control keeps the heavily-documented
Spawn→Control invariant (`consumer.go:750-771`) physically uninterrupted.

### Files

- `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go` — add the `bossHpSenderFn` package var and call it at the end of `spawnMonsterForSession` (`consumer.go:772-790`)
- `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer_test.go` — new cases

Patterns to copy:
`services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer_test.go:359-398`
(`stubDoorAnnounceForVisuals` — the swap-a-package-var-and-record-writer-names helper) and
`consumer_test.go:35-57` (`newTestCtx`, `newTestField`).

Module root: `services/atlas-channel/atlas.com/channel`.

- [ ] **Step 1: Write the failing test**

Ordering (FR-12) is assertable only if both the spawn announce and the gauge write into
the same recorder. `spawnMonsterForSession` currently calls `session.Announce` inline;
Step 2 routes it through the package's existing `doorAnnounce` seam (`consumer.go:825`),
which is already the general session-announce seam in this file — it carries ContiMove,
Jukebox and FieldObstacle traffic, not just doors.

`withRecordingSpawnSeams(t)` — helper stubbing both seams into one shared `[]string`:
`doorAnnounce` appends its `writerName`; the `bossHpSenderFn` stub appends `"bosshp"` and
records `(characterId, monsterTemplateId, currentHp, maxHp)` into a `[]bossHpRecord`.
Restore both with `defer`. Swap-and-restore shape copied from
`consumer_test.go:359-398`.

`TestSpawnMonsterForSession_SendsBossHpAfterSpawn` — table-driven. Every case runs
`spawnMonsterForSession(logrus.New())(newTestCtx(t))(nil)(s)` over its monsters, where
`s` is `session.Model{}` and the monsters are built with
`monster.NewBuilder(uniqueId, newTestField(), templateId).SetHp(...).SetMaxHp(...).MustBuild()`.

| case | monsters | expect writer/seam sequence | expect gauge records |
|---|---|---|---|
| gauge follows Spawn (FR-11, FR-12) | one: `NewBuilder(7301, f, 8800002).SetHp(50000).SetMaxHp(100000)`, `ControlCharacterId` unset | `[MonsterSpawnWriter, "bosshp"]` | 1: `{characterId: 0, 8800002, 50000, 100000}` |
| gauge follows Spawn **and** Control (FR-12) | one: same, plus `.SetControlCharacterId(0)` so the control branch fires for `s.CharacterId() == 0` | `[MonsterSpawnWriter, MonsterControlWriter, "bosshp"]` | 1 record |
| one gauge per monster in enumeration order (FR-13) | three, template ids `8800002`, `8810018`, `100100`, each `SetHp(1).SetMaxHp(2)` | `[Spawn, "bosshp", Spawn, "bosshp", Spawn, "bosshp"]` | 3 records with template ids in exactly that order |
| no monsters → no invocation, no lookup (FR-14) | none — the operator is never called | `[]` | 0 records |
| sender failure does not abort the enumeration (FR-17) | two; stub returns `errors.New("boom")` for the first | both spawns present | 2 records; the operator returns `nil` for both |
| spawn failure suppresses the gauge | one; `doorAnnounce` returns `errors.New("boom")` for `MonsterSpawnWriter` | `[MonsterSpawnWriter]` | 0 records |

Writer names: `monsterpkt.MonsterSpawnWriter`, `monsterpkt.MonsterControlWriter`
(`libs/atlas-packet/monster/clientbound`), already imported by `consumer.go`.

- [ ] **Step 2: Implement**

```go
// bossHpSenderFn sends the current BOSS_HP gauge for one already-spawned
// monster to one entering session. Package-level var so the map-entry
// ordering is assertable without a live writer.
var bossHpSenderFn = func(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, s session.Model, m monster.Model) error {
	g, ok, err := bosshp.NewResolver(l, ctx).Resolve(m.MonsterId(), m.Hp(), m.MaxHp())
	if err != nil {
		l.WithError(err).Errorf("Unable to resolve boss HP gauge for mob [%d]; skipping.", m.UniqueId())
		return nil
	}
	if !ok {
		return nil
	}
	return bosshp.AnnounceOperator(l)(ctx)(wp)(g)(s)
}
```

Also route `spawnMonsterForSession`'s two existing announces through the `doorAnnounce`
seam (`consumer.go:825`) so the ordering is assertable — behaviour-identical, that var is
already `session.Announce(l)(ctx)(wp)(writerName)(enc)(s)`:

```go
if err := doorAnnounce(l, ctx, wp, monsterpkt.MonsterSpawnWriter, writer.SpawnMonsterBody(m, false), s); err != nil {
	return err
}
if m.ControlCharacterId() == s.CharacterId() {
	if err := doorAnnounce(l, ctx, wp, monsterpkt.MonsterControlWriter, writer.StartControlMonsterBody(m, m.ControllerHasAggro()), s); err != nil {
		l.WithError(err).Errorf("SpawnForSelf: unable to re-issue MonsterControl for character [%d] mob [%d].", s.CharacterId(), m.UniqueId())
	}
}
```

Leave the Spawn→Control doc comment (`consumer.go:750-771`) intact.

At the end of `spawnMonsterForSession`'s operator, replacing the bare `return nil`:

```go
if err := bossHpSenderFn(l, ctx, wp, s, m); err != nil {
	l.WithError(err).Errorf("Unable to send boss HP gauge for mob [%d] to character [%d].", m.UniqueId(), s.CharacterId())
}
return nil
```

The error is logged and swallowed so a tag-colour lookup or announce failure never aborts
the spawn enumeration for the remaining monsters (FR-17).

- [ ] **Step 3: Verify**

`go build ./... && go test -race ./kafka/consumer/map/...` from the module root.

---

## Task 8: Record the `gms_12` / `gms_92` missing-`FieldEffect`-writer follow-up

AC-15. Design OQ1 resolved this as flag-and-defer; the finding must be written down with
its evidence, not left in the design doc alone.

### Files

- `docs/tasks/task-297-boss-hp-bar/follow-up-field-effect-writer-gms12-gms92.md` — **new file**
- `services/atlas-configurations/seed-data/templates/template_gms_12_1.json` — read-only; evidence
- `services/atlas-configurations/seed-data/templates/template_gms_92_1.json` — read-only; evidence
- `docs/packets/dispatchers/field_effect.yaml` — read-only; evidence

- [ ] **Step 1: Confirm the evidence before writing it down**

Do not restate the design's claim — re-verify it:

```
grep -l '"writer": "FieldEffect"' services/atlas-configurations/seed-data/templates/*.json
grep -n 'FieldEffect' services/atlas-configurations/seed-data/templates/template_gms_12_1.json services/atlas-configurations/seed-data/templates/template_gms_92_1.json
cat docs/packets/dispatchers/field_effect.yaml
```

Expected (design OQ1): 9 of 11 templates carry the `FieldEffect` writer with a `BOSS_HP`
mode; `template_gms_12_1.json` and `template_gms_92_1.json` carry only
`FieldEffectWeather`; `field_effect.yaml` has no `gms_v92` mode column. If any of that does
not hold, write down what is actually true.

- [ ] **Step 2: Write the follow-up**

The document states: the gap (no `CField::OnFieldEffect` writer on those two templates),
the file:line evidence from Step 1, the blast radius (boss HP gauge, background music,
screen/sound effects, tremble, reward roulette — the whole family, not just `BOSS_HP`),
why it is out of scope here (adding the writer is a per-version bring-up needing its own
IDB derivation and `packet-audit` verification), and how the feature degrades on those
tenants today (NFR-3: `session.Announce` fails at `writerProducer(writerName)`,
`session/processor.go:265-270`, is logged, and nothing else is affected).

- [ ] **Step 3: Verify**

`tools/plan-lint.sh` and the doc guards run under `tools/verify.sh`; confirm no absolute
paths and repo-relative references only.

---

## Final verification

- [ ] Flagless `tools/verify.sh` exits 0 (AC-14). Dispatch `task-verifier` for this —
      never run it inside a large implementer context.
- [ ] Code review before the PR (`task-reviewer` per unit; `backend-guidelines-reviewer`
      over the changed Go packages).
- [ ] Live smoke (AC-12, AC-13), design §4: Zakum (`8800002`) on the gms_v83 tenant —
      the gauge appears on first hit, tracks damage, empties on kill, and is visible to a
      player entering mid-fight; plus an ordinary mob (`100100`) confirming no
      top-of-screen gauge and an unchanged over-head bar.
