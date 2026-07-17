package broadcast

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

// Registry provides tenant-scoped, CAS-serialized access to per
// (tenant, world, family) broadcast QueueModel state in Redis.
type Registry struct {
	queues  *atlas.TenantRegistry[string, QueueModel]
	tenants *atlas.Set
}

var broadcastRegistry *Registry

func queueKey(worldId world.Id, family string) string {
	return fmt.Sprintf("%d:%s", worldId, family)
}

func InitRegistry(client *goredis.Client) {
	broadcastRegistry = &Registry{
		queues:  atlas.NewTenantRegistry[string, QueueModel](client, "world-broadcast", func(k string) string { return k }),
		tenants: atlas.NewSet(client, "world-broadcast:tenants"),
	}
}

func GetRegistry() *Registry {
	return broadcastRegistry
}

// Get returns the current QueueModel for (tenant, worldId, family). If no
// queue has been created yet, it returns atlas.ErrNotFound (via the
// underlying TenantRegistry.Get).
func (r *Registry) Get(ctx context.Context, t tenant.Model, worldId world.Id, family string) (QueueModel, error) {
	key := queueKey(worldId, family)
	return r.queues.Get(ctx, t, key)
}

// Upsert applies fn to the current QueueModel for (tenant, worldId, family)
// under optimistic CAS (WATCH/MULTI/EXEC via TenantRegistry.Update), creating
// an empty QueueModel first if none exists yet. Concurrent create-on-missing
// Puts are idempotent (both write an empty QueueModel{}). WATCH/EXEC does not
// serialize the mutation itself: it detects when another writer changed the
// key between WATCH and EXEC, and the loser retries by re-reading the
// current value and re-applying fn (bounded; see TenantRegistry.Update).
// Because of this, fn may run more than once per Upsert call and must be
// side-effect free / pure in its observable effects. Also tracks t in the
// tenant set for Tenants().
func (r *Registry) Upsert(ctx context.Context, t tenant.Model, worldId world.Id, family string, fn func(QueueModel) QueueModel) (QueueModel, error) {
	key := queueKey(worldId, family)

	result, err := r.queues.Update(ctx, t, key, fn)
	if errors.Is(err, atlas.ErrNotFound) {
		if putErr := r.queues.Put(ctx, t, key, QueueModel{}); putErr != nil {
			var zero QueueModel
			return zero, putErr
		}
		result, err = r.queues.Update(ctx, t, key, fn)
	}
	if err != nil {
		var zero QueueModel
		return zero, err
	}

	r.trackTenant(ctx, t)
	return result, nil
}

func (r *Registry) Tenants() []tenant.Model {
	ctx := context.Background()
	members, err := r.tenants.Members(ctx)
	if err != nil {
		return nil
	}
	results := make([]tenant.Model, 0)
	for _, data := range members {
		var t tenant.Model
		if err := json.Unmarshal([]byte(data), &t); err != nil {
			continue
		}
		results = append(results, t)
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
