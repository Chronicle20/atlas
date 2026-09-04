package monster

import (
	"atlas-maps/map/character"
	"fmt"
	"sync"
	"time"

	monster2 "atlas-maps/data/map/monster"

	goredis "github.com/redis/go-redis/v9"

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
