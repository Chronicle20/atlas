package instance

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

type transportInstanceJSON struct {
	InstanceId    uuid.UUID     `json:"instanceId"`
	RouteId       uuid.UUID     `json:"routeId"`
	TenantId      uuid.UUID     `json:"tenantId"`
	State         InstanceState `json:"state"`
	BoardingUntil time.Time     `json:"boardingUntil"`
	ArrivalAt     time.Time     `json:"arrivalAt"`
	CreatedAt     time.Time     `json:"createdAt"`
}

func toJSON(inst TransportInstance) transportInstanceJSON {
	return transportInstanceJSON{
		InstanceId:    inst.instanceId,
		RouteId:       inst.routeId,
		TenantId:      inst.tenantId,
		State:         inst.state,
		BoardingUntil: inst.boardingUntil,
		ArrivalAt:     inst.arrivalAt,
		CreatedAt:     inst.createdAt,
	}
}

func fromJSON(j transportInstanceJSON) TransportInstance {
	return TransportInstance{
		instanceId:    j.InstanceId,
		routeId:       j.RouteId,
		tenantId:      j.TenantId,
		state:         j.State,
		boardingUntil: j.BoardingUntil,
		arrivalAt:     j.ArrivalAt,
		createdAt:     j.CreatedAt,
		characters:    make([]CharacterEntry, 0),
	}
}

// tenantFromId reconstructs a region-less tenant.Model from a bare UUID, used
// to scope the per-route SET. The region/version segments are unused by routing
// and repopulate; this only affects the (prefixed) Redis key shape.
func tenantFromId(id uuid.UUID) (tenant.Model, bool) {
	t, err := tenant.Create(id, "", 0, 0)
	if err != nil {
		return tenant.Model{}, false
	}
	return t, true
}

type InstanceRegistry struct {
	all    *atlas.Set
	meta   *atlas.Registry[uuid.UUID, transportInstanceJSON]
	chars  *atlas.KeyedHash[uuid.UUID]
	routes *atlas.TenantKeyedSet[uuid.UUID]
}

var instanceRegistry *InstanceRegistry

func InitInstanceRegistry(client *goredis.Client) {
	instanceRegistry = &InstanceRegistry{
		all: atlas.NewSet(client, "transport:instances"),
		meta: atlas.NewRegistry[uuid.UUID, transportInstanceJSON](client, "transport:instance", func(id uuid.UUID) string {
			return id.String()
		}),
		chars: atlas.NewKeyedHash[uuid.UUID](client, "transport:instance:chars", func(id uuid.UUID) string {
			return id.String()
		}),
		routes: atlas.NewTenantKeyedSet[uuid.UUID](client, "transport:route", func(id uuid.UUID) string {
			return id.String()
		}),
	}
}

func getInstanceRegistry() *InstanceRegistry {
	return instanceRegistry
}

func (r *InstanceRegistry) storeMetadata(inst TransportInstance) {
	ctx := context.Background()
	_ = r.meta.Put(ctx, inst.instanceId, toJSON(inst))
	_ = r.all.Add(ctx, inst.instanceId.String())
	if t, ok := tenantFromId(inst.tenantId); ok {
		_ = r.routes.Add(ctx, t, inst.routeId, inst.instanceId.String())
	}
}

func (r *InstanceRegistry) loadMetadata(id uuid.UUID) (TransportInstance, bool) {
	j, err := r.meta.Get(context.Background(), id)
	if err != nil {
		return TransportInstance{}, false
	}
	return fromJSON(j), true
}

func (r *InstanceRegistry) loadCharacters(id uuid.UUID) []CharacterEntry {
	charMap, err := r.chars.GetAll(context.Background(), id)
	if err != nil {
		return nil
	}
	chars := make([]CharacterEntry, 0, len(charMap))
	for _, v := range charMap {
		var entry CharacterEntry
		if err := json.Unmarshal([]byte(v), &entry); err == nil {
			chars = append(chars, entry)
		}
	}
	return chars
}

func (r *InstanceRegistry) loadInstance(id uuid.UUID) (TransportInstance, bool) {
	inst, ok := r.loadMetadata(id)
	if !ok {
		return TransportInstance{}, false
	}
	chars := r.loadCharacters(id)
	if chars != nil {
		inst.characters = chars
	}
	return inst, true
}

// FindOrCreateInstance finds an existing boarding instance with room and an open window,
// or creates a new one with a fresh UUID.
func (r *InstanceRegistry) FindOrCreateInstance(tenantId uuid.UUID, route RouteModel, now time.Time) TransportInstance {
	ctx := context.Background()
	if t, ok := tenantFromId(tenantId); ok {
		members, err := r.routes.Members(ctx, t, route.Id())
		if err == nil {
			for _, member := range members {
				id, err := uuid.Parse(member)
				if err != nil {
					continue
				}
				inst, ok := r.loadMetadata(id)
				if !ok {
					continue
				}
				if inst.state != Boarding || !now.Before(inst.boardingUntil) {
					continue
				}
				count, err := r.chars.Len(ctx, id)
				if err != nil {
					continue
				}
				if uint32(count) < route.Capacity() {
					return inst
				}
			}
		}
	}

	// Create new instance
	instanceId := uuid.New()
	boardingUntil := now.Add(route.BoardingWindow())
	arrivalAt := boardingUntil.Add(route.TravelDuration())
	inst := NewTransportInstance(instanceId, route.Id(), tenantId, boardingUntil, arrivalAt)
	r.storeMetadata(inst)
	return inst
}

