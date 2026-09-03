# One-Time Spawn for `mobTime == -1` Spawn Points — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `atlas-maps` place every `mobTime < 0` spawn point exactly once per field arming, re-arm the field when it empties, and honor the `hide` flag — without changing recurring-spawn behavior on any other map.

**Architecture:** The per-field Redis state splits from one hash into three, all under the existing `maps:spawn` namespace and all keyed by `character.MapKey`: a **recurring** hash (byte-identical in shape to today), a **one-time** hash, and a **meta** hash holding `seeded` and `onetimeFired`. Every `keyFn` gains a leading `v2:` token so each field re-seeds exactly once after deploy. Firing is an atomic `HSETNX`-claim Lua script that returns the batch in the same round trip. Re-arming is a single `HDEL` in `map.ProcessorImpl.Exit`, gated on the field having actually fired, which also emits an existing-contract `DESTROY_FIELD` command to `atlas-monsters`.

**Tech Stack:** Go 1.27, `github.com/redis/go-redis/v9` (Lua via `goredis.NewScript`), `github.com/alicebob/miniredis/v2` v2.38.0 for tests (HSETNX/HDEL/HEXISTS/HLEN all supported in its Lua bridge — verified in `cmd_hash.go`), `segmentio/kafka-go` producers via `atlas-maps/kafka/message`.

**Spec:** `docs/tasks/task-294-mobtime-one-time-spawn/design.md` (PRD: `docs/tasks/task-294-mobtime-one-time-spawn/prd.md`)

## Global Constraints

- **Module root for every Go command in this plan:** `services/atlas-maps/atlas.com/maps` (module `atlas-maps`). All `go build ./...` / `go test ./...` invocations run with that as cwd. No other module is modified.
- **No change to `atlas-monsters`, `atlas-channel`, or `atlas-data`.** PRD §2 non-goal. `atlas-monsters` files are read-only reference in this plan.
- **The recurring path stays structurally untouched.** `Count`, `ReserveEligibleSpawnPoints`, `ResetCooldown`, `reserveEligibleScript`, and `resetCooldownScript` operate on the recurring hash and their logic does not change. This is the primary regression risk (PRD §8) and the `104040000` acceptance criterion.
- **One-time predicate is `MobTime < 0`, never `== -1`** (FR-1.2).
- **Hidden points (`Hide == true`) never enter either hash** (FR-1.4). `storedSpawnPoint` gains no `Hide` field — its Redis JSON shape is unchanged (design D4).
- **All three hashes share namespace `"maps:spawn"`**, so `ClearForTenantId`'s `<prefix>:maps:spawn:<tenantId>:*` SCAN sweeps all of them. Do not introduce a second namespace (design D1/D10).
- Redis key `keyFn` shapes, exact:
  - recurring — `v2:<world>:<channel>:<map>:<instance>`
  - one-time — `v2:onetime:<world>:<channel>:<map>:<instance>`
  - meta — `v2:meta:<world>:<channel>:<map>:<instance>`
- Meta hash field names, exact: `seeded`, `onetimeFired`.
- Preserve existing line endings. Repo-relative paths only in committed files.

---

## Task Dependency Order

Tasks 1 → 7 are strictly sequential; each leaves the module compiling and green.

| Task | Deliverable |
|---|---|
| 1 | `SpawnPoint.Hide` + `Classify` in the data package (old methods still present) |
| 2 | Registry: three hashes, `v2:` keys, three-key seed script, `Count`/`CountOneTime` |
| 3 | Registry: `ClaimOneTimeSpawnPoints`, `RearmOneTime` |
| 4 | `map/monster/processor.go`: fire the batch, recurring-only denominator, FR-4 logs |
| 5 | `DESTROY_FIELD` envelope + producer in `atlas-maps` |
| 6 | `map/processor.go` `Exit`: re-arm on empty + gated `DESTROY_FIELD` |
| 7 | Delete the dead `Spawnable*` surface; shrink the mock; refresh `domain.md` |

---

## Task 1: Carry `Hide` and classify spawn points in the data package

### Files

- `services/atlas-maps/atlas.com/maps/data/map/monster/model.go` — add `Hide bool` to `SpawnPoint`
- `services/atlas-maps/atlas.com/maps/data/map/monster/rest.go` — `Extract` carries `rm.Hide`
- `services/atlas-maps/atlas.com/maps/data/map/monster/classify.go` — **new file**; `Classified` + `Classify`
- `services/atlas-maps/atlas.com/maps/data/map/monster/classify_test.go` — **new file**; table tests
- `services/atlas-maps/atlas.com/maps/data/map/monster/processor.go` — read-only this task; `Spawnable`/`SpawnableSpawnPointProvider`/`GetSpawnableSpawnPoints` stay until Task 7 so the module keeps compiling

Patterns to copy: `services/atlas-maps/atlas.com/maps/data/map/monster/processor_drain_test.go` (same package, plain `testing`, no fixtures needed here).

Module root: `services/atlas-maps/atlas.com/maps`.

### Interfaces

- Produces: `monster.SpawnPoint.Hide bool`; `monster.Classified{Recurring, OneTime, Hidden []SpawnPoint}`; `func Classify(points []SpawnPoint) Classified`. Task 2 consumes both.

- [x] **Step 1: Write the failing tests**

New file `data/map/monster/classify_test.go`, package `monster`. Two test functions.

`TestExtractCarriesHide` — table-driven over `RestModel.Hide`:

| case | `RestModel.Hide` | expect `SpawnPoint.Hide` |
|---|---|---|
| hidden | `true` | `true` |
| visible | `false` | `false` |

Also assert on the hidden case that `Extract` still copies `Id: 7, Template: 9400545, MobTime: 0, Team: 1, CY: -10, F: 1, FH: 3, RX0: -50, RX1: 50, X: 100, Y: 200` into the matching `SpawnPoint` fields (`Cy`, `Fh`, `Rx0`, `Rx1`), so the `Hide` addition cannot silently drop a sibling field.

`TestClassify` — one call with a single input slice, three assertions on the result.

Input slice, in this order:

| id | mobTime | hide | expected bucket |
|---|---|---|---|
| 1 | `30` | false | `Recurring` |
| 2 | `0` | false | `Recurring` |
| 3 | `-1` | false | `OneTime` |
| 4 | `-2` | false | `OneTime` |
| 5 | `0` | true | `Hidden` |
| 6 | `-1` | true | `Hidden` |

Assert: `Recurring` ids are exactly `[1, 2]` in that order; `OneTime` ids are exactly `[3, 4]` in that order; `Hidden` ids are exactly `[5, 6]` in that order. Order matters — `Classify` must preserve input order within each bucket so the seed argv is deterministic.

Add a third subtest `empty input` calling `Classify(nil)` and asserting all three slices have length 0.

- [x] **Step 2: Run the tests and verify they fail**

Run from `services/atlas-maps/atlas.com/maps`:

```bash
go test ./data/map/monster/ -run 'TestClassify|TestExtractCarriesHide' -v
```

Expected: FAIL — `undefined: Classify`, `undefined: Classified`, and `unknown field Hide in struct literal of type SpawnPoint`.

- [x] **Step 3: Add `Hide` to the model**

In `data/map/monster/model.go`, append to `SpawnPoint`:

```go
	Y        int16  // Y coordinate for spawn position
	Hide     bool   // WZ life `hide` flag; a hidden point is never auto-spawned (FR-1.4)
```

- [x] **Step 4: Carry `Hide` through `Extract`**

In `data/map/monster/rest.go`, add the field to the returned literal:

```go
		Y:        rm.Y,
		Hide:     rm.Hide,
	}, nil
```

- [x] **Step 5: Write `Classify`**

New file `data/map/monster/classify.go`:

```go
package monster

// Classified is the three-way partition of a map's monster life entries.
//
// The partition is computed once, in memory, from a single GetSpawnPoints
// drain. Two filtered providers would mean two paginated HTTP fetches of the
// same atlas-data endpoint per field initialization.
type Classified struct {
	Recurring []SpawnPoint // MobTime >= 0, not hidden
	OneTime   []SpawnPoint // MobTime <  0, not hidden
	Hidden    []SpawnPoint // Hide == true, either MobTime
}

// Classify partitions points into recurring, one-time and hidden buckets,
// preserving input order within each bucket.
//
// A hidden point is excluded from spawning entirely (FR-1.4), so the Hide
// test comes first. The one-time predicate is MobTime < 0 rather than
// MobTime == -1: -1 is the only negative value in the GMS 83.1 dataset, but
// an unexpected -2 must behave as one-time rather than falling through to
// the recurring path (FR-1.2).
func Classify(points []SpawnPoint) Classified {
	var c Classified
	for _, p := range points {
		switch {
		case p.Hide:
			c.Hidden = append(c.Hidden, p)
		case p.MobTime < 0:
			c.OneTime = append(c.OneTime, p)
		default:
			c.Recurring = append(c.Recurring, p)
		}
	}
	return c
}
```

- [x] **Step 6: Run the tests and verify they pass**

```bash
go test ./data/map/monster/ -v
```

Expected: PASS, including the pre-existing `processor_drain_test.go` cases.

- [x] **Step 7: Build the module**

```bash
go build ./...
```

Expected: exit 0. `Spawnable` and friends still exist, so nothing else breaks.

- [x] **Step 8: Commit**

```bash
git add data/map/monster/model.go data/map/monster/rest.go data/map/monster/classify.go data/map/monster/classify_test.go
git commit -m "feat(maps): carry Hide into SpawnPoint and classify spawn points"
```

---

## Task 2: Split the registry into recurring / one-time / meta hashes

### Files

- `services/atlas-maps/atlas.com/maps/map/monster/registry.go` — three `TenantKeyedHash` fields, `v2:` keyFns, per-receiver key helpers, three-key seed script, `Count` on recurring, new `CountOneTime`
- `services/atlas-maps/atlas.com/maps/map/monster/registry_test.go` — update `newTestRegistry`; add seeding/partition tests
- `services/atlas-maps/atlas.com/maps/map/monster/processor_test.go` — read-only this task, but `go test ./map/monster/` must stay green; `mockDataProcessor.GetSpawnPoints` already returns the unfiltered slice
- `libs/atlas-redis/keyed_hash.go` — read-only reference; `Key`, `Len`, `Del`, `SetNX`, `ClearForTenantId` are already exported and need no change

Patterns to copy: `services/atlas-maps/atlas.com/maps/map/monster/registry_test.go:25-44` (`setupSpawnTestRedis`, `newTestRegistry`); `services/atlas-maps/atlas.com/maps/map/monster/processor_test.go:29-39` (`TestMain` wiring `InitRegistry` against miniredis).

Module root: `services/atlas-maps/atlas.com/maps`.

### Interfaces

- Consumes from Task 1: `monster2.Classify`, `monster2.Classified`, `SpawnPoint.Hide`.
- Produces:
  - `(*SpawnPointRegistry).Count(ctx, mapKey) (int, error)` — now `HLEN` of the **recurring** hash only (signature unchanged)
  - `(*SpawnPointRegistry).CountOneTime(ctx, mapKey) (int, error)` — new
  - unexported: `(r *SpawnPointRegistry).recurringKey/oneTimeKey/metaKey(mapKey) string`
  - unexported consts `metaFieldSeeded = "seeded"`, `metaFieldOneTimeFired = "onetimeFired"`

- [x] **Step 1: Write the failing tests**

Append to `map/monster/registry_test.go`. First update the shared helper (this is required for every later test in the file, and `newTestRegistry` currently builds only one hash — a nil `oneTime`/`meta` would panic):

```go
func newTestRegistry(client *goredis.Client) *SpawnPointRegistry {
	return newRegistry(client)
}
```

…where `newRegistry` is the constructor extracted in Step 3. Keep the `newTestRegistry` name — `registry_test.go:46,77,121,136` all call it.

New tests, setup shape copied from `registry_test.go:46-72` (miniredis client + `tenant.Create(uuid.New(), "GMS", 83, 1)` + `field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(920010920)).Build()`):

`TestInitializeForMap_PartitionsByMobTimeAndHide` — table of cases, each seeding a fresh field key through `r.InitializeForMap(ctx, mapKey, mockDP, logrus.New())` where `mockDP` is a local minimal `monster2.Processor` stub whose `GetSpawnPoints` returns the case's slice.

| case | input points (id / mobTime / hide) | recurring HLEN | one-time HLEN | meta `seeded` |
|---|---|---|---|---|
| all one-time (`920010920` shape) | ids 1..10, all mobTime `-1`, hide false | 0 | 10 | `"1"` |
| recurring only (`104040000` shape) | ids 1..4, mobTime `0`, hide false | 4 | 0 | `"1"` |
| mixed | id 1 mobTime `0`; id 2 mobTime `-1`; id 3 mobTime `30` | 2 | 1 | `"1"` |
| hidden excluded (`600020300` shape) | id 1 mobTime `0` hide **true** | 0 | 0 | `"1"` |
| empty map | `nil` | 0 | 0 | `"1"` |

Read the counts with `r.Count(ctx, mapKey)` and `r.CountOneTime(ctx, mapKey)`; read the marker with `client.HGet(ctx, r.metaKey(mapKey), metaFieldSeeded).Result()`.

`TestInitializeForMap_IsIdempotent` — seed the mixed case, then mutate the stub's slice to a different 5-point recurring set and call `InitializeForMap` again. Assert recurring HLEN is still 2 and one-time HLEN is still 1.

`TestRegistryKeys_AreV2AndDistinct` — for one `mapKey` on world 1 / channel 2 / map `920010920` / instance `uuid.Nil`, assert:

- `r.recurringKey(mapKey)` ends with `:v2:1:2:920010920:00000000-0000-0000-0000-000000000000`
- `r.oneTimeKey(mapKey)` ends with `:v2:onetime:1:2:920010920:00000000-0000-0000-0000-000000000000`
- `r.metaKey(mapKey)` ends with `:v2:meta:1:2:920010920:00000000-0000-0000-0000-000000000000`
- all three contain the substring `:maps:spawn:` and the tenant UUID string
- all three are pairwise distinct

`TestFlushTenant_ClearsAllThreeHashes` — seed the mixed case for one tenant, assert all three keys exist via `client.Exists`, call `r.FlushTenant(ctx, l, tid)`, assert `deleted == 3` and all three keys are gone. (This is the FR-3.5 key-pattern pin; the arming-behavior half of FR-3.5 lands in Task 3.)

- [x] **Step 2: Run the tests and verify they fail**

```bash
go test ./map/monster/ -run 'TestInitializeForMap_Partitions|TestInitializeForMap_IsIdempotent|TestRegistryKeys|TestFlushTenant_ClearsAllThree' -v
```

Expected: FAIL — `undefined: CountOneTime`, `undefined: metaKey`, `undefined: metaFieldSeeded`, `undefined: newRegistry`.

- [x] **Step 3: Extract a constructor and add the three hashes**

Replace the `SpawnPointRegistry` struct, `InitRegistry`, and `spawnHashKey` in `map/monster/registry.go`:

```go
// SpawnPointRegistry holds three tenant-scoped hashes per field, all under the
// shared "maps:spawn" namespace so ClearForTenantId's namespace-wide SCAN
// sweeps every one of them (design D1/D10):
//
//	recurring — storedSpawnPoint per MobTime >= 0, non-hidden point. Shape and
//	            behavior identical to the single hash this replaces.
//	oneTime   — storedSpawnPoint per MobTime < 0, non-hidden point. Static;
//	            NextSpawnAt is written but never read.
//	meta      — "seeded" = "1"; "onetimeFired" = fire timestamp, present iff
//	            the field is disarmed.
type SpawnPointRegistry struct {
	client  *goredis.Client
	hashes  *atlasredis.TenantKeyedHash[character.MapKey]
	oneTime *atlasredis.TenantKeyedHash[character.MapKey]
	meta    *atlasredis.TenantKeyedHash[character.MapKey]
}

// Meta-hash field names.
const (
	metaFieldSeeded       = "seeded"
	metaFieldOneTimeFired = "onetimeFired"
)

// fieldSuffix renders the field-scoped portion of a spawn key. Tenant scoping
// is applied by TenantKeyedHash itself.
func fieldSuffix(mk character.MapKey) string {
	return fmt.Sprintf("%d:%d:%d:%s",
		mk.Field.WorldId(),
		mk.Field.ChannelId(),
		mk.Field.MapId(),
		mk.Field.Instance().String(),
	)
}

// newRegistry builds a fully-wired registry. The "v2:" token in every keyFn is
// a key-schema break, not a value-schema break: storedSpawnPoint's JSON is
// unchanged, but a field seeded by the pre-task-294 code holds only the
// recurring subset and InitializeForMap's "already seeded" guard would never
// re-seed it. Changing the key shape makes every field re-seed exactly once
// after deploy. The orphaned v1 keys are inert (no reader), bounded (one per
// field per tenant ever visited), and are reaped by the next DATA_UPDATED
// flush, whose SCAN pattern is namespace-wide (design D2).
func newRegistry(rc *goredis.Client) *SpawnPointRegistry {
	return &SpawnPointRegistry{
		client: rc,
		hashes: atlasredis.NewTenantKeyedHash[character.MapKey](rc, "maps:spawn", func(mk character.MapKey) string {
			return "v2:" + fieldSuffix(mk)
		}),
		oneTime: atlasredis.NewTenantKeyedHash[character.MapKey](rc, "maps:spawn", func(mk character.MapKey) string {
			return "v2:onetime:" + fieldSuffix(mk)
		}),
		meta: atlasredis.NewTenantKeyedHash[character.MapKey](rc, "maps:spawn", func(mk character.MapKey) string {
			return "v2:meta:" + fieldSuffix(mk)
		}),
	}
}

// InitRegistry initializes the singleton SpawnPointRegistry with a Redis client.
func InitRegistry(rc *goredis.Client) {
	registryOnce.Do(func() {
		registryInstance = newRegistry(rc)
	})
}

func (r *SpawnPointRegistry) recurringKey(mapKey character.MapKey) string {
	return r.hashes.Key(mapKey.Tenant, mapKey)
}

func (r *SpawnPointRegistry) oneTimeKey(mapKey character.MapKey) string {
	return r.oneTime.Key(mapKey.Tenant, mapKey)
}

func (r *SpawnPointRegistry) metaKey(mapKey character.MapKey) string {
	return r.meta.Key(mapKey.Tenant, mapKey)
}
```

Delete the package-level `spawnHashKey` function entirely. It read `registryInstance.hashes` rather than the receiver's, which silently bypassed any non-singleton registry. Replace its three call sites (`registry.go:195`, `:256`, `:289`) with `r.recurringKey(mapKey)`.

- [x] **Step 4: Replace the seed script**

Replace `initializeScript` with a three-key version. `redis.call('HSET', ...)` per pair rather than one variadic HSET keeps the script identical in shape to what it replaces and avoids a Lua `unpack` length limit on the 50-point maps.

```go
// initializeScript atomically seeds the recurring and one-time hashes for a
// field and stamps the meta "seeded" marker, if and only if the field has not
// been seeded before. Atomic across all three keys: the client is a single-node
// goredis.Client, not a cluster client, so multi-key Lua is safe.
//
// A field with zero points of any kind still gets "seeded", so it costs one
// HEXISTS per pass thereafter instead of a fresh paginated HTTP drain.
//
// KEYS[1] = recurring hash, KEYS[2] = one-time hash, KEYS[3] = meta hash
// ARGV[1] = number of recurring points; ARGV[2..] = field/value pairs,
//           recurring first, then one-time.
// Returns: 1 if seeded by this call, 0 if already seeded.
var initializeScript = goredis.NewScript(`
if redis.call('HEXISTS', KEYS[3], 'seeded') == 1 then
    return 0
end
local nRec = tonumber(ARGV[1])
local i = 2
for k = 1, nRec do
    redis.call('HSET', KEYS[1], ARGV[i], ARGV[i+1])
    i = i + 2
end
while i <= #ARGV do
    redis.call('HSET', KEYS[2], ARGV[i], ARGV[i+1])
    i = i + 2
end
redis.call('HSET', KEYS[3], 'seeded', '1')
return 1
`)
```

- [x] **Step 5: Rewrite `InitializeForMap`**

Replace the body (`registry.go:194-234`):

```go
// InitializeForMap seeds a field's spawn points if it has not been seeded yet.
// The classification happens once, in memory, off a single paginated drain of
// atlas-data's /maps/{id}/monsters — two filtered providers would mean two
// fetches per field initialization (design D3).
func (r *SpawnPointRegistry) InitializeForMap(ctx context.Context, mapKey character.MapKey, dp monster2.Processor, l logrus.FieldLogger) error {
	seeded, err := r.meta.Exists(ctx, mapKey.Tenant, mapKey, metaFieldSeeded)
	if err != nil {
		return err
	}
	if seeded {
		return nil
	}

	spawnPoints, err := dp.GetSpawnPoints(mapKey.Field.MapId())
	if err != nil {
		return err
	}
	classified := monster2.Classify(spawnPoints)

	now := time.Now()
	args := make([]interface{}, 0, 1+(len(classified.Recurring)+len(classified.OneTime))*2)
	args = append(args, strconv.Itoa(len(classified.Recurring)))
	for _, sp := range append(append([]monster2.SpawnPoint{}, classified.Recurring...), classified.OneTime...) {
		data, merr := json.Marshal(toStored(sp, now))
		if merr != nil {
			return merr
		}
		args = append(args, strconv.FormatUint(uint64(sp.Id), 10), string(data))
	}

	_, err = initializeScript.Run(ctx, r.client,
		[]string{r.recurringKey(mapKey), r.oneTimeKey(mapKey), r.metaKey(mapKey)},
		args...).Result()
	if err != nil {
		return err
	}

	l.Debugf("Initialized spawn point registry for map key: Tenant [%s] World [%d] Channel [%d] Map [%d] with %d recurring, %d one-time, %d hidden spawn points",
		mapKey.Tenant.String(), mapKey.Field.WorldId(), mapKey.Field.ChannelId(), mapKey.Field.MapId(),
		len(classified.Recurring), len(classified.OneTime), len(classified.Hidden))

	return nil
}
```

Note the removed `if len(spawnPoints) == 0 { return nil }` early return: a zero-point field must still be stamped `seeded`.

- [x] **Step 6: Point `Count` at the recurring hash and add `CountOneTime`**

`Count`'s body is already `r.hashes.Len(...)`, which is now the recurring hash — leave the code, update the doc comment, and add the sibling:

