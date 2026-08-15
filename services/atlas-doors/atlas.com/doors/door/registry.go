package door

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/point"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	atlasredis "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// storedDoor is the JSON-serializable representation stored in Redis.
type storedDoor struct {
	// tenant
	TenantId string `json:"tenantId"`
	Region   string `json:"region"`
	Major    uint16 `json:"major"`
	Minor    uint16 `json:"minor"`
	// field
	WorldId   byte   `json:"worldId"`
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
	Instance  string `json:"instance"`
	// door
	AreaDoorId       uint32 `json:"areaDoorId"`
	TownDoorId       uint32 `json:"townDoorId"`
	OwnerCharacterId uint32 `json:"ownerCharacterId"`
	PartyId          uint32 `json:"partyId"`
	SkillId          uint32 `json:"skillId"`
	SkillLevel       byte   `json:"skillLevel"`
	TownMapId        uint32 `json:"townMapId"`
	Slot             byte   `json:"slot"`
	TownPortalId     uint32 `json:"townPortalId"`
	AreaX            int16  `json:"areaX"`
	AreaY            int16  `json:"areaY"`
	TownX            int16  `json:"townX"`
	TownY            int16  `json:"townY"`
	DeployMs         int64  `json:"deployMs"`
	ExpiresMs        int64  `json:"expiresMs"`
}

// Registry holds the primary door store plus three secondary indices:
// field (for area-door spawn + field broadcast), owner (for recast/cleanup),
// and town-party (for slot allocation + town broadcast). All reads/writes go
// through the tenant-scoped fields below, which key every entry as
// <prefix>:<namespace>:<TenantKey(t)>:<entityKey>. GetAll (expiry sweep) and
// Clear (test teardown) are the two genuine cross-tenant operations and use
// the deliberate, explicitly-named *AcrossTenants sibling methods (D7)
// instead of a bare non-tenant-scoped instance.
type Registry struct {
	reg      *atlasredis.TenantRegistry[string, storedDoor]
	fieldIdx *atlasredis.TenantKeyedSet[string]
	ownerIdx *atlasredis.TenantKeyedSet[string]
	townIdx  *atlasredis.TenantKeyedSet[string]
}

var (
	registry *Registry
	once     sync.Once
)

func newRegistry(rc *goredis.Client) *Registry {
	id := func(s string) string { return s }
	return &Registry{
		reg:      atlasredis.NewTenantRegistry[string, storedDoor](rc, "door", id),
		fieldIdx: atlasredis.NewTenantKeyedSet[string](rc, "door-field", id),
		ownerIdx: atlasredis.NewTenantKeyedSet[string](rc, "door-owner", id),
		townIdx:  atlasredis.NewTenantKeyedSet[string](rc, "door-town", id),
	}
}

// InitRegistry initialises the singleton registry. Safe to call multiple times;
// only the first call takes effect (sync.Once).
func InitRegistry(rc *goredis.Client) { once.Do(func() { registry = newRegistry(rc) }) }

// GetRegistry returns the singleton door registry.
func GetRegistry() *Registry { return registry }

// --------------------------------------------------------------------------
// Key helpers
// --------------------------------------------------------------------------

// storeSuffix is the entity-key tail for the primary store.
// Full Redis key: atlas:door:<TenantKey(t)>:<areaDoorId>
func storeSuffix(areaDoorId uint32) string {
	return fmt.Sprintf("%d", areaDoorId)
}

// fieldSuffix is the entity-key tail for the field index SET.
// Full Redis key: atlas:door-field:<TenantKey(t)>:<world>:<channel>:<map>:<instance>
func fieldSuffix(f field.Model) string {
	return fmt.Sprintf("%d:%d:%d:%s",
		byte(f.WorldId()), byte(f.ChannelId()), uint32(f.MapId()),
		f.Instance().String())
}

// ownerSuffix is the entity-key tail for the owner index SET.
// Full Redis key: atlas:door-owner:<TenantKey(t)>:<characterId>
func ownerSuffix(characterId character.Id) string {
	return fmt.Sprintf("%d", uint32(characterId))
}

// partyScope returns a discriminator that prevents two solo casters at the same
// town from sharing a town-party index bucket (design §4.3).
func partyScope(partyId uint32, ownerCharacterId character.Id) string {
	if partyId != 0 {
		return fmt.Sprintf("%d", partyId)
	}
	return fmt.Sprintf("solo-%d", uint32(ownerCharacterId))
}