// AddCharacter adds a character to an instance.
// Returns whether the instance was found and the new character count.
func (r *InstanceRegistry) AddCharacter(instanceId uuid.UUID, entry CharacterEntry) (bool, int) {
	ctx := context.Background()
	if _, ok := r.loadMetadata(instanceId); !ok {
		return false, 0
	}
	data, _ := json.Marshal(entry)
	_ = r.chars.Set(ctx, instanceId, strconv.FormatUint(uint64(entry.CharacterId), 10), string(data))
	count, _ := r.chars.Len(ctx, instanceId)
	return true, int(count)
}

// RemoveCharacter removes a character from an instance.
// Returns true if the instance is now empty.
func (r *InstanceRegistry) RemoveCharacter(instanceId uuid.UUID, characterId uint32) bool {
	ctx := context.Background()
	_ = r.chars.Del(ctx, instanceId, strconv.FormatUint(uint64(characterId), 10))
	count, err := r.chars.Len(ctx, instanceId)
	if err != nil {
		return false
	}
	return count == 0
}

// TransitionToInTransit transitions an instance from Boarding to InTransit.
func (r *InstanceRegistry) TransitionToInTransit(instanceId uuid.UUID) bool {
	inst, ok := r.loadMetadata(instanceId)
	if !ok || inst.state != Boarding {
		return false
	}
	inst.state = InTransit
	r.storeMetadata(inst)
	return true
}

// ReleaseInstance removes an instance from all indices and deletes its data.
func (r *InstanceRegistry) ReleaseInstance(instanceId uuid.UUID) {
	ctx := context.Background()
	inst, ok := r.loadMetadata(instanceId)
	if !ok {
		return
	}
	if t, ok := tenantFromId(inst.tenantId); ok {
		_ = r.routes.Remove(ctx, t, inst.routeId, instanceId.String())
	}
	_ = r.all.Remove(ctx, instanceId.String())
	_ = r.meta.Remove(ctx, instanceId)
	_ = r.chars.DeleteKey(ctx, instanceId)
}

// GetInstance returns the instance for a given instance ID.
func (r *InstanceRegistry) GetInstance(instanceId uuid.UUID) (TransportInstance, bool) {
	return r.loadInstance(instanceId)
}

// GetExpiredBoarding returns instances past their boardingUntil still in Boarding state.
func (r *InstanceRegistry) GetExpiredBoarding(now time.Time) []TransportInstance {
	return r.filterInstances(func(inst TransportInstance) bool {
		return inst.state == Boarding && now.After(inst.boardingUntil)
	})
}

// GetExpiredTransit returns instances past their arrivalAt.
func (r *InstanceRegistry) GetExpiredTransit(now time.Time) []TransportInstance {
	return r.filterInstances(func(inst TransportInstance) bool {
		return inst.state == InTransit && now.After(inst.arrivalAt)
	})
}

// GetAllActive returns all active instances.
func (r *InstanceRegistry) GetAllActive() []TransportInstance {
	return r.filterInstances(func(inst TransportInstance) bool { return true })
}

// GetStuck returns instances exceeding the given max lifetime.
func (r *InstanceRegistry) GetStuck(now time.Time, maxLifetime time.Duration) []TransportInstance {
	return r.filterInstances(func(inst TransportInstance) bool {
		return now.Sub(inst.createdAt) > maxLifetime
	})
}

// GetInstancesByRoute returns all instances for a given tenant and route.
func (r *InstanceRegistry) GetInstancesByRoute(tenantId, routeId uuid.UUID) []TransportInstance {
	t, ok := tenantFromId(tenantId)
	if !ok {
		return nil
	}
	members, err := r.routes.Members(context.Background(), t, routeId)
	if err != nil {
		return nil
	}
	var result []TransportInstance
	for _, member := range members {
		id, err := uuid.Parse(member)
		if err != nil {
			continue
		}
		inst, ok := r.loadInstance(id)
		if !ok {
			continue
		}
		result = append(result, inst)
	}
	return result
}

func (r *InstanceRegistry) filterInstances(predicate func(TransportInstance) bool) []TransportInstance {
	members, err := r.all.Members(context.Background())
	if err != nil {
		return nil
	}
	var result []TransportInstance
	for _, member := range members {
		id, err := uuid.Parse(member)
		if err != nil {
			continue
		}
		inst, ok := r.loadInstance(id)
		if !ok {
			continue
		}
		if predicate(inst) {
			result = append(result, inst)
		}
	}
	return result
}
