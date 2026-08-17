package instance

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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

type InstanceRegistry struct {
	all    *atlas.TenantSet
	meta   *atlas.TenantRegistry[uuid.UUID, transportInstanceJSON]
	chars  *atlas.TenantKeyedHash[uuid.UUID]
	routes *atlas.TenantKeyedSet[uuid.UUID]
}

var instanceRegistry *InstanceRegistry

func InitInstanceRegistry(client *goredis.Client) {
	instanceRegistry = &InstanceRegistry{
		all: atlas.NewTenantSet(client, "transport:instances"),
		meta: atlas.NewTenantRegistry[uuid.UUID, transportInstanceJSON](client, "transport:instance", func(id uuid.UUID) string {
			return id.String()
		}),
		chars: atlas.NewTenantKeyedHash[uuid.UUID](client, "transport:instance:chars", func(id uuid.UUID) string {
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

// storeMetadata persists inst under the tenant carried by ctx. Every caller
// of storeMetadata (FindOrCreateInstance, TransitionToInTransit) runs inside
// the tenant that owns inst — instances are only ever created, transitioned
// and released within the request/tick context of their own tenant — so the
// real tenant.Model on ctx is always the correct scope, never a
// reconstructed/region-less one.
func (r *InstanceRegistry) storeMetadata(ctx context.Context, inst TransportInstance) {
	t := tenant.MustFromContext(ctx)
	_ = r.meta.Put(ctx, t, inst.instanceId, toJSON(inst))
	_ = r.all.Add(ctx, t, inst.instanceId.String())
	_ = r.routes.Add(ctx, t, inst.routeId, inst.instanceId.String())
}

func (r *InstanceRegistry) loadMetadata(ctx context.Context, id uuid.UUID) (TransportInstance, bool) {
	t := tenant.MustFromContext(ctx)
	j, err := r.meta.Get(ctx, t, id)
	if err != nil {
		return TransportInstance{}, false
	}
	return fromJSON(j), true
}

func (r *InstanceRegistry) loadCharacters(ctx context.Context, id uuid.UUID) []CharacterEntry {
	t := tenant.MustFromContext(ctx)
	charMap, err := r.chars.GetAll(ctx, t, id)
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

func (r *InstanceRegistry) loadInstance(ctx context.Context, id uuid.UUID) (TransportInstance, bool) {
	inst, ok := r.loadMetadata(ctx, id)
	if !ok {
		return TransportInstance{}, false
	}
	chars := r.loadCharacters(ctx, id)
	if chars != nil {
		inst.characters = chars
	}
	return inst, true
}

// FindOrCreateInstance finds an existing boarding instance with room and an open window,
// or creates a new one with a fresh UUID, scoped to the tenant carried by ctx.
func (r *InstanceRegistry) FindOrCreateInstance(ctx context.Context, route RouteModel, now time.Time) TransportInstance {
	t := tenant.MustFromContext(ctx)

	members, err := r.routes.Members(ctx, t, route.Id())
	if err == nil {
		for _, member := range members {
			id, err := uuid.Parse(member)
			if err != nil {
				continue
			}
			inst, ok := r.loadMetadata(ctx, id)
			if !ok {
				continue
			}
			if inst.state != Boarding || !now.Before(inst.boardingUntil) {
				continue
			}
			count, err := r.chars.Len(ctx, t, id)
			if err != nil {
				continue
			}
			if uint32(count) < route.Capacity() {
				return inst
			}
		}
	}

	// Create new instance
	instanceId := uuid.New()
	boardingUntil := now.Add(route.BoardingWindow())
	arrivalAt := boardingUntil.Add(route.TravelDuration())
	inst := NewTransportInstance(instanceId, route.Id(), t.Id(), boardingUntil, arrivalAt)
	r.storeMetadata(ctx, inst)
	return inst
}

// AddCharacter adds a character to an instance.
// Returns whether the instance was found and the new character count.
func (r *InstanceRegistry) AddCharacter(ctx context.Context, instanceId uuid.UUID, entry CharacterEntry) (bool, int) {
	t := tenant.MustFromContext(ctx)
	if _, ok := r.loadMetadata(ctx, instanceId); !ok {
		return false, 0
	}
	data, _ := json.Marshal(entry)
	_ = r.chars.Set(ctx, t, instanceId, strconv.FormatUint(uint64(entry.CharacterId), 10), string(data))
	count, _ := r.chars.Len(ctx, t, instanceId)
	return true, int(count)
}

// RemoveCharacter removes a character from an instance.
// Returns true if the instance is now empty.
func (r *InstanceRegistry) RemoveCharacter(ctx context.Context, instanceId uuid.UUID, characterId uint32) bool {
	t := tenant.MustFromContext(ctx)
	_ = r.chars.Del(ctx, t, instanceId, strconv.FormatUint(uint64(characterId), 10))
	count, err := r.chars.Len(ctx, t, instanceId)
	if err != nil {
		return false
	}
	return count == 0
}

// TransitionToInTransit transitions an instance from Boarding to InTransit.
func (r *InstanceRegistry) TransitionToInTransit(ctx context.Context, instanceId uuid.UUID) bool {
	inst, ok := r.loadMetadata(ctx, instanceId)
	if !ok || inst.state != Boarding {
		return false
	}
	inst.state = InTransit
	r.storeMetadata(ctx, inst)
	return true
}

// ReleaseInstance removes an instance from all indices and deletes its data.
func (r *InstanceRegistry) ReleaseInstance(ctx context.Context, instanceId uuid.UUID) {
	t := tenant.MustFromContext(ctx)
	inst, ok := r.loadMetadata(ctx, instanceId)
	if !ok {
		return
	}
	_ = r.routes.Remove(ctx, t, inst.routeId, instanceId.String())
	_ = r.all.Remove(ctx, t, instanceId.String())
	_ = r.meta.Remove(ctx, t, instanceId)
	_ = r.chars.DeleteKey(ctx, t, instanceId)
}

// GetInstance returns the instance for a given instance ID.
func (r *InstanceRegistry) GetInstance(ctx context.Context, instanceId uuid.UUID) (TransportInstance, bool) {
	return r.loadInstance(ctx, instanceId)
}

// GetExpiredBoarding returns instances past their boardingUntil still in Boarding state.
func (r *InstanceRegistry) GetExpiredBoarding(ctx context.Context, now time.Time) []TransportInstance {
	return r.filterInstances(ctx, func(inst TransportInstance) bool {
		return inst.state == Boarding && now.After(inst.boardingUntil)
	})
}

// GetExpiredTransit returns instances past their arrivalAt.
func (r *InstanceRegistry) GetExpiredTransit(ctx context.Context, now time.Time) []TransportInstance {
	return r.filterInstances(ctx, func(inst TransportInstance) bool {
		return inst.state == InTransit && now.After(inst.arrivalAt)
	})
}

// GetAllActive returns all active instances for the tenant carried by ctx.
func (r *InstanceRegistry) GetAllActive(ctx context.Context) []TransportInstance {
	return r.filterInstances(ctx, func(inst TransportInstance) bool { return true })
}

// GetStuck returns instances exceeding the given max lifetime.
func (r *InstanceRegistry) GetStuck(ctx context.Context, now time.Time, maxLifetime time.Duration) []TransportInstance {
	return r.filterInstances(ctx, func(inst TransportInstance) bool {
		return now.Sub(inst.createdAt) > maxLifetime
	})
}

// GetInstancesByRoute returns all instances for the tenant carried by ctx and a given route.
func (r *InstanceRegistry) GetInstancesByRoute(ctx context.Context, routeId uuid.UUID) []TransportInstance {
	t := tenant.MustFromContext(ctx)
	members, err := r.routes.Members(ctx, t, routeId)
	if err != nil {
		return nil
	}
	var result []TransportInstance
	for _, member := range members {
		id, err := uuid.Parse(member)
		if err != nil {
			continue
		}
		inst, ok := r.loadInstance(ctx, id)
		if !ok {
			continue
		}
		result = append(result, inst)
	}
	return result
}

// filterInstances sweeps only the tenant carried by ctx — the per-tenant SET
// (all) makes this naturally tenant-scoped, unlike the former env-global Set.
func (r *InstanceRegistry) filterInstances(ctx context.Context, predicate func(TransportInstance) bool) []TransportInstance {
	t := tenant.MustFromContext(ctx)
	members, err := r.all.Members(ctx, t)
	if err != nil {
		return nil
	}
	var result []TransportInstance
	for _, member := range members {
		id, err := uuid.Parse(member)
		if err != nil {
			continue
		}
		inst, ok := r.loadInstance(ctx, id)
		if !ok {
			continue
		}
		if predicate(inst) {
			result = append(result, inst)
		}
	}
	return result
}
