package expression

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/Chronicle20/atlas-constants/field"
	atlas "github.com/Chronicle20/atlas-redis"
	"github.com/Chronicle20/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

const defaultTTL = 5 * time.Second

type Registry struct {
	reg       *atlas.TTLRegistry[uint32, Model]
	client    *goredis.Client
	tenantKey string
}

var registry *Registry

func InitRegistry(client *goredis.Client) {
	registry = &Registry{
		reg: atlas.NewTTLRegistry[uint32, Model](client, "expression", func(k uint32) string {
			return strconv.FormatUint(uint64(k), 10)
		}, defaultTTL),
		client:    client,
		tenantKey: "atlas:expression:_tenants",
	}
}

func GetRegistry() *Registry {
	return registry
}

func (r *Registry) add(ctx context.Context, characterId uint32, f field.Model, expression uint32) Model {
	t := tenant.MustFromContext(ctx)
	expiration := time.Now().Add(defaultTTL)

	e := NewModelBuilder(t).
		SetCharacterId(characterId).
		SetLocation(f).
		SetExpression(expression).
		SetExpiration(expiration).
		MustBuild()

	_ = r.reg.Put(ctx, t, characterId, e)
	r.trackTenant(ctx, t)
	return e
}

func (r *Registry) get(ctx context.Context, characterId uint32) (Model, bool) {
	t := tenant.MustFromContext(ctx)
	v, err := r.reg.Get(ctx, t, characterId)
	if err != nil {
		return Model{}, false
	}
	return v, true
}

func (r *Registry) clear(ctx context.Context, characterId uint32) {
	t := tenant.MustFromContext(ctx)
	_ = r.reg.Remove(ctx, t, characterId)
}

func (r *Registry) popExpired(ctx context.Context) []Model {
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
	r.client.SAdd(ctx, r.tenantKey, string(data))
}

func (r *Registry) getTrackedTenants(ctx context.Context) []tenant.Model {
	members, err := r.client.SMembers(ctx, r.tenantKey).Result()
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