```go
// Count returns the number of RECURRING spawn points registered for a field.
// One-time points are deliberately excluded: this count is the denominator of
// getMonsterMax, and a one-time batch must not inflate the recurring
// population target (FR-2.5).
func (r *SpawnPointRegistry) Count(ctx context.Context, mapKey character.MapKey) (int, error) {

// CountOneTime returns the number of one-time spawn points registered for a
// field. Used only on the zero-recurring branch of SpawnMonsters, to keep the
// "no spawn points" log from lying about an already-disarmed one-time field
// (FR-4.1).
func (r *SpawnPointRegistry) CountOneTime(ctx context.Context, mapKey character.MapKey) (int, error) {
	n, err := r.oneTime.Len(ctx, mapKey.Tenant, mapKey)
	if err != nil {
		return 0, err
	}
	return int(n), nil
}
```

- [x] **Step 7: Run the package tests**

```bash
go test ./map/monster/ -v
```

Expected: PASS, including the pre-existing `TestInitializeForMap`, `TestCount`, `TestReserveEligibleSpawnPoints_*`, `TestResetCooldown`, `TestMapKeyIsolation`, `TestSpawnMonsters_*` and all four `TestSpawnPointRegistry_FlushTenant_*` / `TestFlushTenant_*` cases. If `TestFlushTenant_MatchesWriteKeyUnderEnvPrefix` (`registry_test.go:135`) asserts a literal key without `v2:`, update the expected literal to the `v2:` shape and no other part of the test.

- [x] **Step 8: Build and vet the module**

```bash
go build ./... && go vet ./...
```

Expected: exit 0.

- [x] **Step 9: Commit**

```bash
git add map/monster/registry.go map/monster/registry_test.go
git commit -m "feat(maps): split spawn registry into recurring, one-time and meta hashes"
```

---

## Task 3: Atomic one-time claim and re-arm

### Files

- `services/atlas-maps/atlas.com/maps/map/monster/registry.go` — `claimOneTimeScript`, `ClaimOneTimeSpawnPoints`, `RearmOneTime`
- `services/atlas-maps/atlas.com/maps/map/monster/registry_test.go` — claim/re-arm/concurrency/flush tests

Patterns to copy: `services/atlas-maps/atlas.com/maps/map/monster/registry.go:130-172` (`reserveEligibleScript` + its `[]interface{}` result decoding in `ReserveEligibleSpawnPoints`, `registry.go:259-283`) — the claim decodes an HGETALL-shaped flat array the same way; `services/atlas-maps/atlas.com/maps/map/monster/processor_test.go:490-542` (`TestReserveEligibleSpawnPoints_ConcurrentNoDoubleReserve`) for the goroutine-burst concurrency shape.

Module root: `services/atlas-maps/atlas.com/maps`.

### Interfaces

- Consumes from Task 2: `r.metaKey`, `r.oneTimeKey`, `metaFieldOneTimeFired`, `CountOneTime`.
- Produces:
  - `(*SpawnPointRegistry).ClaimOneTimeSpawnPoints(ctx context.Context, mapKey character.MapKey) ([]*CooldownSpawnPoint, error)`
  - `(*SpawnPointRegistry).RearmOneTime(ctx context.Context, mapKey character.MapKey) (bool, error)`

  Task 4 consumes `ClaimOneTimeSpawnPoints`; Task 6 consumes `RearmOneTime`.

- [x] **Step 1: Write the failing tests**

Append to `map/monster/registry_test.go`. Setup shape copied from Task 2's tests (miniredis client + `newTestRegistry` + `tenant.Create(uuid.New(), "GMS", 83, 1)`).

`TestClaimOneTimeSpawnPoints` — subtests:

| subtest | field seeded with | first call returns | second call returns |
|---|---|---|---|
| `armed field fires the full batch` | 10 one-time points, ids 1..10, all template `9300044`, mobTime `-1` | 10 points, ids `{1..10}` as a set, every `Template == 9300044` | 0 points |
| `recurring-only field claims nothing` | 4 recurring points, mobTime `0` | 0 points | 0 points |
| `mixed field fires only the one-time subset` | id 1 mobTime `0`, id 2 mobTime `-1`, id 3 mobTime `30` | 1 point, id `2` | 0 points |
| `unseeded field returns nothing` | (no `InitializeForMap` call) | 0 points | 0 points |

For the first subtest, also assert after the first call that `client.HGet(ctx, r.metaKey(mapKey), metaFieldOneTimeFired)` returns a non-empty string parseable as an `int64` millisecond timestamp within 60s of `time.Now().UnixMilli()`, and that the one-time hash still has HLEN 10 (the claim does not consume the points, it only disarms the field).

For the second subtest, also assert `r.Count(ctx, mapKey) == 4` afterwards and that every recurring point's `NextSpawnAt` is unchanged — the claim must not touch the recurring hash.

`TestClaimOneTimeSpawnPoints_ConcurrentFiresExactlyOnce` — seed 10 one-time points, launch 8 goroutines through a `sync.WaitGroup` each calling `ClaimOneTimeSpawnPoints` on the same `mapKey`, collect the returned lengths under a mutex. Assert exactly one goroutine got `len == 10` and the other seven got `len == 0`. (FR-2.3.)

`TestRearmOneTime` — subtests:

| subtest | sequence | expectations |
|---|---|---|
| `fired field re-arms once` | seed 10 one-time; claim; `RearmOneTime` | first `RearmOneTime` returns `true`; a second immediately after returns `false` |
| `never-fired field returns false` | seed 10 one-time; `RearmOneTime` without claiming | returns `false` |
| `re-armed field fires a fresh full batch` | seed 10; claim (10); `RearmOneTime` (true); claim again | second claim returns 10 points |
| `re-arm leaves the recurring hash untouched` | seed mixed (2 recurring with a stamped `NextSpawnAt`, 1 one-time); claim; `RearmOneTime` | recurring HLEN still 2 and both `NextSpawnAt` values byte-identical to before, read via `GetSpawnPointsForMap` (FR-3.3) |

`TestRearmOneTime_IsPerFieldKey` — seed and claim the same map id on channel 0 and channel 1 (two `field.NewBuilder(world.Id(0), channel.Id(0)/channel.Id(1), _map.Id(920010920)).Build()` keys). `RearmOneTime` on the channel-0 key, then claim both. Assert the channel-0 claim returns 10 and the channel-1 claim returns 0. Repeat the same shape for two distinct `SetInstance(uuid.New())` instances on channel 0. (FR-3.4.)

`TestFlushTenant_ReArmsDisarmedField` — seed 10 one-time; claim (10); `FlushTenant`; `InitializeForMap` again with the same stub; claim. Assert the post-flush claim returns 10. (FR-3.5 behavior half.)

- [x] **Step 2: Run the tests and verify they fail**

```bash
go test ./map/monster/ -run 'TestClaimOneTime|TestRearmOneTime|TestFlushTenant_ReArms' -v
```

Expected: FAIL — `r.ClaimOneTimeSpawnPoints undefined`, `r.RearmOneTime undefined`.

- [x] **Step 3: Add the claim script**

In `map/monster/registry.go`, after `resetCooldownScript`:

```go
// claimOneTimeScript atomically claims a field's one-time batch and returns it
// in the same round trip.
//
// HSETNX is the claim: a single atomic Redis write, strictly stronger than the
// read-then-write FR-2.3 forbids. Folding the payload fetch into the same
// script makes both the firing path and the already-disarmed path exactly one
// round trip, which is the PRD §8 performance requirement.
//
// The meta hash is claimed even when the one-time hash is empty. That costs one
// wasted HSETNX on the first pass for a recurring-only field and keeps the
// script single-branch; the field is then "disarmed with nothing to fire",
// which is indistinguishable from "fired" and equally correct.
//
// KEYS[1] = meta hash, KEYS[2] = one-time hash
// ARGV[1] = nowMilli
// Returns: {} when the field is already disarmed or has no one-time points,
//          otherwise the HGETALL of the one-time hash.
var claimOneTimeScript = goredis.NewScript(`
if redis.call('HSETNX', KEYS[1], 'onetimeFired', ARGV[1]) == 0 then
    return {}
