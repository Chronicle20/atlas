package monster

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	atlasredis "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// SelfDestructTimerEntry is an armed timer-driven detonation (task-253 design
// D6, FR-3.1): a mob whose selfDestruction block carries no HP predicate
// detonates at FireAt with the animation named by Action.
type SelfDestructTimerEntry struct {
	monsterId uint32
	field     field.Model
	action    byte
	fireAt    time.Time
}

func NewSelfDestructTimerEntry(monsterId uint32, f field.Model, action byte, fireAt time.Time) SelfDestructTimerEntry {
	return SelfDestructTimerEntry{monsterId: monsterId, field: f, action: action, fireAt: fireAt}
}

func (e SelfDestructTimerEntry) MonsterId() uint32  { return e.monsterId }
func (e SelfDestructTimerEntry) Field() field.Model { return e.field }
func (e SelfDestructTimerEntry) Action() byte       { return e.action }
func (e SelfDestructTimerEntry) FireAt() time.Time  { return e.fireAt }

type storedSelfDestructTimer struct {
	TenantId           string      `json:"tenantId"`
	TenantRegion       string      `json:"tenantRegion"`
	TenantMajorVersion uint16      `json:"tenantMajorVersion"`
	TenantMinorVersion uint16      `json:"tenantMinorVersion"`
	UniqueId           uint32      `json:"uniqueId"`
	MonsterId          uint32      `json:"monsterId"`
	Field              field.Model `json:"field"`
	Action             byte        `json:"action"`
	FireAtMs           int64       `json:"fireAtMs"`
}

// SelfDestructTimerRegistry is tenant-scoped: the stored key is
// atlas:self-destruct-timer:<tenantId>:<region>:<major>.<minor>:<uniqueId>.
// GetAll is the one genuine cross-tenant operation — the periodic sweep has no
// tenant to loop over — and uses the explicit GetAllAcrossTenants sibling, the
// same shape DropTimerRegistry uses.
type SelfDestructTimerRegistry struct {
	reg *atlasredis.TenantRegistry[uint32, storedSelfDestructTimer]
}

var (
	selfDestructTimerRegistry *SelfDestructTimerRegistry
	selfDestructTimerOnce     sync.Once
)

func InitSelfDestructTimerRegistry(rc *goredis.Client) {
	selfDestructTimerOnce.Do(func() {
		reg := atlasredis.NewTenantRegistry[uint32, storedSelfDestructTimer](rc, "self-destruct-timer", func(id uint32) string { return strconv.FormatUint(uint64(id), 10) })
		selfDestructTimerRegistry = &SelfDestructTimerRegistry{reg: reg}
	})
}

func GetSelfDestructTimerRegistry() *SelfDestructTimerRegistry {
	return selfDestructTimerRegistry
}

func (r *SelfDestructTimerRegistry) Register(ctx context.Context, t tenant.Model, uniqueId uint32, e SelfDestructTimerEntry) {
	_ = r.reg.Put(ctx, t, uniqueId, storedSelfDestructTimer{
		TenantId:           t.Id().String(),
		TenantRegion:       t.Region(),
		TenantMajorVersion: t.MajorVersion(),
		TenantMinorVersion: t.MinorVersion(),
		UniqueId:           uniqueId,
		MonsterId:          e.monsterId,
		Field:              e.field,
		Action:             e.action,
		FireAtMs:           e.fireAt.UnixMilli(),
	})
}

func (r *SelfDestructTimerRegistry) Unregister(ctx context.Context, t tenant.Model, uniqueId uint32) {
	_ = r.reg.Remove(ctx, t, uniqueId)
}

func (r *SelfDestructTimerRegistry) GetAll(ctx context.Context) map[MonsterKey]SelfDestructTimerEntry {
	result := make(map[MonsterKey]SelfDestructTimerEntry)
	items, err := r.reg.GetAllAcrossTenants(ctx)
	if err != nil {
		return result
	}
	for _, sd := range items {
		t, entry := fromStoredSelfDestructTimer(sd)
		result[MonsterKey{Tenant: t, MonsterId: sd.UniqueId}] = entry
	}
	return result
}

func fromStoredSelfDestructTimer(sd storedSelfDestructTimer) (tenant.Model, SelfDestructTimerEntry) {
	tid, _ := uuid.Parse(sd.TenantId)
	t, _ := tenant.Create(tid, sd.TenantRegion, sd.TenantMajorVersion, sd.TenantMinorVersion)
	return t, SelfDestructTimerEntry{
		monsterId: sd.MonsterId,
		field:     sd.Field,
		action:    sd.Action,
		fireAt:    time.UnixMilli(sd.FireAtMs),
	}
}
