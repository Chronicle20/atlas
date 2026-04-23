package reactor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	objectid "github.com/Chronicle20/atlas/libs/atlas-object-id"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

const (
	allReactorsKey = "reactors:all"
)

type Registry struct {
	client    *goredis.Client
	allocator objectid.Allocator
}

var reg *Registry

func InitRegistry(client *goredis.Client) {
	reg = &Registry{client: client, allocator: objectid.NewRedisAllocator(client)}
}

func GetRegistry() *Registry {
	return reg
}

type MapKey struct {
	worldId   world.Id
	channelId channel.Id
	mapId     _map.Id
	instance  uuid.UUID
}

func NewMapKey(f field.Model) MapKey {
	return MapKey{
		worldId:   f.WorldId(),
		channelId: f.ChannelId(),
		mapId:     f.MapId(),
		instance:  f.Instance(),
	}
}

func reactorKey(t tenant.Model, id uint32) string {
	return fmt.Sprintf("reactor:%s:%d", t.Id().String(), id)
}

func reactorIdStr(id uint32) string {
	return fmt.Sprintf("%d", id)
}

// allSetMember encodes a tenant+id pair for the global reactors:all set.
func allSetMember(t tenant.Model, id uint32) string {
	return fmt.Sprintf("%s:%d", t.Id().String(), id)
}

func mapSetKey(t tenant.Model, mk MapKey) string {
	return fmt.Sprintf("reactors:map:%s:%d:%d:%d:%s", t.Id().String(), mk.worldId, mk.channelId, mk.mapId, mk.instance.String())
}

func cooldownKey(t tenant.Model, mk MapKey, classification uint32, x int16, y int16) string {
	return fmt.Sprintf("reactor:cd:%s:%d:%d:%d:%s:%d:%d:%d", t.Id().String(), mk.worldId, mk.channelId, mk.mapId, mk.instance.String(), classification, x, y)
}

func spotKey(t tenant.Model, mk MapKey, classification uint32, x int16, y int16) string {
	return fmt.Sprintf("reactor:spot:%s:%d:%d:%d:%s:%d:%d:%d", t.Id().String(), mk.worldId, mk.channelId, mk.mapId, mk.instance.String(), classification, x, y)
}

func spotPatternKey(t tenant.Model, mk MapKey) string {
	return fmt.Sprintf("reactor:spot:%s:%d:%d:%d:%s:*", t.Id().String(), mk.worldId, mk.channelId, mk.mapId, mk.instance.String())
}


func (r *Registry) store(t tenant.Model, id uint32, m Model) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return r.client.Set(context.Background(), reactorKey(t, id), data, 0).Err()
}

func (r *Registry) load(t tenant.Model, id uint32) (Model, bool) {
	data, err := r.client.Get(context.Background(), reactorKey(t, id)).Bytes()
	if err != nil {
		return Model{}, false
	}
	var m Model
	if err := json.Unmarshal(data, &m); err != nil {
		return Model{}, false
	}
	return m, true
}

func (r *Registry) Get(t tenant.Model, id uint32) (Model, error) {
	m, ok := r.load(t, id)
	if !ok {
		return Model{}, errors.New("unable to locate reactor")
	}
	return m, nil
}

func (r *Registry) GetAll() map[tenant.Model][]Model {
	members, err := r.client.SMembers(context.Background(), allReactorsKey).Result()
	if err != nil {
		return make(map[tenant.Model][]Model)
	}

	res := make(map[tenant.Model][]Model)
	for _, member := range members {
		// Members are stored as "{tenantId}:{id}". The legacy "{id}"-only form
		// is skipped; a rolling restart drops those.
		sep := -1
		for i := len(member) - 1; i >= 0; i-- {
			if member[i] == ':' {
				sep = i
				break
			}
		}
		if sep < 0 {
			continue
		}
		id, err := strconv.ParseUint(member[sep+1:], 10, 32)
		if err != nil {
			continue
		}
		tenantId, err := uuid.Parse(member[:sep])
		if err != nil {
			continue
		}
		te, err := tenant.Create(tenantId, "", 0, 0)
		if err != nil {
			continue
		}
		if m, ok := r.load(te, uint32(id)); ok {
			// Prefer the tenant stored on the model (it has region/version too).
			res[m.Tenant()] = append(res[m.Tenant()], m)
		}
	}
	return res
}

func (r *Registry) GetInField(t tenant.Model, f field.Model) []Model {
	mk := NewMapKey(f)
	key := mapSetKey(t, mk)

	members, err := r.client.SMembers(context.Background(), key).Result()
	if err != nil {
		return make([]Model, 0)
	}

	result := make([]Model, 0, len(members))
	for _, member := range members {
		id, err := strconv.ParseUint(member, 10, 32)
		if err != nil {
			continue
		}
		if m, ok := r.load(t, uint32(id)); ok {
			result = append(result, m)
		}
	}
	return result
}