end
local entries = redis.call('HGETALL', KEYS[2])
if #entries == 0 then
    return {}
end
return entries
`)
```

- [x] **Step 4: Add `ClaimOneTimeSpawnPoints` and `RearmOneTime`**

```go
// ClaimOneTimeSpawnPoints disarms the field and returns its one-time spawn
// points, or nil if the field was already disarmed or has none. Exactly one
// concurrent caller can receive a non-empty batch.
//
// Crash window: if the pod dies between this returning and CreateMonster being
// issued, the batch is lost until the field re-arms. That is inherent to any
// claim-then-act split, and CreateMonster is already fire-and-forget on the
// recurring path.
func (r *SpawnPointRegistry) ClaimOneTimeSpawnPoints(ctx context.Context, mapKey character.MapKey) ([]*CooldownSpawnPoint, error) {
	result, err := claimOneTimeScript.Run(ctx, r.client,
		[]string{r.metaKey(mapKey), r.oneTimeKey(mapKey)},
		time.Now().UnixMilli()).Result()
	if err != nil {
		return nil, err
	}

	arr, ok := result.([]interface{})
	if !ok || len(arr) == 0 {
		return nil, nil
	}

	var claimed []*CooldownSpawnPoint
	for i := 1; i < len(arr); i += 2 {
		valueStr, ok := arr[i].(string)
		if !ok {
			continue
		}
		var stored storedSpawnPoint
		if err := json.Unmarshal([]byte(valueStr), &stored); err != nil {
			continue
		}
		claimed = append(claimed, fromStored(stored))
	}

	return claimed, nil
}

// RearmOneTime clears a field's one-time fired marker, returning true iff the
// marker was actually present — i.e. iff the field had fired and is now armed
// again. The caller uses that bool to scope the DESTROY_FIELD despawn to fields
// that really fired a batch, so the 4,207 unaffected maps keep behaving exactly
// as they do on main (design D7).
//
// HDEL is atomic, so no Lua is needed. It touches only the meta hash: the
// recurring hash and its cooldown state are untouched (FR-3.3).
func (r *SpawnPointRegistry) RearmOneTime(ctx context.Context, mapKey character.MapKey) (bool, error) {
	n, err := r.client.HDel(ctx, r.metaKey(mapKey), metaFieldOneTimeFired).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
```

Note the decode loop starts at `i = 1` and steps by 2, taking values from an HGETALL-shaped `[field, value, field, value, ...]` array — unlike `ReserveEligibleSpawnPoints`, whose array has a leading total element.

- [x] **Step 5: Run the tests and verify they pass**

```bash
go test ./map/monster/ -v
```

Expected: PASS, all new and pre-existing tests.

- [x] **Step 6: Run the package with the race detector**

```bash
go test -race ./map/monster/
```

Expected: PASS, no race reports — this is the gate on the concurrency test.

- [x] **Step 7: Commit**

```bash
git add map/monster/registry.go map/monster/registry_test.go
git commit -m "feat(maps): atomic one-time spawn claim and per-field re-arm"
```

---

## Task 4: Fire the one-time batch in `SpawnMonsters`

### Files

- `services/atlas-maps/atlas.com/maps/map/monster/processor.go` — claim-and-fire before the count; recurring-only denominator; remove the silent `totalCount == 0` return; FR-4 logs
- `services/atlas-maps/atlas.com/maps/map/monster/processor_test.go` — new `SpawnMonsters` cases
- `services/atlas-maps/atlas.com/maps/monster/processor.go` — read-only; `CreateMonster(transactionId, field, monsterId, x, y, fh, team)` is the call being made

Patterns to copy: `services/atlas-maps/atlas.com/maps/map/monster/processor_test.go:757-830` (`TestSpawnMonsters_CooldownValidation` — the `ProcessorImpl{l, ctx: tctx, t: te, dp, cp, mp}` struct literal, the `mockCharProc.charactersInMap[mapKey]` / `mockMonsterProc.monstersInMap[mapKey]` seeding, and the `time.Sleep(500 * time.Millisecond)` that lets the `routine.Go` spawns land before `GetCreatedMonsters()` is read).

Module root: `services/atlas-maps/atlas.com/maps`.

### Interfaces

- Consumes from Task 3: `ClaimOneTimeSpawnPoints`; from Task 2: `Count` (recurring), `CountOneTime`.
- Produces: no new exported surface. `SpawnMonsters(transactionId uuid.UUID, f field.Model) error` signature unchanged.

- [x] **Step 1: Write the failing tests**

Append to `map/monster/processor_test.go`. Every case calls `registry.Reset(ctx)` first, as the existing tests do.

`TestSpawnMonsters_OneTimeBatch` — subtests:

| subtest | spawn points | characters | `monstersInMap` | expected `GetCreatedMonsters()` |
|---|---|---|---|---|
| `solo entrant fires all ten` (`920010920`) | 10 points, ids 1..10, template `9300044`, mobTime `-1`, distinct X = `100*id` | 1 (`[]uint32{1001}`) | 0 | exactly 10, one per spawn point, every `MonsterId == 9300044`, the X set equal to `{100,200,…,1000}` |
| `eight-point map fires all eight` (`920010910`) | 8 points, template `9300044`, mobTime `-1` | 1 | 0 | exactly 8 |
| `second pass fires nothing more` | 10 points, mobTime `-1` | 1 | 10 after the first pass | first `SpawnMonsters` creates 10; after `mockMonsterProc.Reset()` and setting `monstersInMap` to 10, a second `SpawnMonsters` creates 0 |
| `second character does not re-fire` | 10 points, mobTime `-1` | 1, then 2 (`[]uint32{1001, 1002}`) | 10 for the second pass | second pass creates 0 |

`TestSpawnMonsters_OneTimeIgnoresCharacterCount` (FR-2.2) — 10 one-time points, 1 character. Assert 10 monsters are created, and assert explicitly that `p.getMonsterMax(1, 10) == 8` — i.e. the batch size is not the rate-formula answer.

`TestSpawnMonsters_MixedMapUsesRecurringDenominator` (FR-2.5) — spawn points: 5 recurring (ids 1..5, mobTime `0`, template `100100`) plus 3 one-time (ids 6..8, mobTime `-1`, template `9300044`). 2 characters, `monstersInMap` 0.

Assert:
- `registry.Count(ctx, mapKey) == 5` and `registry.CountOneTime(ctx, mapKey) == 3`
- `p.getMonsterMax(2, 5) == 4` — the denominator is 5, not 8
- created monsters total `3 + 4 == 7`: exactly 3 with `MonsterId == 9300044` and exactly 4 with `MonsterId == 100100`

`TestSpawnMonsters_HiddenPointNeverSpawns` (FR-1.4, `600020300` shape) — one spawn point, id 1, template `9400545`, mobTime `0`, `Hide: true`. 1 character, `monstersInMap` 0. Assert 0 monsters created and `registry.Count == 0` and `registry.CountOneTime == 0`.

`TestSpawnMonsters_RecurringOnlyRegression` (`104040000` shape) — 39 spawn points, all mobTime `0`, template `100100`, 2 characters, `monstersInMap` 0. Assert created count `== p.getMonsterMax(2, 39)` and that a second immediate pass creates 0 more (all reserved points are on the 5s default cooldown). This is the `main`-parity pin.

