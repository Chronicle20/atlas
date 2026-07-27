// Package hidden tracks which characters are currently GM-hidden
// (SuperGmHide 9101004), shared across atlas-monsters replicas via Redis so
// any pod's controller election observes the same set (PRD FR-1.3).
package hidden

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	atlasredis "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// storedHidden carries full tenant identity alongside the character id so
// GetAll can rebuild tenant.Model for the reconciliation sweep — the same
// pattern as storedMonster in monster/registry.go.
type storedHidden struct {
	TenantId           string `json:"tenantId"`
	TenantRegion       string `json:"tenantRegion"`
	TenantMajorVersion uint16 `json:"tenantMajorVersion"`
	TenantMinorVersion uint16 `json:"tenantMinorVersion"`
	CharacterId        uint32 `json:"characterId"`
}

// Registry pairs a payload store with a per-tenant SET index (mirroring the
// monster registry's reg + mapIdx pairing):
//   - reg: atlas:hidden-character:<tenantId>:<characterId> -> storedHidden
//   - tenantIdx: atlas:hidden-characters:<tenantKey>:all -> SET of characterIds
type Registry struct {
	reg       *atlasredis.Registry[string, storedHidden]
	tenantIdx *atlasredis.TenantKeyedSet[string]
}

var (
	registry *Registry
	once     sync.Once
)

func newRegistry(rc *goredis.Client) *Registry {
	return &Registry{
		reg:       atlasredis.NewRegistry[string, storedHidden](rc, "hidden-character", func(s string) string { return s }),
		tenantIdx: atlasredis.NewTenantKeyedSet[string](rc, "hidden-characters", func(s string) string { return s }),
	}
}

func InitRegistry(rc *goredis.Client) {
	once.Do(func() {
		registry = newRegistry(rc)
	})
}

// GetRegistry returns the singleton, or nil before InitRegistry — callers
// must nil-check (same contract as GetPuppetRegistry).
func GetRegistry() *Registry {
	return registry
}

func payloadSuffix(t tenant.Model, characterId uint32) string {
	return fmt.Sprintf("%s:%d", t.Id().String(), characterId)
}

const tenantSetKey = "all"

// Add marks characterId as GM-hidden. Idempotent (SADD + Put overwrite).
func (r *Registry) Add(ctx context.Context, t tenant.Model, characterId uint32) error {
	if err := r.reg.Put(ctx, payloadSuffix(t, characterId), storedHidden{
		TenantId:           t.Id().String(),
		TenantRegion:       t.Region(),
		TenantMajorVersion: t.MajorVersion(),
		TenantMinorVersion: t.MinorVersion(),
		CharacterId:        characterId,
	}); err != nil {
		return err
	}
	return r.tenantIdx.Add(ctx, t, tenantSetKey, strconv.FormatUint(uint64(characterId), 10))
}

// Remove clears characterId's hidden mark. Idempotent (SREM + Remove of a
// missing key are both no-ops).
func (r *Registry) Remove(ctx context.Context, t tenant.Model, characterId uint32) error {
	if err := r.reg.Remove(ctx, payloadSuffix(t, characterId)); err != nil {
		return err
	}
	return r.tenantIdx.Remove(ctx, t, tenantSetKey, strconv.FormatUint(uint64(characterId), 10))
}

// MemberSet returns the hidden character ids for one tenant, fetched once
// per election (FR-4.1).
func (r *Registry) MemberSet(ctx context.Context, t tenant.Model) (map[uint32]struct{}, error) {
	members, err := r.tenantIdx.Members(ctx, t, tenantSetKey)
	if err != nil {
		return nil, err
	}
	out := make(map[uint32]struct{}, len(members))
	for _, m := range members {
		id, perr := strconv.ParseUint(m, 10, 32)
		if perr != nil {
			continue
		}
		out[uint32(id)] = struct{}{}
	}
	return out, nil
}

// GetAll returns every hidden character grouped by tenant — the
// reconciliation sweep's iteration source (mirrors Registry.GetMonsters).
func (r *Registry) GetAll(ctx context.Context) map[tenant.Model][]uint32 {
	result := make(map[tenant.Model][]uint32)
	all, err := r.reg.GetAll(ctx)
	if err != nil {
		return result
	}
	for _, sh := range all {
		tid, perr := uuid.Parse(sh.TenantId)
		if perr != nil {
			continue
		}
		t, terr := tenant.Create(tid, sh.TenantRegion, sh.TenantMajorVersion, sh.TenantMinorVersion)
		if terr != nil {
			continue
		}
		result[t] = append(result[t], sh.CharacterId)
	}
	return result
}

// Clear removes all hidden-character state (tests / operational reset).
// Deleting this state fail-opens to pre-task behavior (design D4).
//
// GetAll is read BEFORE reg.Clear wipes payloads, since the tenant index
// loop below needs to know which tenants to clear the SET index for.
func (r *Registry) Clear(ctx context.Context) {
	all := r.GetAll(ctx)
	_, _ = r.reg.Clear(ctx)
	for t := range all {
		_ = r.tenantIdx.Clear(ctx, t, tenantSetKey)
	}
}
