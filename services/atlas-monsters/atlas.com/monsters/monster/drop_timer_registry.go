package monster

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/Chronicle20/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

type DropTimerEntry struct {
	monsterId    uint32
	field        field.Model
	dropPeriod   time.Duration
	weaponAttack uint32
	maxHp        uint32
	lastDropAt   time.Time
	lastHitAt    time.Time
}

func (e DropTimerEntry) MonsterId() uint32        { return e.monsterId }
func (e DropTimerEntry) Field() field.Model        { return e.field }
func (e DropTimerEntry) DropPeriod() time.Duration { return e.dropPeriod }
func (e DropTimerEntry) WeaponAttack() uint32      { return e.weaponAttack }
func (e DropTimerEntry) MaxHp() uint32             { return e.maxHp }
func (e DropTimerEntry) LastDropAt() time.Time     { return e.lastDropAt }
func (e DropTimerEntry) LastHitAt() time.Time      { return e.lastHitAt }

type storedDropTimer struct {
	TenantId           string      `json:"tenantId"`
	TenantRegion       string      `json:"tenantRegion"`
	TenantMajorVersion uint16      `json:"tenantMajorVersion"`
	TenantMinorVersion uint16      `json:"tenantMinorVersion"`
	UniqueId           uint32      `json:"uniqueId"`
	MonsterId          uint32      `json:"monsterId"`
	Field              field.Model `json:"field"`
	DropPeriodMs       int64       `json:"dropPeriodMs"`
	WeaponAttack       uint32      `json:"weaponAttack"`
	MaxHp              uint32      `json:"maxHp"`
	LastDropAtMs       int64       `json:"lastDropAtMs"`
	LastHitAtMs        int64       `json:"lastHitAtMs"`
}

type DropTimerRegistry struct {
	client *goredis.Client
}

var dropTimerRegistry *DropTimerRegistry
var dropTimerOnce sync.Once

func InitDropTimerRegistry(rc *goredis.Client) {
	dropTimerOnce.Do(func() {
		dropTimerRegistry = &DropTimerRegistry{client: rc}
	})
}

func GetDropTimerRegistry() *DropTimerRegistry {
	return dropTimerRegistry
}

func dropTimerKey(t tenant.Model, uniqueId uint32) string {
	return fmt.Sprintf("atlas:drop-timer:%s:%d", t.Id().String(), uniqueId)
}

func (r *DropTimerRegistry) Register(ctx context.Context, t tenant.Model, uniqueId uint32, e DropTimerEntry) {
	key := dropTimerKey(t, uniqueId)
	sd := storedDropTimer{
		TenantId:           t.Id().String(),
		TenantRegion:       t.Region(),
		TenantMajorVersion: t.MajorVersion(),
		TenantMinorVersion: t.MinorVersion(),
		UniqueId:           uniqueId,
		MonsterId:          e.monsterId,
		Field:              e.field,
		DropPeriodMs:       e.dropPeriod.Milliseconds(),
		WeaponAttack:       e.weaponAttack,
		MaxHp:              e.maxHp,
		LastDropAtMs:       e.lastDropAt.UnixMilli(),
		LastHitAtMs:        timeToMillis(e.lastHitAt),
	}
	data, err := json.Marshal(sd)
	if err != nil {
		return
	}
	r.client.Set(ctx, key, data, 0)
}

func (r *DropTimerRegistry) Unregister(ctx context.Context, t tenant.Model, uniqueId uint32) {
	key := dropTimerKey(t, uniqueId)
	r.client.Del(ctx, key)
}

func (r *DropTimerRegistry) RecordHit(ctx context.Context, t tenant.Model, uniqueId uint32, hitTime time.Time) {
	key := dropTimerKey(t, uniqueId)
	err := r.client.Watch(ctx, func(tx *goredis.Tx) error {
		val, err := tx.Get(ctx, key).Result()
		if err != nil {
			return err
		}
		var sd storedDropTimer
		if err := json.Unmarshal([]byte(val), &sd); err != nil {
			return err
		}
		sd.LastHitAtMs = hitTime.UnixMilli()
		data, err := json.Marshal(sd)
		if err != nil {
			return err
		}
		_, err = tx.TxPipelined(ctx, func(pipe goredis.Pipeliner) error {
			pipe.Set(ctx, key, data, 0)
			return nil
		})
		return err
	}, key)
	if err != nil {
		return
	}
}

func (r *DropTimerRegistry) UpdateLastDrop(ctx context.Context, t tenant.Model, uniqueId uint32, dropTime time.Time) {
	key := dropTimerKey(t, uniqueId)
	err := r.client.Watch(ctx, func(tx *goredis.Tx) error {
		val, err := tx.Get(ctx, key).Result()
		if err != nil {
			return err
		}
		var sd storedDropTimer
		if err := json.Unmarshal([]byte(val), &sd); err != nil {
			return err
		}
		sd.LastDropAtMs = dropTime.UnixMilli()
		data, err := json.Marshal(sd)
		if err != nil {
			return err
		}
		_, err = tx.TxPipelined(ctx, func(pipe goredis.Pipeliner) error {
			pipe.Set(ctx, key, data, 0)
			return nil
		})
		return err
	}, key)
	if err != nil {
		return
	}
}

func (r *DropTimerRegistry) GetAll(ctx context.Context) map[MonsterKey]DropTimerEntry {
	result := make(map[MonsterKey]DropTimerEntry)
	var cursor uint64
	for {
		keys, nextCursor, err := r.client.Scan(ctx, cursor, "atlas:drop-timer:*", 100).Result()
		if err != nil {
			return result
		}
		if len(keys) > 0 {
			pipe := r.client.Pipeline()
			cmds := make([]*goredis.StringCmd, len(keys))
			for i, k := range keys {
				cmds[i] = pipe.Get(ctx, k)
			}
			_, err := pipe.Exec(ctx)
			if err != nil && err != goredis.Nil {
				cursor = nextCursor
				if cursor == 0 {
					break
				}
				continue
			}
			for _, cmd := range cmds {
				val, err := cmd.Result()
				if err != nil {
					continue
				}
				var sd storedDropTimer
				if err := json.Unmarshal([]byte(val), &sd); err != nil {
					continue
				}
				t, entry := fromStoredDropTimer(sd)
				mk := MonsterKey{Tenant: t, MonsterId: sd.UniqueId}
				result[mk] = entry
			}
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return result
}

func fromStoredDropTimer(sd storedDropTimer) (tenant.Model, DropTimerEntry) {
	tid, _ := uuid.Parse(sd.TenantId)
	t, _ := tenant.Create(tid, sd.TenantRegion, sd.TenantMajorVersion, sd.TenantMinorVersion)
	var lastHitAt time.Time
	if sd.LastHitAtMs != 0 {
		lastHitAt = time.UnixMilli(sd.LastHitAtMs)
	}
	return t, DropTimerEntry{
		monsterId:    sd.MonsterId,
		field:        sd.Field,
		dropPeriod:   time.Duration(sd.DropPeriodMs) * time.Millisecond,
		weaponAttack: sd.WeaponAttack,
		maxHp:        sd.MaxHp,
		lastDropAt:   time.UnixMilli(sd.LastDropAtMs),
		lastHitAt:    lastHitAt,
	}
}

func timeToMillis(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixMilli()
}
