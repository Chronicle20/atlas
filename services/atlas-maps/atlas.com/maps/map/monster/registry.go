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

// SpawnPointRegistry manages spawn point cooldowns backed by Redis hashes.
// Each map's spawn points are stored as a Redis hash keyed by MapKey.
// Hash field: spawn point ID (string)
// Hash value: JSON-encoded storedSpawnPoint with NextSpawnAt as Unix milliseconds
// SpawnPointRegistry manages spawn point cooldowns backed by Redis hashes,
// scoped per tenant via TenantKeyedHash (D7): the rendered key embeds
// TenantKey(mk.Tenant), which starts with the bare tenant UUID.
type SpawnPointRegistry struct {
	client *goredis.Client
	hashes *atlasredis.TenantKeyedHash[character.MapKey]
}

var (
	registryInstance *SpawnPointRegistry
	registryOnce     sync.Once
)

// InitRegistry initializes the singleton SpawnPointRegistry with a Redis client.
func InitRegistry(rc *goredis.Client) {
	registryOnce.Do(func() {
		registryInstance = &SpawnPointRegistry{
			client: rc,
			hashes: atlasredis.NewTenantKeyedHash[character.MapKey](rc, "maps:spawn", func(mk character.MapKey) string {
				// Tenant scoping is applied by TenantKeyedHash itself (mk.Tenant is
				// passed explicitly on every call below); the key fn only encodes
				// the field-scoped portion.
				return fmt.Sprintf("%d:%d:%d:%s",
					mk.Field.WorldId(),
					mk.Field.ChannelId(),
					mk.Field.MapId(),
					mk.Field.Instance().String(),
				)
			}),
		}
	})
}

// GetRegistry returns the singleton SpawnPointRegistry instance.
func GetRegistry() *SpawnPointRegistry {
	return registryInstance
}

func spawnHashKey(mapKey character.MapKey) string {
	return registryInstance.hashes.Key(mapKey.Tenant, mapKey)
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

// initializeScript atomically initializes spawn points for a map if not already present.
var initializeScript = goredis.NewScript(`
if redis.call('EXISTS', KEYS[1]) == 1 then
    return 0
end
for i = 1, #ARGV, 2 do
    redis.call('HSET', KEYS[1], ARGV[i], ARGV[i+1])
end
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

// InitializeForMap initializes spawn points for a map if not already present in Redis.
// Uses a Lua script for atomic check-and-initialize to prevent duplicate initialization.
func (r *SpawnPointRegistry) InitializeForMap(ctx context.Context, mapKey character.MapKey, dp monster2.Processor, l logrus.FieldLogger) error {
	key := spawnHashKey(mapKey)

	n, err := r.hashes.Len(ctx, mapKey.Tenant, mapKey)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}

	spawnPoints, err := dp.GetSpawnableSpawnPoints(mapKey.Field.MapId())
	if err != nil {
		return err
	}

	if len(spawnPoints) == 0 {
		return nil
	}

	now := time.Now()
	args := make([]interface{}, 0, len(spawnPoints)*2)
	for _, sp := range spawnPoints {
		stored := toStored(sp, now)
		data, err := json.Marshal(stored)
		if err != nil {
			return err
		}
		args = append(args, strconv.FormatUint(uint64(sp.Id), 10), string(data))
	}

	_, err = initializeScript.Run(ctx, r.client, []string{key}, args...).Result()
	if err != nil {
		return err
	}

	l.Debugf("Initialized spawn point registry for map key: Tenant [%s] World [%d] Channel [%d] Map [%d] with %d spawn points",
		mapKey.Tenant.String(), mapKey.Field.WorldId(), mapKey.Field.ChannelId(), mapKey.Field.MapId(), len(spawnPoints))

	return nil
}

// Count returns the number of spawn points registered for a map. The spawn
// point set is fixed after initialization, so this count is stable and is not
// subject to the spawn-time eligibility race.
func (r *SpawnPointRegistry) Count(ctx context.Context, mapKey character.MapKey) (int, error) {
	n, err := r.hashes.Len(ctx, mapKey.Tenant, mapKey)
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
	key := spawnHashKey(mapKey)
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
	key := spawnHashKey(mapKey)
	nowMilli := time.Now().UnixMilli()
	resetCooldownScript.Run(ctx, r.client, []string{key}, templateId, nowMilli)
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
		if err := r.hashes.Set(ctx, mapKey.Tenant, mapKey, strconv.FormatUint(uint64(csp.SpawnPoint.Id), 10), string(data)); err != nil {
			return err
		}
	}
	return nil
}