// townSuffix is the entity-key tail for the town-party index SET.
// Full Redis key: atlas:door-town:<TenantKey(t)>:<world>:<channel>:<townMap>:<partyScope>
func townSuffix(f field.Model, townMapId _map.Id, partyId uint32, ownerCharacterId character.Id) string {
	return fmt.Sprintf("%d:%d:%d:%s",
		byte(f.WorldId()), byte(f.ChannelId()), uint32(townMapId),
		partyScope(partyId, ownerCharacterId))
}

// memberKey is the string stored inside index SETs — the areaDoorId as decimal.
func memberKey(areaDoorId uint32) string {
	return fmt.Sprintf("%d", areaDoorId)
}

// --------------------------------------------------------------------------
// Stored ↔ domain converters
// --------------------------------------------------------------------------

func toStored(t tenant.Model, m Model) storedDoor {
	return storedDoor{
		TenantId:         t.Id().String(),
		Region:           t.Region(),
		Major:            t.MajorVersion(),
		Minor:            t.MinorVersion(),
		WorldId:          byte(m.fld.WorldId()),
		ChannelId:        byte(m.fld.ChannelId()),
		MapId:            uint32(m.fld.MapId()),
		Instance:         m.fld.Instance().String(),
		AreaDoorId:       m.areaDoorId,
		TownDoorId:       m.townDoorId,
		OwnerCharacterId: uint32(m.ownerCharacterId),
		PartyId:          m.partyId,
		SkillId:          uint32(m.skillId),
		SkillLevel:       m.skillLevel,
		TownMapId:        uint32(m.townMapId),
		Slot:             m.slot,
		TownPortalId:     m.townPortalId,
		AreaX:            int16(m.areaX),
		AreaY:            int16(m.areaY),
		TownX:            int16(m.townX),
		TownY:            int16(m.townY),
		DeployMs:         timeToMs(m.deployTime),
		ExpiresMs:        timeToMs(m.expiresAt),
	}
}

func fromStored(sd storedDoor) (tenant.Model, Model, error) {
	tenantId, err := uuid.Parse(sd.TenantId)
	if err != nil {
		return tenant.Model{}, Model{}, fmt.Errorf("parse tenantId: %w", err)
	}
	t, err := tenant.Create(tenantId, sd.Region, sd.Major, sd.Minor)
	if err != nil {
		return tenant.Model{}, Model{}, fmt.Errorf("create tenant: %w", err)
	}
	inst, err := uuid.Parse(sd.Instance)
	if err != nil {
		return tenant.Model{}, Model{}, fmt.Errorf("parse instance: %w", err)
	}
	f := field.NewBuilder(world.Id(sd.WorldId), channel.Id(sd.ChannelId), _map.Id(sd.MapId)).
		SetInstance(inst).Build()

	m := NewBuilder().
		SetAreaDoorId(sd.AreaDoorId).
		SetTownDoorId(sd.TownDoorId).
		SetOwnerCharacterId(character.Id(sd.OwnerCharacterId)).
		SetPartyId(sd.PartyId).
		SetSkillId(skill.Id(sd.SkillId)).
		SetSkillLevel(sd.SkillLevel).
		SetField(f).
		SetTownMapId(_map.Id(sd.TownMapId)).
		SetSlot(sd.Slot).
		SetTownPortalId(sd.TownPortalId).
		SetAreaX(point.X(sd.AreaX)).
		SetAreaY(point.Y(sd.AreaY)).
		SetTownX(point.X(sd.TownX)).
		SetTownY(point.Y(sd.TownY)).
		SetDeployTime(msToTime(sd.DeployMs)).
		SetExpiresAt(msToTime(sd.ExpiresMs)).
		Build()

	return t, m, nil
}

// timeToMs converts t to Unix milliseconds, returning 0 for the zero value.
func timeToMs(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixMilli()
}

// msToTime converts Unix milliseconds to a time.Time, returning zero for ms==0.
func msToTime(ms int64) time.Time {
	if ms == 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms)
}

// --------------------------------------------------------------------------
// Registry methods
// --------------------------------------------------------------------------

var errDoorNotFound = errors.New("door not found")

// Put stores the door in the primary registry and adds it to all three indices.
func (r *Registry) Put(ctx context.Context, t tenant.Model, m Model) error {
	if err := r.reg.Put(ctx, t, storeSuffix(m.areaDoorId), toStored(t, m)); err != nil {
		return err
	}
	mk := memberKey(m.areaDoorId)
	_ = r.fieldIdx.Add(ctx, t, fieldSuffix(m.fld), mk)
	_ = r.ownerIdx.Add(ctx, t, ownerSuffix(m.ownerCharacterId), mk)
	_ = r.townIdx.Add(ctx, t, townSuffix(m.fld, m.townMapId, m.partyId, m.ownerCharacterId), mk)
	return nil
}

