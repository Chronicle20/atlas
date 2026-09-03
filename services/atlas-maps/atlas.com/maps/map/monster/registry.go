package monster

import (
	"atlas-maps/map/character"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	monster2 "atlas-maps/data/map/monster"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	atlasredis "github.com/Chronicle20/atlas/libs/atlas-redis"
)

// storedSpawnPoint is the JSON-serializable format for Redis hash storage.
type storedSpawnPoint struct {
	Id          uint32 `json:"id"`
	Template    uint32 `json:"template"`
	MobTime     int32  `json:"mobTime"`
	Team        int8   `json:"team"`
	Cy          int16  `json:"cy"`
	F           uint32 `json:"f"`
	Fh          int16  `json:"fh"`
	Rx0         int16  `json:"rx0"`
	Rx1         int16  `json:"rx1"`
	X           int16  `json:"x"`
	Y           int16  `json:"y"`
	NextSpawnAt int64  `json:"nextSpawnAt"`
}

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

var (
	registryInstance *SpawnPointRegistry
	registryOnce     sync.Once
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

// GetRegistry returns the singleton SpawnPointRegistry instance.
func GetRegistry() *SpawnPointRegistry {
	return registryInstance
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

func toStored(sp monster2.SpawnPoint, nextSpawnAt time.Time) storedSpawnPoint {
	return storedSpawnPoint{
		Id: sp.Id, Template: sp.Template, MobTime: sp.MobTime, Team: sp.Team,
		Cy: sp.Cy, F: sp.F, Fh: sp.Fh, Rx0: sp.Rx0, Rx1: sp.Rx1, X: sp.X, Y: sp.Y,
		NextSpawnAt: nextSpawnAt.UnixMilli(),
	}
}

func fromStored(s storedSpawnPoint) *CooldownSpawnPoint {
	return &CooldownSpawnPoint{
		SpawnPoint: monster2.SpawnPoint{
			Id: s.Id, Template: s.Template, MobTime: s.MobTime, Team: s.Team,
			Cy: s.Cy, F: s.F, Fh: s.Fh, Rx0: s.Rx0, Rx1: s.Rx1, X: s.X, Y: s.Y,
		},
		NextSpawnAt: time.UnixMilli(s.NextSpawnAt),
	}
}

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
//
//	recurring first, then one-time.
//
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

// reserveEligibleScript atomically selects up to `limit` eligible spawn points
// (NextSpawnAt <= now), stamps each selected point's cooldown, and returns only
// the reserved points. Folding eligibility selection and cooldown reservation
// into a single atomic script closes the check-then-reserve race that let
// concurrent spawn passes (character-enter + periodic task) each reserve the
// same points and over-spawn beyond the spawn-point count.
//
// Per-point cooldown mirrors the previous Go logic: MobTime seconds when
// MobTime > 0, otherwise the default cooldown. Eligible points are shuffled via
// a seeded LCG so that, when fewer than all eligible points are reserved, the
// selection is spread rather than always favouring the same points.
//
// KEYS[1] = spawn hash key
// ARGV[1] = nowMilli
// ARGV[2] = limit (max points to reserve)
// ARGV[3] = defaultCooldownMillis (used when a point's MobTime <= 0)
// ARGV[4] = shuffle seed
// Returns: [totalCount, field1, value1, ...] for the RESERVED points only.
var reserveEligibleScript = goredis.NewScript(`
local entries = redis.call('HGETALL', KEYS[1])
local now = tonumber(ARGV[1])
local limit = tonumber(ARGV[2])
local defaultCd = tonumber(ARGV[3])
local seed = tonumber(ARGV[4])
local total = math.floor(#entries / 2)

local eligible = {}
for i = 1, #entries, 2 do
    local field = entries[i]
    local data = cjson.decode(entries[i+1])
    if data.nextSpawnAt <= now then
        eligible[#eligible + 1] = {field = field, data = data}
    end
end

-- Fisher-Yates shuffle with a portable LCG (avoids non-deterministic math.random).
local n = #eligible
for i = n, 2, -1 do
    seed = (seed * 1103515245 + 12345) % 2147483648
    local j = (seed % i) + 1
    eligible[i], eligible[j] = eligible[j], eligible[i]
end

local result = {tostring(total)}
local reserved = 0
for i = 1, n do
    if reserved >= limit then break end
    local e = eligible[i]
    local cd = defaultCd
    if e.data.mobTime and e.data.mobTime > 0 then
        cd = e.data.mobTime * 1000
    end
    e.data.nextSpawnAt = now + cd
    local encoded = cjson.encode(e.data)
    redis.call('HSET', KEYS[1], e.field, encoded)
    result[#result + 1] = e.field
    result[#result + 1] = encoded
    reserved = reserved + 1
end
return result
`)

// resetCooldownScript resets cooldown for spawn points matching a template ID with MobTime > 0.
// Computes NextSpawnAt = nowMilli + (mobTime * 1000) per spawn point.
var resetCooldownScript = goredis.NewScript(`
local entries = redis.call('HGETALL', KEYS[1])
local templateId = tonumber(ARGV[1])
local nowMilli = tonumber(ARGV[2])
for i = 1, #entries, 2 do
    local field = entries[i]
    local value = entries[i+1]
    local data = cjson.decode(value)
    if data.template == templateId and data.mobTime > 0 then
        data.nextSpawnAt = nowMilli + (data.mobTime * 1000)
        redis.call('HSET', KEYS[1], field, cjson.encode(data))
    end
end
return 1
`)

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
//
//	otherwise the HGETALL of the one-time hash.
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

// InitializeForMap seeds a field's spawn points if it has not been seeded yet.
// The classification happens once, in memory, off a single paginated drain of
// atlas-data's /maps/{id}/monsters — two filtered providers would mean two
// fetches per field initialization (design D3).
func (r *SpawnPointRegistry) InitializeForMap(ctx context.Context, mapKey character.MapKey, dp monster2.Processor, l logrus.FieldLogger) error {
	seeded, err := r.meta.Exists(ctx, mapKey.Tenant, mapKey, metaFieldSeeded)
	if err != nil {
		return err
	}
	if !seeded {
		// Back-compat: a recurring hash written directly (e.g. via
		// SetSpawnPointsForMap, or a field seeded before the meta-seeded
		// marker existed) without the atomic seed script must not be
		// silently overwritten just because the marker itself is absent.
		n, herr := r.hashes.Len(ctx, mapKey.Tenant, mapKey)
		if herr != nil {
			return herr
		}
		if n > 0 {
			seeded = true
		}
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

// Count returns the number of RECURRING spawn points registered for a field.
// One-time points are deliberately excluded: this count is the denominator of
// getMonsterMax, and a one-time batch must not inflate the recurring
// population target (FR-2.5).
func (r *SpawnPointRegistry) Count(ctx context.Context, mapKey character.MapKey) (int, error) {
	n, err := r.hashes.Len(ctx, mapKey.Tenant, mapKey)
	if err != nil {
		return 0, err
	}
	return int(n), nil
}

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

// ReserveEligibleSpawnPoints atomically selects up to `limit` eligible spawn
// points, stamps their cooldowns, and returns the reserved points. This is the
// concurrency-safe replacement for a GetEligibleSpawnPoints + UpdateCooldowns
// pair: concurrent callers can never reserve the same point twice, so a burst
// of spawn passes for one field cannot exceed the spawn-point count.
func (r *SpawnPointRegistry) ReserveEligibleSpawnPoints(ctx context.Context, mapKey character.MapKey, limit int, defaultCooldown time.Duration, seed int64) ([]*CooldownSpawnPoint, error) {
	if limit <= 0 {
		return nil, nil
	}
	key := r.recurringKey(mapKey)
	nowMilli := time.Now().UnixMilli()

	result, err := reserveEligibleScript.Run(ctx, r.client, []string{key},
		nowMilli, limit, defaultCooldown.Milliseconds(), seed).Result()
	if err != nil {
		return nil, err
	}

	arr, ok := result.([]interface{})
	if !ok || len(arr) == 0 {
		return nil, nil
	}

	var reserved []*CooldownSpawnPoint
	for i := 1; i+1 < len(arr); i += 2 {
		valueStr, ok := arr[i+1].(string)
		if !ok {
			continue
		}
		var stored storedSpawnPoint
		if err := json.Unmarshal([]byte(valueStr), &stored); err != nil {
			continue
		}
		reserved = append(reserved, fromStored(stored))
	}

	return reserved, nil
}

// ResetCooldown resets the cooldown for all spawn points matching the given template ID with MobTime > 0.
// This is called when a boss monster is killed to enforce the full MobTime delay from the kill time.
func (r *SpawnPointRegistry) ResetCooldown(ctx context.Context, mapKey character.MapKey, templateId uint32) {
	key := r.recurringKey(mapKey)
	nowMilli := time.Now().UnixMilli()
	resetCooldownScript.Run(ctx, r.client, []string{key}, templateId, nowMilli)
}

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

// Reset clears all spawn point registries, across every tenant. Primarily
// used for testing.
func (r *SpawnPointRegistry) Reset(ctx context.Context) {
	_, _ = r.hashes.ClearAllAcrossTenants(ctx)
}

// GetSpawnPointsForMap returns the spawn points for a specific map key.
// Primarily used for testing and debugging.
func (r *SpawnPointRegistry) GetSpawnPointsForMap(ctx context.Context, mapKey character.MapKey) ([]*CooldownSpawnPoint, bool) {
	entries, err := r.hashes.GetAll(ctx, mapKey.Tenant, mapKey)
	if err != nil || len(entries) == 0 {
		return nil, false
	}

	var spawnPoints []*CooldownSpawnPoint
	for _, value := range entries {
		var stored storedSpawnPoint
		if err := json.Unmarshal([]byte(value), &stored); err != nil {
			continue
		}
		spawnPoints = append(spawnPoints, fromStored(stored))
	}

	return spawnPoints, true
}

// FlushTenant deletes every spawn-point hash for tenantId.
// Delegates to TenantKeyedHash.ClearForTenantId, which SCAN(COUNT=100) +
// pipelines DEL per batch against <prefix>:maps:spawn:<uuid>:* — TenantKey(t)
// starts with the bare tenant UUID, so this matches every (region, version)
// the tenant has ever been keyed under without FlushTenant needing to know
// either (the TenantDeleted-style caller here only carries the UUID).
func (r *SpawnPointRegistry) FlushTenant(ctx context.Context, l logrus.FieldLogger, tenantId uuid.UUID) (int, error) {
	deleted, err := r.hashes.ClearForTenantId(ctx, tenantId)
	if err != nil {
		l.WithError(err).Warnf("Spawn-registry flush failure for tenant [%s].", tenantId)
	}
	return deleted, err
}

// SetSpawnPointsForMap sets spawn points for a map key directly. Primarily used for testing.
func (r *SpawnPointRegistry) SetSpawnPointsForMap(ctx context.Context, mapKey character.MapKey, spawnPoints []*CooldownSpawnPoint) error {
	for _, csp := range spawnPoints {
		stored := toStored(csp.SpawnPoint, csp.NextSpawnAt)
		data, _ := json.Marshal(stored)
		if err := r.hashes.Set(ctx, mapKey.Tenant, mapKey, strconv.FormatUint(uint64(csp.Id), 10), string(data)); err != nil {
			return err
		}
	}
	return nil
}
