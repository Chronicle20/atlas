package reactor

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	objectid "github.com/Chronicle20/atlas/libs/atlas-object-id"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

type Registry struct {
	reactors  *atlas.TenantRegistry[uint32, Model]
	all       *atlas.Set
	mapSets   *atlas.TenantKeyedSet[MapKey]
	cooldowns *atlas.TenantKeyedHash[MapKey] // field=class:x:y -> expiry unix ms
	spots     *atlas.TenantKeyedHash[MapKey] // field=class:x:y -> "1"
	allocator objectid.Allocator
}

var reg *Registry

func mapKeyFn(mk MapKey) string {
	return fmt.Sprintf("%d:%d:%d:%s", mk.worldId, mk.channelId, mk.mapId, mk.instance.String())
}

func InitRegistry(client *goredis.Client) {
	reg = &Registry{
		reactors: atlas.NewTenantRegistry[uint32, Model](client, "reactor", func(id uint32) string {
			return strconv.FormatUint(uint64(id), 10)
		}),
		all:       atlas.NewSet(client, "reactors:all"),
		mapSets:   atlas.NewTenantKeyedSet[MapKey](client, "reactors:map", mapKeyFn),
		cooldowns: atlas.NewTenantKeyedHash[MapKey](client, "reactor:cd", mapKeyFn),
		spots:     atlas.NewTenantKeyedHash[MapKey](client, "reactor:spot", mapKeyFn),
		allocator: objectid.NewRedisAllocator(client),
	}
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

func reactorIdStr(id uint32) string {
	return strconv.FormatUint(uint64(id), 10)
}

// allSetMember encodes a tenant+id pair for the global reactors:all set.
// Format: {uuid}:{region}:{major}:{minor}:{id}
func allSetMember(t tenant.Model, id uint32) string {
	return fmt.Sprintf("%s:%s:%d:%d:%d", t.Id().String(), t.Region(), t.MajorVersion(), t.MinorVersion(), id)
}

// posField is the hash field for cooldown/spot entries within a map hash.
func posField(classification uint32, x int16, y int16) string {
	return fmt.Sprintf("%d:%d:%d", classification, x, y)
}

func (r *Registry) load(t tenant.Model, id uint32) (Model, bool) {
	m, err := r.reactors.Get(context.Background(), t, id)
	if err != nil {
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
	members, err := r.all.Members(context.Background())
	if err != nil {
		return make(map[tenant.Model][]Model)
	}
	res := make(map[tenant.Model][]Model)
	for _, member := range members {
		// Members are stored as "{uuid}:{region}:{major}:{minor}:{id}".
		// A UUID is 36 chars; skip past it to find the first colon after the UUID.
		if len(member) < 38 { // 36 uuid + 1 colon + at least 1 char
			continue
		}
		uuidStr := member[:36]
		rest := member[37:] // skip uuid + first ':'
		// rest = "{region}:{major}:{minor}:{id}"
		// Find the last ':' to separate id from the tenant fields.
		lastColon := -1
		for i := len(rest) - 1; i >= 0; i-- {
			if rest[i] == ':' {
				lastColon = i
				break
			}
		}
		if lastColon < 0 {
			continue
		}
		id, err := strconv.ParseUint(rest[lastColon+1:], 10, 32)
		if err != nil {
			continue
		}
		tenantFields := rest[:lastColon] // "{region}:{major}:{minor}"
		// Split into region, major, minor.
		var region string
		var major, minor uint64
		// tenantFields has the format "{region}:{major}:{minor}".
		// Find the last two colons (from right) to parse major and minor.
		lc2 := -1
		for i := len(tenantFields) - 1; i >= 0; i-- {
			if tenantFields[i] == ':' {
				lc2 = i
				break
			}
		}
		if lc2 < 0 {
			continue
		}
		minor, err = strconv.ParseUint(tenantFields[lc2+1:], 10, 16)
		if err != nil {
			continue
		}
		regionMajor := tenantFields[:lc2]
		lc3 := -1
		for i := len(regionMajor) - 1; i >= 0; i-- {
			if regionMajor[i] == ':' {
				lc3 = i
				break
			}
		}
		if lc3 < 0 {
			continue
		}
		major, err = strconv.ParseUint(regionMajor[lc3+1:], 10, 16)
		if err != nil {
			continue
		}
		region = regionMajor[:lc3]
		tenantId, err := uuid.Parse(uuidStr)
		if err != nil {
			continue
		}
		te, err := tenant.Create(tenantId, region, uint16(major), uint16(minor))
		if err != nil {
			continue
		}
		if m, ok := r.load(te, uint32(id)); ok {
			res[m.Tenant()] = append(res[m.Tenant()], m)
		}
	}
	return res
}

func (r *Registry) GetInField(t tenant.Model, f field.Model) []Model {
	mk := NewMapKey(f)
	members, err := r.mapSets.Members(context.Background(), t, mk)
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
	if err := r.reactors.Put(ctx, t, id, m); err != nil {
		_ = r.allocator.Release(ctx, t, id)
		return Model{}, err
	}
	mk := NewMapKey(m.Field())
	_ = r.all.Add(ctx, allSetMember(t, id))
	_ = r.mapSets.Add(ctx, t, mk, reactorIdStr(id))
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
	if err := r.reactors.Put(context.Background(), t, id, updated); err != nil {
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
	_ = r.reactors.Remove(ctx, t, id)
	_ = r.all.Remove(ctx, allSetMember(t, id))
	_ = r.mapSets.Remove(ctx, t, mk, reactorIdStr(id))
	_ = r.allocator.Release(ctx, t, id)
}

func (r *Registry) RecordCooldown(t tenant.Model, mk MapKey, classification uint32, x int16, y int16, delay uint32) {
	if delay == 0 {
		return
	}
	expiry := time.Now().Add(time.Millisecond * time.Duration(delay)).UnixMilli()
	_ = r.cooldowns.Set(context.Background(), t, mk, posField(classification, x, y), strconv.FormatInt(expiry, 10))
}

func (r *Registry) IsOnCooldown(t tenant.Model, mk MapKey, classification uint32, x int16, y int16) bool {
	v, err := r.cooldowns.Get(context.Background(), t, mk, posField(classification, x, y))
	if err != nil {
		return false
	}
	expiry, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return false
	}
	if time.Now().UnixMilli() >= expiry {
		// Lazily prune the stale field.
		_ = r.cooldowns.Del(context.Background(), t, mk, posField(classification, x, y))
		return false
	}
	return true
}

func (r *Registry) ClearCooldown(t tenant.Model, mk MapKey, classification uint32, x int16, y int16) {
	_ = r.cooldowns.Del(context.Background(), t, mk, posField(classification, x, y))
}

func (r *Registry) ClearAllCooldownsForMap(t tenant.Model, mk MapKey) {
	_ = r.cooldowns.DeleteKey(context.Background(), t, mk)
}

func (r *Registry) CleanupExpiredCooldowns() {
	// No-op: cooldowns are pruned lazily in IsOnCooldown and cleared per-map.
}

// TryClaimSpot atomically reserves a (classification, x, y) slot within a map
// instance. Returns true if this caller owns the slot.
func (r *Registry) TryClaimSpot(t tenant.Model, mk MapKey, classification uint32, x int16, y int16) bool {
	ok, err := r.spots.SetNX(context.Background(), t, mk, posField(classification, x, y), "1")
	if err != nil {
		return false
	}
	return ok
}

func (r *Registry) ReleaseSpot(t tenant.Model, mk MapKey, classification uint32, x int16, y int16) {
	_ = r.spots.Del(context.Background(), t, mk, posField(classification, x, y))
}

func (r *Registry) ClearAllSpotsForMap(t tenant.Model, mk MapKey) {
	_ = r.spots.DeleteKey(context.Background(), t, mk)
}
