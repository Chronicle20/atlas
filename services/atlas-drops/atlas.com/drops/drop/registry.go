package drop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	objectid "github.com/Chronicle20/atlas/libs/atlas-object-id"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

const (
	allDropsKey = "drops:all"
)

type dropEntry struct {
	Drop       Model  `json:"drop"`
	ReservedBy uint32 `json:"reservedBy"`
}

type DropRegistry struct {
	client    *goredis.Client
	allocator objectid.Allocator
}

var registry *DropRegistry

func InitRegistry(client *goredis.Client) {
	registry = &DropRegistry{client: client, allocator: objectid.NewRedisAllocator(client)}
}

func GetRegistry() *DropRegistry {
	return registry
}

func dropKey(t tenant.Model, id uint32) string {
	return fmt.Sprintf("drop:%s:%d", t.Id().String(), id)
}

func dropIdStr(id uint32) string {
	return fmt.Sprintf("%d", id)
}

func mapSetKey(tenantId uuid.UUID, f field.Model) string {
	return fmt.Sprintf("drops:map:%s:%d:%d:%d:%s", tenantId.String(), f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String())
}

// allSetMember encodes a tenant+id pair for the global drops:all set.
func allSetMember(t tenant.Model, id uint32) string {
	return fmt.Sprintf("%s:%d", t.Id().String(), id)
}

func (d *DropRegistry) storeEntry(t tenant.Model, id uint32, entry dropEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	return d.client.Set(context.Background(), dropKey(t, id), data, 0).Err()
}

func (d *DropRegistry) loadEntry(t tenant.Model, id uint32) (dropEntry, bool) {
	data, err := d.client.Get(context.Background(), dropKey(t, id)).Bytes()
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
	ctx := context.Background()

	id, err := d.allocator.Allocate(ctx, t)
	if err != nil {
		return Model{}, fmt.Errorf("allocate drop oid: %w", err)
	}

	drop, err := mb.SetId(id).SetStatus(StatusAvailable).Build()
	if err != nil {
		_ = d.allocator.Release(ctx, t, id)
		return Model{}, err
	}

	entry := dropEntry{Drop: drop}
	if err := d.storeEntry(t, drop.Id(), entry); err != nil {
		_ = d.allocator.Release(ctx, t, id)
		return Model{}, err
	}

	mk := mapSetKey(t.Id(), mb.field)
	idStr := dropIdStr(drop.Id())
	pipe := d.client.Pipeline()
	pipe.SAdd(ctx, allDropsKey, allSetMember(t, drop.Id()))
	pipe.SAdd(ctx, mk, idStr)
	_, _ = pipe.Exec(ctx)

	return drop, nil
}

func (d *DropRegistry) getDrop(t tenant.Model, dropId uint32) (Model, bool) {
	entry, ok := d.loadEntry(t, dropId)
	if !ok {
		return Model{}, false
	}
	return entry.Drop, true
}

func (d *DropRegistry) GetDrop(t tenant.Model, dropId uint32) (Model, error) {
	drop, ok := d.getDrop(t, dropId)
	if !ok {
		return Model{}, errors.New("drop not found")
	}
	return drop, nil
}

func (d *DropRegistry) ReserveDrop(t tenant.Model, dropId uint32, characterId uint32, partyId uint32, petSlot int8) (Model, error) {
	entry, ok := d.loadEntry(t, dropId)
	if !ok {
		return Model{}, errors.New("unable to locate drop")
	}

	if !entry.Drop.CanBeReservedBy(characterId, partyId) {
		return Model{}, errors.New("drop is not available for this character")
	}

	if entry.Drop.Status() == StatusAvailable {
		entry.Drop = entry.Drop.Reserve(petSlot)
		entry.ReservedBy = characterId
		if err := d.storeEntry(t, dropId, entry); err != nil {
			return Model{}, err
		}
		return entry.Drop, nil
	}

	if entry.ReservedBy == characterId {
		return entry.Drop, nil
	}
	return Model{}, errors.New("reserved by another party")
}

func (d *DropRegistry) CancelDropReservation(t tenant.Model, dropId uint32, characterId uint32) {
	entry, ok := d.loadEntry(t, dropId)
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
	_ = d.storeEntry(t, dropId, entry)
}

func (d *DropRegistry) RemoveDrop(t tenant.Model, dropId uint32) (Model, error) {
	entry, ok := d.loadEntry(t, dropId)
	if !ok {
		return Model{}, nil
	}

	drop := entry.Drop
	ctx := context.Background()
	mk := mapSetKey(t.Id(), drop.Field())
	idStr := dropIdStr(dropId)

	pipe := d.client.Pipeline()
	pipe.Del(ctx, dropKey(t, dropId))
	pipe.SRem(ctx, allDropsKey, allSetMember(t, dropId))
	pipe.SRem(ctx, mk, idStr)
	_, _ = pipe.Exec(ctx)

	_ = d.allocator.Release(ctx, t, dropId)

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
		if drop, ok := d.getDrop(t, uint32(id)); ok {
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
		// Members are stored as "{tenantId}:{id}". Skip legacy "{id}"-only rows.
		sep := strings.LastIndex(member, ":")
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
		if drop, ok := d.getDrop(te, uint32(id)); ok {
			drops = append(drops, drop)
		}
	}
	return drops
}
