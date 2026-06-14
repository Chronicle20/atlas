package door

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	atlasredis "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
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
// and town-party (for slot allocation + town broadcast).
type Registry struct {
	reg      *atlasredis.Registry[string, storedDoor]
	fieldIdx *atlasredis.KeyedSet[string]
	ownerIdx *atlasredis.KeyedSet[string]
	townIdx  *atlasredis.KeyedSet[string]
}

var registry *Registry
var once sync.Once

func newRegistry(rc *goredis.Client) *Registry {
	id := func(s string) string { return s }
	return &Registry{
		reg:      atlasredis.NewRegistry[string, storedDoor](rc, "door", id),
		fieldIdx: atlasredis.NewKeyedSet[string](rc, "door-field", id),
		ownerIdx: atlasredis.NewKeyedSet[string](rc, "door-owner", id),
		townIdx:  atlasredis.NewKeyedSet[string](rc, "door-town", id),
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
// Full Redis key: atlas:door:<tenantId>:<areaDoorId>
func storeSuffix(t tenant.Model, areaDoorId uint32) string {
	return fmt.Sprintf("%s:%d", t.Id().String(), areaDoorId)
}

// fieldSuffix is the entity-key tail for the field index SET.
// Full Redis key: atlas:door-field:<tenantId>:<world>:<channel>:<map>:<instance>
func fieldSuffix(t tenant.Model, f field.Model) string {
	return fmt.Sprintf("%s:%d:%d:%d:%s",
		t.Id().String(),
		byte(f.WorldId()), byte(f.ChannelId()), uint32(f.MapId()),
		f.Instance().String())
}

// ownerSuffix is the entity-key tail for the owner index SET.
// Full Redis key: atlas:door-owner:<tenantId>:<characterId>
func ownerSuffix(t tenant.Model, characterId uint32) string {
	return fmt.Sprintf("%s:%d", t.Id().String(), characterId)
}

// partyScope returns a discriminator that prevents two solo casters at the same
// town from sharing a town-party index bucket (design §4.3).
func partyScope(partyId, ownerCharacterId uint32) string {
	if partyId != 0 {
		return fmt.Sprintf("%d", partyId)
	}
	return fmt.Sprintf("solo-%d", ownerCharacterId)
}

// townSuffix is the entity-key tail for the town-party index SET.
// Full Redis key: atlas:door-town:<tenantId>:<world>:<channel>:<townMap>:<partyScope>
func townSuffix(t tenant.Model, f field.Model, townMapId _map.Id, partyId, ownerCharacterId uint32) string {
	return fmt.Sprintf("%s:%d:%d:%d:%s",
		t.Id().String(),
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
		OwnerCharacterId: m.ownerCharacterId,
		PartyId:          m.partyId,
		SkillId:          m.skillId,
		SkillLevel:       m.skillLevel,
		TownMapId:        uint32(m.townMapId),
		Slot:             m.slot,
		TownPortalId:     m.townPortalId,
		AreaX:            m.areaX,
		AreaY:            m.areaY,
		TownX:            m.townX,
		TownY:            m.townY,
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
		SetOwnerCharacterId(sd.OwnerCharacterId).
		SetPartyId(sd.PartyId).
		SetSkillId(sd.SkillId).
		SetSkillLevel(sd.SkillLevel).
		SetField(f).
		SetTownMapId(_map.Id(sd.TownMapId)).
		SetSlot(sd.Slot).
		SetTownPortalId(sd.TownPortalId).
		SetAreaX(sd.AreaX).
		SetAreaY(sd.AreaY).
		SetTownX(sd.TownX).
		SetTownY(sd.TownY).
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
	if err := r.reg.Put(ctx, storeSuffix(t, m.areaDoorId), toStored(t, m)); err != nil {
		return err
	}
	mk := memberKey(m.areaDoorId)
	_ = r.fieldIdx.Add(ctx, fieldSuffix(t, m.fld), mk)
	_ = r.ownerIdx.Add(ctx, ownerSuffix(t, m.ownerCharacterId), mk)
	_ = r.townIdx.Add(ctx, townSuffix(t, m.fld, m.townMapId, m.partyId, m.ownerCharacterId), mk)
	return nil
}

// Get retrieves a single door by its areaDoorId.
func (r *Registry) Get(ctx context.Context, t tenant.Model, areaDoorId uint32) (Model, error) {
	sd, err := r.reg.Get(ctx, storeSuffix(t, areaDoorId))
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
	return r.lookupByIndex(ctx, t, r.fieldIdx, fieldSuffix(t, f))
}

// GetByOwner returns all doors owned by characterId.
func (r *Registry) GetByOwner(ctx context.Context, t tenant.Model, characterId uint32) ([]Model, error) {
	return r.lookupByIndex(ctx, t, r.ownerIdx, ownerSuffix(t, characterId))
}

// GetInTownParty returns all doors in the town-party bucket for the given
// field, townMapId, and party/owner scope.
func (r *Registry) GetInTownParty(ctx context.Context, t tenant.Model, f field.Model, townMapId _map.Id, partyId, ownerCharacterId uint32) ([]Model, error) {
	return r.lookupByIndex(ctx, t, r.townIdx, townSuffix(t, f, townMapId, partyId, ownerCharacterId))
}

// lookupByIndex fetches all doors referenced by a secondary index SET.
func (r *Registry) lookupByIndex(ctx context.Context, t tenant.Model, idx *atlasredis.KeyedSet[string], suffix string) ([]Model, error) {
	members, err := idx.Members(ctx, suffix)
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
		sd, gerr := r.reg.Get(ctx, storeSuffix(t, id))
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
	sd, err := r.reg.Get(ctx, storeSuffix(t, areaDoorId))
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
	_ = r.fieldIdx.Remove(ctx, fieldSuffix(t, m.fld), mk)
	_ = r.ownerIdx.Remove(ctx, ownerSuffix(t, m.ownerCharacterId), mk)
	_ = r.townIdx.Remove(ctx, townSuffix(t, m.fld, m.townMapId, m.partyId, m.ownerCharacterId), mk)
	_ = r.reg.Remove(ctx, storeSuffix(t, areaDoorId))

	return nil
}

// GetAll returns all doors grouped by tenant.
func (r *Registry) GetAll(ctx context.Context) (map[tenant.Model][]Model, error) {
	result := make(map[tenant.Model][]Model)
	all, err := r.reg.GetAll(ctx)
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

// Clear removes all doors and all index entries (useful in tests).
func (r *Registry) Clear(ctx context.Context) {
	_, _ = r.reg.Clear(ctx)
	_, _ = r.fieldIdx.ClearAll(ctx)
	_, _ = r.ownerIdx.ClearAll(ctx)
	_, _ = r.townIdx.ClearAll(ctx)
}
