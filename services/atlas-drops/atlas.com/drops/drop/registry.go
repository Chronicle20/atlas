package drop

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	objectid "github.com/Chronicle20/atlas/libs/atlas-object-id"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

type dropEntry struct {
	Drop       Model  `json:"drop"`
	ReservedBy uint32 `json:"reservedBy"`
}

type DropRegistry struct {
	entries   *atlas.TenantRegistry[uint32, dropEntry]
	all       *atlas.Set
	mapSets   *atlas.TenantKeyedSet[field.Model]
	allocator objectid.Allocator
}

var registry *DropRegistry

func InitRegistry(client *goredis.Client) {
	registry = &DropRegistry{
		entries: atlas.NewTenantRegistry[uint32, dropEntry](client, "drop", func(id uint32) string {
			return strconv.FormatUint(uint64(id), 10)
		}),
		all: atlas.NewSet(client, "drops:all"),
		mapSets: atlas.NewTenantKeyedSet[field.Model](client, "drops:map", func(f field.Model) string {
			return fmt.Sprintf("%d:%d:%d:%s", f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String())
		}),
		allocator: objectid.NewRedisAllocator(client),
	}
}

func GetRegistry() *DropRegistry {
	return registry
}

func dropIdStr(id uint32) string {
	return strconv.FormatUint(uint64(id), 10)
}

// allSetMember encodes a tenant+id pair for the global drops:all set.
// Format: "<tenantKey>:<id>" where tenantKey = "<uuid>:<region>:<major>.<minor>".
// This allows GetAllDrops to fully reconstruct the tenant without an external registry.
func allSetMember(t tenant.Model, id uint32) string {
	return fmt.Sprintf("%s:%d", atlas.TenantKey(t), id)
}

// parseTenantFromKey reconstructs a tenant.Model from a tenantKey string of the
// form "<uuid>:<region>:<major>.<minor>". Returns error if any segment is invalid.
func parseTenantFromKey(tenantKey string) (tenant.Model, error) {
	// TenantKey format: "<uuid>:<region>:<major>.<minor>"
	// UUID string is 36 chars with hyphens and no colons, so first 36 chars = uuid,
	// then ':' separator, then region (no dots), then ':' separator, then "<major>.<minor>".
	if len(tenantKey) < 38 { // 36 (uuid) + 1 (:) + at least 1 char
		return tenant.Model{}, errors.New("tenant key too short")
	}
	tenantId, err := uuid.Parse(tenantKey[:36])
	if err != nil {
		return tenant.Model{}, fmt.Errorf("parse tenant uuid: %w", err)
	}
	rest := tenantKey[37:] // skip uuid + ':'
	lastColon := strings.LastIndex(rest, ":")
	if lastColon < 0 {
		return tenant.Model{}, errors.New("missing version segment in tenant key")
	}
	region := rest[:lastColon]
	versionStr := rest[lastColon+1:]
	dot := strings.Index(versionStr, ".")
	if dot < 0 {
		return tenant.Model{}, errors.New("missing dot in version segment")
	}
	major, err := strconv.ParseUint(versionStr[:dot], 10, 16)
	if err != nil {
		return tenant.Model{}, fmt.Errorf("parse major version: %w", err)
	}
	minor, err := strconv.ParseUint(versionStr[dot+1:], 10, 16)
	if err != nil {
		return tenant.Model{}, fmt.Errorf("parse minor version: %w", err)
	}
	return tenant.Create(tenantId, region, uint16(major), uint16(minor))
}

func (d *DropRegistry) loadEntry(t tenant.Model, id uint32) (dropEntry, bool) {
	entry, err := d.entries.Get(context.Background(), t, id)
	if err != nil {
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
	if err := d.entries.Put(ctx, t, drop.Id(), entry); err != nil {
		_ = d.allocator.Release(ctx, t, id)
		return Model{}, err
	}

	_ = d.all.Add(ctx, allSetMember(t, drop.Id()))
	_ = d.mapSets.Add(ctx, t, mb.field, dropIdStr(drop.Id()))

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
		if err := d.entries.Put(context.Background(), t, dropId, entry); err != nil {
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
	_ = d.entries.Put(context.Background(), t, dropId, entry)
}

func (d *DropRegistry) RemoveDrop(t tenant.Model, dropId uint32) (Model, error) {
	entry, ok := d.loadEntry(t, dropId)
	if !ok {
		return Model{}, nil
	}
	drop := entry.Drop
	ctx := context.Background()

	_ = d.entries.Remove(ctx, t, dropId)
	_ = d.all.Remove(ctx, allSetMember(t, dropId))
	_ = d.mapSets.Remove(ctx, t, drop.Field(), dropIdStr(dropId))
	_ = d.allocator.Release(ctx, t, dropId)

	return drop, nil
}

func (d *DropRegistry) GetDropsForMap(t tenant.Model, f field.Model) ([]Model, error) {
	members, err := d.mapSets.Members(context.Background(), t, f)
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
	members, err := d.all.Members(context.Background())
	if err != nil {
		return nil
	}
	drops := make([]Model, 0, len(members))
	for _, member := range members {
		// Member format: "<tenantKey>:<id>" where tenantKey = "<uuid>:<region>:<major>.<minor>".
		// The drop ID is always the last colon-separated segment; everything before is the tenant key.
		sep := strings.LastIndex(member, ":")
		if sep < 0 {
			continue
		}
		id, err := strconv.ParseUint(member[sep+1:], 10, 32)
		if err != nil {
			continue
		}
		te, err := parseTenantFromKey(member[:sep])
		if err != nil {
			continue
		}
		if drop, ok := d.getDrop(te, uint32(id)); ok {
			drops = append(drops, drop)
		}
	}
	return drops
}
