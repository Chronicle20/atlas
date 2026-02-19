package reactor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

const (
	nextIdKey      = "reactors:next_id"
	allReactorsKey = "reactors:all"
	minId          = uint32(1000000001)
	maxId          = uint32(2000000000)
)

type Registry struct {
	client *goredis.Client
}

var reg *Registry

func InitRegistry(client *goredis.Client) {
	reg = &Registry{client: client}
	client.SetNX(context.Background(), nextIdKey, minId-1, 0)
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

func reactorKey(id uint32) string {
	return fmt.Sprintf("reactor:%d", id)
}

func reactorIdStr(id uint32) string {
	return fmt.Sprintf("%d", id)
}

func mapSetKey(t tenant.Model, mk MapKey) string {
	return fmt.Sprintf("reactors:map:%s:%d:%d:%d:%s", t.Id().String(), mk.worldId, mk.channelId, mk.mapId, mk.instance.String())
}

func cooldownKey(t tenant.Model, mk MapKey, classification uint32, x int16, y int16) string {
	return fmt.Sprintf("reactor:cd:%s:%d:%d:%d:%s:%d:%d:%d", t.Id().String(), mk.worldId, mk.channelId, mk.mapId, mk.instance.String(), classification, x, y)
}

var incrScript = goredis.NewScript(`
local id = redis.call('INCR', KEYS[1])
if id > tonumber(ARGV[1]) then
    redis.call('SET', KEYS[1], ARGV[2])
    return tonumber(ARGV[2])
end
return id
`)

func (r *Registry) getNextId() uint32 {
	result, err := incrScript.Run(context.Background(), r.client, []string{nextIdKey}, maxId, minId).Int64()
	if err != nil {
		return minId
	}
	return uint32(result)
}

func (r *Registry) store(id uint32, m Model) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return r.client.Set(context.Background(), reactorKey(id), data, 0).Err()
}

func (r *Registry) load(id uint32) (Model, bool) {
	data, err := r.client.Get(context.Background(), reactorKey(id)).Bytes()
	if err != nil {
		return Model{}, false
	}
	var m Model
	if err := json.Unmarshal(data, &m); err != nil {
		return Model{}, false
	}
	return m, true
}

func (r *Registry) Get(id uint32) (Model, error) {
	m, ok := r.load(id)
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
		id, err := strconv.ParseUint(member, 10, 32)
		if err != nil {
			continue
		}
		if m, ok := r.load(uint32(id)); ok {
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
		if m, ok := r.load(uint32(id)); ok {
			result = append(result, m)
		}
	}
	return result
}

func (r *Registry) Create(t tenant.Model, b *ModelBuilder) (Model, error) {
	id := r.getNextId()
	m, err := b.SetId(id).UpdateTime().Build()
	if err != nil {
		return Model{}, err
	}

	if err := r.store(id, m); err != nil {
		return Model{}, err
	}

	mk := NewMapKey(m.Field())
	idStr := reactorIdStr(id)
	pipe := r.client.Pipeline()
	pipe.SAdd(context.Background(), allReactorsKey, idStr)
	pipe.SAdd(context.Background(), mapSetKey(t, mk), idStr)
	_, _ = pipe.Exec(context.Background())

	return m, nil
}

func (r *Registry) Update(id uint32, modifier func(*ModelBuilder)) (Model, error) {
	m, ok := r.load(id)
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

	if err := r.store(id, updated); err != nil {
		return Model{}, err
	}
	return updated, nil
}

func (r *Registry) Remove(t tenant.Model, id uint32) {
	m, ok := r.load(id)
	if !ok {
		return
	}

	mk := NewMapKey(m.Field())
	idStr := reactorIdStr(id)

	pipe := r.client.Pipeline()
	pipe.Del(context.Background(), reactorKey(id))
	pipe.SRem(context.Background(), allReactorsKey, idStr)
	pipe.SRem(context.Background(), mapSetKey(t, mk), idStr)
	_, _ = pipe.Exec(context.Background())
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
