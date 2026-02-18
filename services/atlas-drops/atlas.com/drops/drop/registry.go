package drop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/Chronicle20/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

const (
	nextIdKey   = "drops:next_id"
	allDropsKey = "drops:all"
	minId       = uint32(1000000001)
	maxId       = uint32(2000000000)
)

type dropEntry struct {
	Drop       Model  `json:"drop"`
	ReservedBy uint32 `json:"reservedBy"`
}

type DropRegistry struct {
	client *goredis.Client
}

var registry *DropRegistry

func InitRegistry(client *goredis.Client) {
	registry = &DropRegistry{client: client}
	client.SetNX(context.Background(), nextIdKey, minId-1, 0)
}

func GetRegistry() *DropRegistry {
	return registry
}

func dropKey(id uint32) string {
	return fmt.Sprintf("drop:%d", id)
}

func dropIdStr(id uint32) string {
	return fmt.Sprintf("%d", id)
}

func mapSetKey(tenantId uuid.UUID, f field.Model) string {
	return fmt.Sprintf("drops:map:%s:%d:%d:%d:%s", tenantId.String(), f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String())
}

var incrScript = goredis.NewScript(`
local id = redis.call('INCR', KEYS[1])
if id > tonumber(ARGV[1]) then
    redis.call('SET', KEYS[1], ARGV[2])
    return tonumber(ARGV[2])
end
return id
`)

func (d *DropRegistry) getNextUniqueId() uint32 {
	result, err := incrScript.Run(context.Background(), d.client, []string{nextIdKey}, maxId, minId).Int64()
	if err != nil {
		return minId
	}
	return uint32(result)
}

func (d *DropRegistry) storeEntry(id uint32, entry dropEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	return d.client.Set(context.Background(), dropKey(id), data, 0).Err()
}

func (d *DropRegistry) loadEntry(id uint32) (dropEntry, bool) {
	data, err := d.client.Get(context.Background(), dropKey(id)).Bytes()
	if err != nil {
		return dropEntry{}, false
	}
	var entry dropEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return dropEntry{}, false
	}
	return entry, true
}

func (d *DropRegistry) CreateDrop(mb *ModelBuilder) (Model, error) {
	t := mb.Tenant()
	currentUniqueId := d.getNextUniqueId()

	drop, err := mb.SetId(currentUniqueId).SetStatus(StatusAvailable).Build()
	if err != nil {
		return Model{}, err
	}

	entry := dropEntry{Drop: drop}
	if err := d.storeEntry(drop.Id(), entry); err != nil {
		return Model{}, err
	}

	mk := mapSetKey(t.Id(), mb.field)
	idStr := dropIdStr(drop.Id())
	pipe := d.client.Pipeline()
	pipe.SAdd(context.Background(), allDropsKey, idStr)
	pipe.SAdd(context.Background(), mk, idStr)
	_, _ = pipe.Exec(context.Background())

	return drop, nil
}

func (d *DropRegistry) getDrop(dropId uint32) (Model, bool) {
	entry, ok := d.loadEntry(dropId)
	if !ok {
		return Model{}, false
	}
	return entry.Drop, true
}

func (d *DropRegistry) GetDrop(dropId uint32) (Model, error) {
	drop, ok := d.getDrop(dropId)
	if !ok {
		return Model{}, errors.New("drop not found")
	}
	return drop, nil
}

func (d *DropRegistry) ReserveDrop(dropId uint32, characterId uint32, petSlot int8) (Model, error) {
	entry, ok := d.loadEntry(dropId)
	if !ok {
		return Model{}, errors.New("unable to locate drop")
	}

	if entry.Drop.Status() == StatusAvailable {
		entry.Drop = entry.Drop.Reserve(petSlot)
		entry.ReservedBy = characterId
		if err := d.storeEntry(dropId, entry); err != nil {
			return Model{}, err
		}
		return entry.Drop, nil
	}

	if entry.ReservedBy == characterId {
		return entry.Drop, nil
	}
	return Model{}, errors.New("reserved by another party")
}

func (d *DropRegistry) CancelDropReservation(dropId uint32, characterId uint32) {
	entry, ok := d.loadEntry(dropId)
	if !ok {
		return
	}

	if entry.ReservedBy != characterId {
		return
	}

	if entry.Drop.Status() != StatusReserved {
		return
	}

	entry.Drop = entry.Drop.CancelReservation()
	entry.ReservedBy = 0
	_ = d.storeEntry(dropId, entry)
}

func (d *DropRegistry) RemoveDrop(dropId uint32) (Model, error) {
	entry, ok := d.loadEntry(dropId)
	if !ok {
		return Model{}, nil
	}

	drop := entry.Drop
	t := drop.Tenant()
	mk := mapSetKey(t.Id(), drop.Field())
	idStr := dropIdStr(dropId)

	pipe := d.client.Pipeline()
	pipe.Del(context.Background(), dropKey(dropId))
	pipe.SRem(context.Background(), allDropsKey, idStr)
	pipe.SRem(context.Background(), mk, idStr)
	_, _ = pipe.Exec(context.Background())

	return drop, nil
}

func (d *DropRegistry) GetDropsForMap(t tenant.Model, f field.Model) ([]Model, error) {
	mk := mapSetKey(t.Id(), f)
	members, err := d.client.SMembers(context.Background(), mk).Result()
	if err != nil {
		return make([]Model, 0), nil
	}

	drops := make([]Model, 0, len(members))
	for _, member := range members {
		id, err := strconv.ParseUint(member, 10, 32)
		if err != nil {
			continue
		}
		if drop, ok := d.getDrop(uint32(id)); ok {
			drops = append(drops, drop)
		}
	}
	return drops, nil
}

func (d *DropRegistry) GetAllDrops() []Model {
	members, err := d.client.SMembers(context.Background(), allDropsKey).Result()
	if err != nil {
		return nil
	}

	drops := make([]Model, 0, len(members))
	for _, member := range members {
		id, err := strconv.ParseUint(member, 10, 32)
		if err != nil {
			continue
		}
		if drop, ok := d.getDrop(uint32(id)); ok {
			drops = append(drops, drop)
		}
	}
	return drops
}
