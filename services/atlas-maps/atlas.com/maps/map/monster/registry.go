package monster

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	monster2 "atlas-maps/data/map/monster"
	"atlas-maps/map/character"

	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
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
type SpawnPointRegistry struct {
	client *goredis.Client
}

var (
	registryInstance *SpawnPointRegistry
	registryOnce     sync.Once
)

// InitRegistry initializes the singleton SpawnPointRegistry with a Redis client.
func InitRegistry(rc *goredis.Client) {
	registryOnce.Do(func() {
		registryInstance = &SpawnPointRegistry{client: rc}
	})
}

// GetRegistry returns the singleton SpawnPointRegistry instance.
func GetRegistry() *SpawnPointRegistry {
	return registryInstance
}

func spawnHashKey(mapKey character.MapKey) string {
	return fmt.Sprintf("atlas:maps:spawn:%s:%d:%d:%d:%s",
		mapKey.Tenant.String(),
		mapKey.Field.WorldId(),
		mapKey.Field.ChannelId(),
		mapKey.Field.MapId(),
		mapKey.Field.Instance().String(),
	)
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

// eligibleScript returns total spawn point count and eligible entries (NextSpawnAt <= now).
// Returns: [totalCount, field1, value1, field2, value2, ...]
var eligibleScript = goredis.NewScript(`
local entries = redis.call('HGETALL', KEYS[1])
local now = tonumber(ARGV[1])
local total = math.floor(#entries / 2)
local result = {tostring(total)}
for i = 1, #entries, 2 do
    local field = entries[i]
    local value = entries[i+1]
    local data = cjson.decode(value)
    if data.nextSpawnAt <= now then
        result[#result + 1] = field
        result[#result + 1] = value
    end
end
return result
`)

// updateCooldownsScript atomically updates NextSpawnAt for multiple spawn points.
// ARGV: pairs of (spawnPointId, newNextSpawnAtMilli)
var updateCooldownsScript = goredis.NewScript(`
for i = 1, #ARGV, 2 do
    local field = ARGV[i]
    local newNextSpawnAt = tonumber(ARGV[i+1])
    local value = redis.call('HGET', KEYS[1], field)
    if value then
        local data = cjson.decode(value)
        data.nextSpawnAt = newNextSpawnAt
        redis.call('HSET', KEYS[1], field, cjson.encode(data))
    end
end
return 1
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

	exists, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return err
	}
	if exists > 0 {
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

// GetEligibleSpawnPoints returns eligible spawn points and total count via Lua script.
// A spawn point is eligible if its NextSpawnAt <= now.
func (r *SpawnPointRegistry) GetEligibleSpawnPoints(ctx context.Context, mapKey character.MapKey) ([]*CooldownSpawnPoint, int, error) {
	key := spawnHashKey(mapKey)
	nowMilli := time.Now().UnixMilli()

	result, err := eligibleScript.Run(ctx, r.client, []string{key}, nowMilli).Result()
	if err != nil {
		return nil, 0, err
	}

	arr, ok := result.([]interface{})
	if !ok || len(arr) == 0 {
		return nil, 0, nil
	}

	totalStr, ok := arr[0].(string)
	if !ok {
		return nil, 0, fmt.Errorf("unexpected total count type")
	}
	totalCount, err := strconv.Atoi(totalStr)
	if err != nil {
		return nil, 0, err
	}

	var eligible []*CooldownSpawnPoint
	for i := 1; i+1 < len(arr); i += 2 {
		valueStr, ok := arr[i+1].(string)
		if !ok {
			continue
		}
		var stored storedSpawnPoint
		if err := json.Unmarshal([]byte(valueStr), &stored); err != nil {
			continue
		}
		eligible = append(eligible, fromStored(stored))
	}

	return eligible, totalCount, nil
}

// UpdateCooldowns atomically updates NextSpawnAt for multiple spawn points.
func (r *SpawnPointRegistry) UpdateCooldowns(ctx context.Context, mapKey character.MapKey, updates map[uint32]time.Time) error {
	if len(updates) == 0 {
		return nil
	}
	key := spawnHashKey(mapKey)
	args := make([]interface{}, 0, len(updates)*2)
	for spId, nextSpawnAt := range updates {
		args = append(args, strconv.FormatUint(uint64(spId), 10), nextSpawnAt.UnixMilli())
	}
	_, err := updateCooldownsScript.Run(ctx, r.client, []string{key}, args...).Result()
	return err
}

// ResetCooldown resets the cooldown for all spawn points matching the given template ID with MobTime > 0.
// This is called when a boss monster is killed to enforce the full MobTime delay from the kill time.
func (r *SpawnPointRegistry) ResetCooldown(ctx context.Context, mapKey character.MapKey, templateId uint32) {
	key := spawnHashKey(mapKey)
	nowMilli := time.Now().UnixMilli()
	resetCooldownScript.Run(ctx, r.client, []string{key}, templateId, nowMilli)
}

// Reset clears all spawn point registries. Primarily used for testing.
func (r *SpawnPointRegistry) Reset(ctx context.Context) {
	iter := r.client.Scan(ctx, 0, "atlas:maps:spawn:*", 0).Iterator()
	for iter.Next(ctx) {
		r.client.Del(ctx, iter.Val())
	}
}

// GetSpawnPointsForMap returns the spawn points for a specific map key.
// Primarily used for testing and debugging.
func (r *SpawnPointRegistry) GetSpawnPointsForMap(ctx context.Context, mapKey character.MapKey) ([]*CooldownSpawnPoint, bool) {
	key := spawnHashKey(mapKey)
	entries, err := r.client.HGetAll(ctx, key).Result()
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

// SetSpawnPointsForMap sets spawn points for a map key directly. Primarily used for testing.
func (r *SpawnPointRegistry) SetSpawnPointsForMap(ctx context.Context, mapKey character.MapKey, spawnPoints []*CooldownSpawnPoint) error {
	key := spawnHashKey(mapKey)
	pipe := r.client.Pipeline()
	for _, csp := range spawnPoints {
		stored := toStored(csp.SpawnPoint, csp.NextSpawnAt)
		data, _ := json.Marshal(stored)
		pipe.HSet(ctx, key, strconv.FormatUint(uint64(csp.SpawnPoint.Id), 10), string(data))
	}
	_, err := pipe.Exec(ctx)
	return err
}