`TestSpawnMonsters_ZeroSpawnPointsLogsBothCounts` (FR-4.1) — no spawn points at all, 1 character. Use `test.NewNullLogger()` from `github.com/sirupsen/logrus/hooks/test` (already imported by `map/processor_test.go`; add the import here), set `logger.SetLevel(logrus.DebugLevel)`, and assert one entry in `hook.AllEntries()` has message `Field [<f.Id()>] has no spawn points: one-time [0] recurring [0].`. Assert `SpawnMonsters` returned `nil`.

- [x] **Step 2: Run the tests and verify they fail**

```bash
go test ./map/monster/ -run 'TestSpawnMonsters_OneTime|TestSpawnMonsters_Mixed|TestSpawnMonsters_Hidden|TestSpawnMonsters_RecurringOnly|TestSpawnMonsters_ZeroSpawnPoints' -v
```

Expected: FAIL — the one-time subtests create 0 monsters, and the zero-point subtest finds no log entry.

- [x] **Step 3: Fire the batch and switch to the recurring denominator**

In `map/monster/processor.go`, replace lines 99-121 (from `totalCount, err := registry.Count(...)` through the `toSpawn <= 0` return) with:

```go
	// Claim and fire the field's one-time batch before anything else consults a
	// count. The claim is a single atomic HSETNX inside Lua, so exactly one of
	// the concurrent triggers (character-enter, and the 10s NewRespawn task)
	// can win it; the loser sees an empty batch (FR-2.1..2.4).
	fired, err := registry.ClaimOneTimeSpawnPoints(p.ctx, mapKey)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to claim one-time spawn points for field [%s].", f.Id())
		return err
	}
	if len(fired) > 0 {
		p.l.Debugf("Firing [%d] one-time spawn points for field [%s].", len(fired), f.Id())
		for _, csp := range fired {
			sp := csp.SpawnPoint
			routine.Go(p.l, p.ctx, func(_ context.Context) {
				p.mp.CreateMonster(transactionId, f, sp.Template, sp.X, sp.Y, sp.Fh, sp.Team)
			})
		}
	}

	// Recurring points only: a one-time batch must not inflate the recurring
	// population target (FR-2.5).
	recurringCount, err := registry.Count(p.ctx, mapKey)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to count spawn points for field [%s].", f.Id())
		return err
	}
	if recurringCount == 0 {
		oneTimeCount, cerr := registry.CountOneTime(p.ctx, mapKey)
		if cerr != nil {
			p.l.WithError(cerr).Errorf("Failed to count one-time spawn points for field [%s].", f.Id())
			return cerr
		}
		if oneTimeCount == 0 {
			p.l.Debugf("Field [%s] has no spawn points: one-time [0] recurring [0].", f.Id())
		}
		return nil
	}

	// monstersInMap counts EVERY monster in the field, including the one-time
	// ones. On a mixed map, once the one-time batch is alive it will usually
	// exceed monstersMax, so recurring spawns are suppressed until enough
	// one-time monsters are killed. This is deliberate, matches the reference
	// implementation's map-wide respawn deficit, and is not a regression: no
	// mixed map spawns anything from its one-time set today. Attributing the
	// count per-origin would require an atlas-monsters contract change, which
	// is a PRD non-goal. Do not "fix" this.
	monstersInMap, err := p.mp.CountInMap(transactionId, f)
	if err != nil {
		// Skip this pass rather than assuming zero monsters: a transient count
		// failure treated as zero would spawn the full deficit and over-populate
		// the map. Under-spawning for one tick is the safe direction.
		p.l.WithError(err).Errorf("Unable to count monsters in map; skipping spawn for field [%s] to avoid over-spawn.", f.Id())
		return err
	}

	monstersMax := p.getMonsterMax(c, recurringCount)
	toSpawn := monstersMax - monstersInMap
	if toSpawn <= 0 {
		return nil
	}
```

The `routine.Go` closure captures `sp` by value because `sp := csp.SpawnPoint` is a fresh variable per iteration — the same shape as the existing recurring loop at `processor.go:140-146`.

Everything from `seed := time.Now().UnixNano() & 0x7fffffff` onward is unchanged.

- [x] **Step 4: Update the package doc comment**

The package comment at `map/monster/processor.go:1-18` and the `SpawnMonsters` doc comment at `:67-73` describe the old five-step flow. Update `SpawnMonsters`' comment to:

```go
// SpawnMonsters implements the core spawn logic with cooldown enforcement.
//
//  1. Initialize the spawn point registry for this field (lazy, one HTTP drain)
//  2. Return early if the field has no characters
//  3. Atomically claim and fire the field's one-time batch, uncapped (FR-2.1..2.4)
//  4. Compute the recurring deficit from the RECURRING spawn-point count only
//  5. Atomically reserve up to that many eligible recurring points and spawn them
```

- [x] **Step 5: Run the tests and verify they pass**

```bash
go test ./map/monster/ -v
```

Expected: PASS, all new and pre-existing tests.

- [x] **Step 6: Build and vet**

```bash
go build ./... && go vet ./...
```

Expected: exit 0.

- [x] **Step 7: Commit**

```bash
git add map/monster/processor.go map/monster/processor_test.go
git commit -m "feat(maps): fire one-time spawn batches and scope getMonsterMax to recurring points"
```

---

## Task 5: `DESTROY_FIELD` command envelope and producer

### Files

- `services/atlas-maps/atlas.com/maps/kafka/message/monster/kafka.go` — `EnvCommandTopic`, `CommandTypeDestroyField`, `FieldCommand[E]`, `DestroyFieldBody`
- `services/atlas-maps/atlas.com/maps/map/producer.go` — `destroyFieldCommandProvider`
- `services/atlas-maps/atlas.com/maps/map/producer_test.go` — **new file**; the cross-service seam assertion
- `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka.go:12-14,26,103,252-259` — **read-only**; the consumer contract being matched (`EnvCommandTopic = "COMMAND_TOPIC_MONSTER"`, `CommandTypeDestroyField = "DESTROY_FIELD"`, `destroyFieldCommandBody struct{}`, `fieldCommand[E]`)
- `services/atlas-monsters/docs/kafka.md:461-473` — **read-only**; the documented `DESTROY_FIELD` JSON
- `deploy/k8s/base/env-configmap.yaml:68` — **read-only**; `COMMAND_TOPIC_MONSTER` is already in the shared `atlas-env` ConfigMap that `atlas-maps` gets via `envFrom`, so no manifest change is needed

Patterns to copy: `services/atlas-maps/atlas.com/maps/kafka/message/mapactions/kafka.go:12-28` (the exact `EnvCommandTopic` / `CommandType*` / `Command[E]` shape) and `services/atlas-maps/atlas.com/maps/map/producer.go:47-63` (`enterMapActionsProvider` — an existing cross-service command provider living in `map/producer.go`, which is why `destroyFieldCommandProvider` goes here rather than in a new `monster/producer.go`).

Module root: `services/atlas-maps/atlas.com/maps`.

### Interfaces

- Produces:
  - `monsterKafka.EnvCommandTopic topic.Token`, `monsterKafka.CommandTypeDestroyField string`
  - `monsterKafka.FieldCommand[E any]` with JSON keys **exactly** `worldId`, `channelId`, `mapId`, `instance`, `type`, `body` — and **no `transactionId`**, because `atlas-monsters`' `fieldCommand` has none
  - `monsterKafka.DestroyFieldBody struct{}`
  - package-private `destroyFieldCommandProvider(f field.Model) model.Provider[[]kafka.Message]` in package `_map`

  Task 6 consumes `EnvCommandTopic` and `destroyFieldCommandProvider`.

- [x] **Step 1: Write the failing test**

New file `map/producer_test.go`, package `_map`. This is the cross-service seam test CLAUDE.md's review gate requires: `verify.sh` cannot see envelope drift between two services, and the consumer's body is `struct{}`, so a silent field-name drift fails open with no error anywhere.