func (r *Registry) Create(t tenant.Model, b *ModelBuilder) (Model, error) {
	ctx := context.Background()
	id, err := r.allocator.Allocate(ctx, t)
	if err != nil {
		return Model{}, fmt.Errorf("allocate reactor oid: %w", err)
	}
	m, err := b.SetId(id).UpdateTime().Build()
	if err != nil {
		_ = r.allocator.Release(ctx, t, id)
		return Model{}, err
	}

	if err := r.store(t, id, m); err != nil {
		_ = r.allocator.Release(ctx, t, id)
		return Model{}, err
	}

	mk := NewMapKey(m.Field())
	idStr := reactorIdStr(id)
	pipe := r.client.Pipeline()
	pipe.SAdd(ctx, allReactorsKey, allSetMember(t, id))
	pipe.SAdd(ctx, mapSetKey(t, mk), idStr)
	_, _ = pipe.Exec(ctx)

	return m, nil
}

func (r *Registry) Update(t tenant.Model, id uint32, modifier func(*ModelBuilder)) (Model, error) {
	m, ok := r.load(t, id)
	if !ok {
		return Model{}, errors.New("unable to locate reactor")
	}

	b := NewFromModel(m)
	modifier(b)
	b.UpdateTime()
	updated, err := b.Build()
	if err != nil {
		return Model{}, err
	}

	if err := r.store(t, id, updated); err != nil {
		return Model{}, err
	}
	return updated, nil
}

func (r *Registry) Remove(t tenant.Model, id uint32) {
	m, ok := r.load(t, id)
	if !ok {
		return
	}

	ctx := context.Background()
	mk := NewMapKey(m.Field())
	idStr := reactorIdStr(id)

	pipe := r.client.Pipeline()
	pipe.Del(ctx, reactorKey(t, id))
	pipe.SRem(ctx, allReactorsKey, allSetMember(t, id))
	pipe.SRem(ctx, mapSetKey(t, mk), idStr)
	_, _ = pipe.Exec(ctx)

	_ = r.allocator.Release(ctx, t, id)
}

func (r *Registry) RecordCooldown(t tenant.Model, mk MapKey, classification uint32, x int16, y int16, delay uint32) {
	if delay == 0 {
		return
	}
	key := cooldownKey(t, mk, classification, x, y)
	r.client.Set(context.Background(), key, "1", time.Millisecond*time.Duration(delay))
}

func (r *Registry) IsOnCooldown(t tenant.Model, mk MapKey, classification uint32, x int16, y int16) bool {
	key := cooldownKey(t, mk, classification, x, y)
	exists, err := r.client.Exists(context.Background(), key).Result()
	if err != nil {
		return false
	}
	return exists > 0
}

func (r *Registry) ClearCooldown(t tenant.Model, mk MapKey, classification uint32, x int16, y int16) {
	key := cooldownKey(t, mk, classification, x, y)
	r.client.Del(context.Background(), key)
}

func cooldownPatternKey(t tenant.Model, mk MapKey) string {
	return fmt.Sprintf("reactor:cd:%s:%d:%d:%d:%s:*", t.Id().String(), mk.worldId, mk.channelId, mk.mapId, mk.instance.String())
}

func (r *Registry) ClearAllCooldownsForMap(t tenant.Model, mk MapKey) {
	pattern := cooldownPatternKey(t, mk)
	var cursor uint64
	for {
		keys, next, err := r.client.Scan(context.Background(), cursor, pattern, 100).Result()
		if err != nil {
			return
		}
		if len(keys) > 0 {
			r.client.Del(context.Background(), keys...)
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
}

func (r *Registry) CleanupExpiredCooldowns() {
	// No-op: Redis TTL handles expiration automatically
}

// TryClaimSpot atomically reserves a (classification, x, y) slot within a map
// instance, so concurrent Create calls cannot produce two reactors stacked at
// the same position. Returns true if this caller owns the slot, false if it was
// already claimed. The caller is responsible for ReleaseSpot on failure or
// destruction.
func (r *Registry) TryClaimSpot(t tenant.Model, mk MapKey, classification uint32, x int16, y int16) bool {
	key := spotKey(t, mk, classification, x, y)
	ok, err := r.client.SetNX(context.Background(), key, "1", 0).Result()
	if err != nil {
		return false
	}
	return ok
}

func (r *Registry) ReleaseSpot(t tenant.Model, mk MapKey, classification uint32, x int16, y int16) {
	r.client.Del(context.Background(), spotKey(t, mk, classification, x, y))
}

func (r *Registry) ClearAllSpotsForMap(t tenant.Model, mk MapKey) {
	pattern := spotPatternKey(t, mk)
	var cursor uint64
	for {
		keys, next, err := r.client.Scan(context.Background(), cursor, pattern, 100).Result()
		if err != nil {
			return
		}
		if len(keys) > 0 {
			r.client.Del(context.Background(), keys...)
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
}