// Get retrieves a single door by its areaDoorId.
func (r *Registry) Get(ctx context.Context, t tenant.Model, areaDoorId uint32) (Model, error) {
	sd, err := r.reg.Get(ctx, t, storeSuffix(areaDoorId))
	if errors.Is(err, atlasredis.ErrNotFound) {
		return Model{}, errDoorNotFound
	}
	if err != nil {
		return Model{}, err
	}
	_, m, err := fromStored(sd)
	return m, err
}

// GetInField returns all doors whose area field matches f.
func (r *Registry) GetInField(ctx context.Context, t tenant.Model, f field.Model) ([]Model, error) {
	return r.lookupByIndex(ctx, t, r.fieldIdx, fieldSuffix(f))
}

// GetByOwner returns all doors owned by characterId.
func (r *Registry) GetByOwner(ctx context.Context, t tenant.Model, characterId character.Id) ([]Model, error) {
	return r.lookupByIndex(ctx, t, r.ownerIdx, ownerSuffix(characterId))
}

// GetInTownParty returns all doors in the town-party bucket for the given
// field, townMapId, and party/owner scope.
func (r *Registry) GetInTownParty(ctx context.Context, t tenant.Model, f field.Model, townMapId _map.Id, partyId uint32, ownerCharacterId character.Id) ([]Model, error) {
	return r.lookupByIndex(ctx, t, r.townIdx, townSuffix(f, townMapId, partyId, ownerCharacterId))
}

// lookupByIndex fetches all doors referenced by a secondary index SET.
func (r *Registry) lookupByIndex(ctx context.Context, t tenant.Model, idx *atlasredis.TenantKeyedSet[string], suffix string) ([]Model, error) {
	members, err := idx.Members(ctx, t, suffix)
	if err != nil || len(members) == 0 {
		return nil, err
	}
	result := make([]Model, 0, len(members))
	for _, mk := range members {
		// Parse the areaDoorId from the member string — stored as decimal.
		var id uint32
		if _, err := fmt.Sscanf(mk, "%d", &id); err != nil {
			continue
		}
		sd, gerr := r.reg.Get(ctx, t, storeSuffix(id))
		if gerr != nil {
			continue
		}
		_, m, gerr := fromStored(sd)
		if gerr != nil {
			continue
		}
		result = append(result, m)
	}
	return result, nil
}

// Remove deletes a door and clears it from all three indices. It reads the
// stored door first to reconstruct the exact index keys.
func (r *Registry) Remove(ctx context.Context, t tenant.Model, areaDoorId uint32) error {
	sd, err := r.reg.Get(ctx, t, storeSuffix(areaDoorId))
	if errors.Is(err, atlasredis.ErrNotFound) {
		return errDoorNotFound
	}
	if err != nil {
		return err
	}
	_, m, err := fromStored(sd)
	if err != nil {
		return err
	}

	mk := memberKey(areaDoorId)
	_ = r.fieldIdx.Remove(ctx, t, fieldSuffix(m.fld), mk)
	_ = r.ownerIdx.Remove(ctx, t, ownerSuffix(m.ownerCharacterId), mk)
	_ = r.townIdx.Remove(ctx, t, townSuffix(m.fld, m.townMapId, m.partyId, m.ownerCharacterId), mk)
	_ = r.reg.Remove(ctx, t, storeSuffix(areaDoorId))

	return nil
}

// GetAll returns all doors grouped by tenant. This is a deliberate,
// explicitly-named cross-tenant enumeration (D7) via
// TenantRegistry.GetAllAcrossTenants — the expiry sweep has no tenant to
// loop over; it needs every tenant's live doors at once.
func (r *Registry) GetAll(ctx context.Context) (map[tenant.Model][]Model, error) {
	result := make(map[tenant.Model][]Model)
	all, err := r.reg.GetAllAcrossTenants(ctx)
	if err != nil {
		return result, err
	}
	for _, sd := range all {
		t, m, derr := fromStored(sd)
		if derr != nil {
			continue
		}
		result[t] = append(result[t], m)
	}
	return result, nil
}

// Clear removes all doors and all index entries across every tenant (useful
// in tests). Same cross-tenant rationale as GetAll.
func (r *Registry) Clear(ctx context.Context) {
	_, _ = r.reg.ClearAllAcrossTenants(ctx)
	_, _ = r.fieldIdx.ClearAllAcrossTenants(ctx)
	_, _ = r.ownerIdx.ClearAllAcrossTenants(ctx)
	_, _ = r.townIdx.ClearAllAcrossTenants(ctx)
}