`TestDestroyFieldCommandProvider_MatchesConsumerEnvelope`:

- Build `f := field.NewBuilder(world.Id(1), channel.Id(2), _map.Id(920010920)).SetInstance(uuid.MustParse("11111111-2222-3333-4444-555555555555")).Build()`.
- Call `destroyFieldCommandProvider(f)()`, assert exactly 1 message and no error.
- Unmarshal `msgs[0].Value` into `map[string]json.RawMessage` and assert the key set is **exactly** `{"worldId", "channelId", "mapId", "instance", "type", "body"}` — no more, no fewer. Fail with the actual sorted key list on mismatch.
- Assert the values: `worldId` = `1`, `channelId` = `2`, `mapId` = `920010920`, `instance` = `"11111111-2222-3333-4444-555555555555"`, `type` = `"DESTROY_FIELD"`, `body` = `{}`.
- Assert `msgs[0].Key` equals `producer.CreateKey(int(f.MapId()))`.

- [x] **Step 2: Run the test and verify it fails**

```bash
go test ./map/ -run TestDestroyFieldCommandProvider -v
```

Expected: FAIL — `undefined: destroyFieldCommandProvider`.

- [x] **Step 3: Add the envelope to the maps-side kafka message package**

In `kafka/message/monster/kafka.go`, extend the existing const blocks and append the types:

```go
const (
	EnvEventTopicMonsterStatus topic.Token = "EVENT_TOPIC_MONSTER_STATUS"
	EnvCommandTopic            topic.Token = "COMMAND_TOPIC_MONSTER"
)

const (
	CommandTypeDestroyField = "DESTROY_FIELD"
)
```

and, after `DamageEntry`:

```go
// FieldCommand is a field-scoped command on COMMAND_TOPIC_MONSTER. The field
// names and types mirror atlas-monsters' fieldCommand
// (services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka.go)
// field-for-field. Note there is deliberately NO transactionId: the consumer's
// envelope has none, and an extra key would be silently dropped rather than
// rejected. Any drift here fails open, which is why map/producer_test.go pins
// the emitted key set exactly.
type FieldCommand[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	Type      string     `json:"type"`
	Body      E          `json:"body"`
}

// DestroyFieldBody matches atlas-monsters' destroyFieldCommandBody: empty.
type DestroyFieldBody struct{}
```

The file already imports `uuid`, `channel`, `_map`, `world` and `topic`; no import change is needed.

- [x] **Step 4: Add the provider**

In `map/producer.go`, add the import `monsterKafka "atlas-maps/kafka/message/monster"` and append:

```go
// destroyFieldCommandProvider despawns every monster in a field. Emitted only
// when a field that actually fired a one-time batch empties (design D7): without
// it, "party kills 6 of 10 Lucidas, leaves, next party enters" yields 4
// survivors plus 10 fresh monsters. Gating on the fired flag keeps the 4,207
// unaffected maps bit-identical to main.
func destroyFieldCommandProvider(f field.Model) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(f.MapId()))
	value := &monsterKafka.FieldCommand[monsterKafka.DestroyFieldBody]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		Type:      monsterKafka.CommandTypeDestroyField,
		Body:      monsterKafka.DestroyFieldBody{},
	}
	return producer.SingleMessageProvider(key, value)
}
```

- [x] **Step 5: Run the test and verify it passes**

```bash
go test ./map/ -run TestDestroyFieldCommandProvider -v
```

Expected: PASS.

- [x] **Step 6: Build and vet**

```bash
go build ./... && go vet ./...
```

Expected: exit 0. `destroyFieldCommandProvider` is referenced only by the test until Task 6; `go vet` does not flag an unused package-level function.

- [x] **Step 7: Commit**

```bash
git add kafka/message/monster/kafka.go map/producer.go map/producer_test.go
git commit -m "feat(maps): add DESTROY_FIELD command envelope and provider"
```

---

## Task 6: Re-arm on field empty in `map.ProcessorImpl.Exit`

### Files

- `services/atlas-maps/atlas.com/maps/map/processor.go:105-110` — the `Exit` field-empties block
- `services/atlas-maps/atlas.com/maps/map/processor_test.go` — `Exit` re-arm and `DESTROY_FIELD` cases

