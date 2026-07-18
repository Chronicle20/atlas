package game

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	goredis "github.com/redis/go-redis/v9"

	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// defaultTTL is the inactivity window before an RPS session is considered abandoned.
const defaultTTL = 5 * time.Minute

// Registry is a Redis-backed TTL session registry for active RPS games,
// keyed by (tenant, characterId), with per-tenant tracking so a sweeper
// can fan out over tenants to reclaim abandoned sessions.
type Registry struct {
	reg     *atlas.TTLRegistry[uint32, Model]
	tenants *atlas.Set
}

var registry *Registry

// InitRegistry initializes the package-level Registry singleton with the given Redis client.
func InitRegistry(client *goredis.Client) {
	registry = &Registry{
		reg: atlas.NewTTLRegistry[uint32, Model](client, "rps", func(k uint32) string {
			return strconv.FormatUint(uint64(k), 10)
		}, defaultTTL),
		tenants: atlas.NewSet(client, "rps:_tenants"),
	}
}

// GetRegistry returns the package-level Registry singleton.
func GetRegistry() *Registry {
	return registry
}

// Put stores the given Model, keyed by its tenant and character id, refreshing its TTL.
func (r *Registry) Put(ctx context.Context, m Model) {
	t := tenant.MustFromContext(ctx)
	_ = r.reg.Put(ctx, t, m.CharacterId(), m)
	r.trackTenant(ctx, t)
}

// Get retrieves the active game for the given character id in the requesting tenant.
func (r *Registry) Get(ctx context.Context, characterId uint32) (Model, bool) {
	t := tenant.MustFromContext(ctx)
	v, err := r.reg.Get(ctx, t, characterId)
	if err != nil {
		return Model{}, false
	}
	return v, true
}

// Remove deletes the active game for the given character id in the requesting tenant.
func (r *Registry) Remove(ctx context.Context, characterId uint32) {
	t := tenant.MustFromContext(ctx)
	_ = r.reg.Remove(ctx, t, characterId)
}

// PopExpired returns and removes all expired sessions across every tracked tenant.
func (r *Registry) PopExpired(ctx context.Context) []Model {
	tenants := r.getTrackedTenants(ctx)
	var results []Model
	for _, t := range tenants {
		expired, err := r.reg.PopExpired(ctx, t)
		if err != nil {
			continue
		}
		results = append(results, expired...)
	}
	return results
}

func (r *Registry) trackTenant(ctx context.Context, t tenant.Model) {
	data, err := json.Marshal(&t)
	if err != nil {
		return
	}
	_ = r.tenants.Add(ctx, string(data))
}

func (r *Registry) getTrackedTenants(ctx context.Context) []tenant.Model {
	members, err := r.tenants.Members(ctx)
	if err != nil {
		return nil
	}
	var tenants []tenant.Model
	for _, m := range members {
		var t tenant.Model
		if err := json.Unmarshal([]byte(m), &t); err != nil {
			continue
		}
		tenants = append(tenants, t)
	}
	return tenants
}

// SetNowFunc overrides the clock function for testing.
func (r *Registry) SetNowFunc(fn func() time.Time) {
	r.reg.SetNowFunc(fn)
}
