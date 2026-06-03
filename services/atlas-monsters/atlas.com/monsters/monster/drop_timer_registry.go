package monster

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	atlasredis "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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

// dropTimerSuffix returns the registry key suffix for a drop timer entry:
// "<tenantId>:<uniqueId>" — identical to the tail of the old dropTimerKey.
func dropTimerSuffix(t tenant.Model, uniqueId uint32) string {
	return fmt.Sprintf("%s:%d", t.Id().String(), uniqueId)
}

type DropTimerRegistry struct {
	reg *atlasredis.Registry[string, storedDropTimer]
}

var dropTimerRegistry *DropTimerRegistry
var dropTimerOnce sync.Once

func InitDropTimerRegistry(rc *goredis.Client) {
	dropTimerOnce.Do(func() {
		reg := atlasredis.NewRegistry[string, storedDropTimer](rc, "drop-timer", func(s string) string { return s })
		dropTimerRegistry = &DropTimerRegistry{reg: reg}
	})
}

func GetDropTimerRegistry() *DropTimerRegistry {
	return dropTimerRegistry
}

func (r *DropTimerRegistry) Register(ctx context.Context, t tenant.Model, uniqueId uint32, e DropTimerEntry) {
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
	_ = r.reg.Put(ctx, dropTimerSuffix(t, uniqueId), sd)
}

func (r *DropTimerRegistry) Unregister(ctx context.Context, t tenant.Model, uniqueId uint32) {
	_ = r.reg.Remove(ctx, dropTimerSuffix(t, uniqueId))
}

func (r *DropTimerRegistry) RecordHit(ctx context.Context, t tenant.Model, uniqueId uint32, hitTime time.Time) {
	_, _ = r.reg.Update(ctx, dropTimerSuffix(t, uniqueId), func(sd storedDropTimer) storedDropTimer {
		sd.LastHitAtMs = hitTime.UnixMilli()
		return sd
	})
}

func (r *DropTimerRegistry) UpdateLastDrop(ctx context.Context, t tenant.Model, uniqueId uint32, dropTime time.Time) {
	_, _ = r.reg.Update(ctx, dropTimerSuffix(t, uniqueId), func(sd storedDropTimer) storedDropTimer {
		sd.LastDropAtMs = dropTime.UnixMilli()
		return sd
	})
}

func (r *DropTimerRegistry) GetAll(ctx context.Context) map[MonsterKey]DropTimerEntry {
	result := make(map[MonsterKey]DropTimerEntry)
	items, err := r.reg.GetAll(ctx)
	if err != nil {
		return result
	}
	for _, sd := range items {
		t, entry := fromStoredDropTimer(sd)
		mk := MonsterKey{Tenant: t, MonsterId: sd.UniqueId}
		result[mk] = entry
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