Patterns to copy: `services/atlas-maps/atlas.com/maps/map/processor_test.go:297-364` (`TestProcessorImpl_Exit` — `createTestContext()`, `createTestProcessor(logger, ctx, mockCp, mockPp)`, `message.NewBuffer()`, `buf.GetAll()`); `services/atlas-maps/atlas.com/maps/map/processor_test.go:32-42` (`TestMain` already calls `monster2.InitRegistry(rc)` against miniredis, so the registry is live in this package's tests).

Module root: `services/atlas-maps/atlas.com/maps`.

### Interfaces

- Consumes from Task 3: `monster2.GetRegistry().RearmOneTime(ctx, character.MapKey) (bool, error)`; from Task 5: `monsterKafka.EnvCommandTopic`, `destroyFieldCommandProvider`.
- Produces: no signature change. `Exit(mb *message.Buffer) func(uuid.UUID, field.Model, uint32) error`.

- [x] **Step 1: Write the failing tests**

Append to `map/processor_test.go`. Each case builds the field's registry state directly via `monster2.GetRegistry()` before calling `Exit`, and reads the tenant out of `createTestContext()` so the `MapKey` matches (`te, _ := tenant.FromContext(ctx)`; add the import if absent).

`TestProcessorImpl_Exit_RearmsAndDestroysOnEmpty` — subtests:

| subtest | registry state before `Exit` | `GetCharactersInMap` returns | expect `RearmOneTime` observable | expect `DESTROY_FIELD` messages |
|---|---|---|---|---|
| `last character leaves a fired field` | seeded with 10 one-time points, then claimed (disarmed) | `nil` (empty) | a post-`Exit` `ClaimOneTimeSpawnPoints` returns 10 points | 1 |
| `characters remain` | seeded + claimed | `[]uint32{2222}` | a post-`Exit` claim returns 0 points | 0 |
| `never-fired field empties` | seeded with 4 recurring points, never claimed | `nil` | n/a | 0 |
| `unseeded field empties` | nothing seeded | `nil` | n/a | 0 |

Set the mock's behavior with `mockCp.getCharactersInMapFunc = func(uuid.UUID, field.Model) ([]uint32, error) { return ..., nil }` (`map/processor_test.go:50,65-70`).

In every subtest also assert the `CHARACTER_EXIT` event is still buffered: `len(buf.GetAll()[mapKafka.EnvEventTopicMapStatus]) == 1`. Read the destroy messages with `buf.GetAll()[monsterKafka.EnvCommandTopic]`.

For the first subtest, additionally unmarshal the single destroy message and assert `type == "DESTROY_FIELD"` and `mapId` equals the field's map id — the full envelope shape is already pinned by Task 5's test.

`TestProcessorImpl_Exit_RearmIsPerFieldKey` (FR-3.4) — seed and claim the same map id on channel 0 and channel 1. Call `Exit` on the channel-0 field with `getCharactersInMapFunc` returning empty. Assert a subsequent claim on channel 0 returns 10 points and a subsequent claim on channel 1 returns 0.

`TestProcessorImpl_Exit_LogsRearm` (FR-4.3) — the fired-field case with `logger.SetLevel(logrus.DebugLevel)`; assert a hook entry with message `Re-armed one-time spawn points for field [<f.Id()>].`.

- [x] **Step 2: Run the tests and verify they fail**

```bash
go test ./map/ -run 'TestProcessorImpl_Exit_Rearm|TestProcessorImpl_Exit_Logs' -v
```

Expected: FAIL — no `DESTROY_FIELD` message is buffered and a post-`Exit` claim returns 0.

- [x] **Step 3: Add the field-empties block**

Replace `map/processor.go:105-110` with:

```go
func (p *ProcessorImpl) Exit(mb *message.Buffer) func(transactionId uuid.UUID, f field.Model, characterId uint32) error {
	return func(transactionId uuid.UUID, f field.Model, characterId uint32) error {
		p.cp.Exit(transactionId, f, characterId)

		// The departing character may have emptied the field. cp.Exit mutates
		// the in-memory character registry synchronously, so this read already
		// observes the departure — no ordering hazard.
		//
		// Exit is the single funnel for logout, TransitionMap and
		// TransitionChannel, so all three paths re-arm through this one block.
		// Both transition paths call Exit(oldField) then Enter(newField) on
		// distinct field keys, so re-arming the old field cannot race the new
		// field's spawn pass.
		if remaining, err := p.cp.GetCharactersInMap(transactionId, f); err == nil && len(remaining) == 0 {
			mapKey := character.MapKey{Tenant: tenant.MustFromContext(p.ctx), Field: f}
			rearmed, rerr := monster2.GetRegistry().RearmOneTime(p.ctx, mapKey)
			if rerr != nil {
				p.l.WithError(rerr).Errorf("Failed to re-arm one-time spawn points for field [%s].", f.Id())
			} else if rearmed {
				p.l.Debugf("Re-armed one-time spawn points for field [%s].", f.Id())
				// Scoped deliberately: only a field that actually fired a batch
				// has its residual monsters destroyed. A recurring-only field
				// never fires, so it never emits, and its emptying behavior is
				// bit-identical to main (design D7).
				_ = mb.Put(monsterKafka.EnvCommandTopic, destroyFieldCommandProvider(f))
			}
		}

		return mb.Put(mapKafka.EnvEventTopicMapStatus, exitMapProvider(transactionId, f, characterId))
	}
}
```

Add these imports to `map/processor.go`:

```go
	monsterKafka "atlas-maps/kafka/message/monster"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
```

`character` and `monster2 "atlas-maps/map/monster"` are already imported (`map/processor.go:8-9`).

**Rebase note (FR-3.2):** PR #1566 / task-278 adds a field-empties block at this exact site and is confirmed *not* in this branch's history (`git log --oneline main | grep task-278` → no match). If #1566 lands first, merge into a **single** `len(remaining) == 0` block carrying both bodies — never two adjacent emptiness checks.

- [x] **Step 4: Run the tests and verify they pass**

```bash
go test ./map/ -v
```

Expected: PASS, including the pre-existing `TestProcessorImpl_Exit`, `TestProcessorImpl_ExitAndEmit`, `TestProcessorImpl_TransitionMap`, `TestProcessorImpl_TransitionChannel` cases. Those use a `mockCharacterProcessor` whose `getCharactersInMapFunc` is nil and therefore returns `nil, nil` — an empty field — so they will now exercise the re-arm path against an unseeded field, which is a no-op returning `false` and emits nothing.

- [x] **Step 5: Run the module tests with the race detector**

```bash
go test -race ./...
```

Expected: PASS.

- [x] **Step 6: Commit**

```bash
git add map/processor.go map/processor_test.go
git commit -m "feat(maps): re-arm one-time spawns and despawn residuals when a field empties"
```

---

## Task 7: Remove the dead `Spawnable` surface

### Files

- `services/atlas-maps/atlas.com/maps/data/map/monster/processor.go` — delete `SpawnableSpawnPointProvider`, `GetSpawnableSpawnPoints`, `Spawnable` and their two interface entries
- `services/atlas-maps/atlas.com/maps/map/monster/processor_test.go:743-755` — delete the two matching `mockDataProcessor` methods
- `services/atlas-maps/docs/domain.md:350-352` — replace the two `Spawnable*` bullets

Module root: `services/atlas-maps/atlas.com/maps`.

### Interfaces

- Consumes: nothing new.
- Produces: `monster.Processor` shrinks to exactly two methods.

- [x] **Step 1: Confirm there are no remaining callers**

Run from the worktree root:

```bash
grep -rn "GetSpawnableSpawnPoints\|SpawnableSpawnPointProvider\|Spawnable(" services/
```

Expected: hits only in `data/map/monster/processor.go`, `map/monster/processor_test.go`, and `services/atlas-maps/docs/domain.md` — the three files this task edits. If any other file appears, STOP and report: an earlier task left a caller behind.

- [x] **Step 2: Shrink the interface and delete the methods**

`data/map/monster/processor.go` — the interface becomes:

```go
type Processor interface {
	SpawnPointProvider(mapId _map.Id) model.Provider[[]SpawnPoint]
	GetSpawnPoints(mapId _map.Id) ([]SpawnPoint, error)
}
```

Delete `SpawnableSpawnPointProvider` (lines 46-48), `GetSpawnableSpawnPoints` (54-56) and `Spawnable` (58-60). A predicate named `Spawnable` that no longer means "spawnable" is a live trap for the next reader; `Classify` is the single classification authority now.

- [x] **Step 3: Shrink the mock**

In `map/monster/processor_test.go`, delete `func (m *mockDataProcessor) SpawnableSpawnPointProvider(...)` and `func (m *mockDataProcessor) GetSpawnableSpawnPoints(...)`. `SpawnPointProvider` and `GetSpawnPoints` stay and already return the unfiltered `mockSpawnPoints` slice.

- [x] **Step 4: Update the service docs**

In `services/atlas-maps/docs/domain.md`, replace the two bullets at lines 350 and 352 with:

```markdown
- GetSpawnPoints: Gets every spawn point on a map, unfiltered
- Classify: Partitions spawn points into recurring (MobTime >= 0), one-time (MobTime < 0) and hidden (Hide == true) buckets
```

Keep the surrounding bullets and the `SpawnPointProvider` line as-is.

- [x] **Step 5: Build, vet and test the whole module**

```bash
go build ./... && go vet ./... && go test ./...
```

Expected: exit 0, all packages PASS. A compile error here means a caller was missed in Step 1.

- [x] **Step 6: Run the repo verification gate**

From the worktree root:

```bash
tools/verify.sh
```

Expected: exit 0, flagless. `--quick` / `--no-docker` do not count.

- [x] **Step 7: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/data/map/monster/processor.go \
        services/atlas-maps/atlas.com/maps/map/monster/processor_test.go \
        services/atlas-maps/docs/domain.md
git commit -m "refactor(maps): drop the Spawnable predicate now that Classify owns classification"
```

---

## PR body requirements

The PR body must name these as **behavior changes**, not bury them in a test (design D4, D2, D8):

1. `600020300` (Wolf Spider Cavern, `9400545`) and `800020130` (Encounter with the Buddha, `9400013`) each lose their single `hide = true` spawn point and become empty of static monsters. Both currently spawn on `main`. PRD-sanctioned under FR-1.4.
2. The `v2:` key-schema token orphans every pre-existing `maps:spawn` key. The orphans are inert, bounded at one hash per field per tenant ever visited, and are reaped by the next `DATA_UPDATED` flush, whose SCAN pattern is namespace-wide. No manual Redis cleanup is required.
3. On the 61 mixed maps, recurring spawns are suppressed while the one-time batch is alive, because `monstersInMap` counts every monster in the field. This matches the reference implementation and is not a regression from `main`, where no mixed map spawns its one-time set at all.
4. Fields that fire a one-time batch now emit `DESTROY_FIELD` to `atlas-monsters` when they empty. Recurring-only fields never fire and never emit.

Plus the mandatory live smoke test section: enter `920010920`, observe 10 Lucidas client-side, kill them, confirm no respawn over three `NewRespawn` ticks (>30s), leave with the last character, re-enter, confirm 10 again.
