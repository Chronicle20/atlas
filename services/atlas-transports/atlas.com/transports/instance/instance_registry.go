package instance

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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

func marshalInstanceMetadata(inst TransportInstance) ([]byte, error) {
	return json.Marshal(transportInstanceJSON{
		InstanceId:    inst.instanceId,
		RouteId:       inst.routeId,
		TenantId:      inst.tenantId,
		State:         inst.state,
		BoardingUntil: inst.boardingUntil,
		ArrivalAt:     inst.arrivalAt,
		CreatedAt:     inst.createdAt,
	})
}

func unmarshalInstanceMetadata(data []byte) (TransportInstance, error) {
	var j transportInstanceJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return TransportInstance{}, err
	}
	return TransportInstance{
		instanceId:    j.InstanceId,
		routeId:       j.RouteId,
		tenantId:      j.TenantId,
		state:         j.State,
		boardingUntil: j.BoardingUntil,
		arrivalAt:     j.ArrivalAt,
		createdAt:     j.CreatedAt,
		characters:    make([]CharacterEntry, 0),
	}, nil
}

type InstanceRegistry struct {
	client *goredis.Client
}

var instanceRegistry *InstanceRegistry

func InitInstanceRegistry(client *goredis.Client) {
	instanceRegistry = &InstanceRegistry{client: client}
}

func getInstanceRegistry() *InstanceRegistry {
	return instanceRegistry
}

const allInstancesKey = "transport:instances"

func instanceMetaKey(id uuid.UUID) string {
	return fmt.Sprintf("transport:instance:%s", id.String())
}

func instanceCharsKey(id uuid.UUID) string {
	return fmt.Sprintf("transport:instance:%s:chars", id.String())
}

func instanceRouteSetKey(tenantId, routeId uuid.UUID) string {
	return fmt.Sprintf("transport:route:%s:%s", tenantId.String(), routeId.String())
}

func (r *InstanceRegistry) storeMetadata(inst TransportInstance) {
	ctx := context.Background()
	data, err := marshalInstanceMetadata(inst)
	if err != nil {
		return
	}
	_ = r.client.Set(ctx, instanceMetaKey(inst.instanceId), data, 0).Err()
	_ = r.client.SAdd(ctx, allInstancesKey, inst.instanceId.String()).Err()
	_ = r.client.SAdd(ctx, instanceRouteSetKey(inst.tenantId, inst.routeId), inst.instanceId.String()).Err()
}

func (r *InstanceRegistry) loadMetadata(id uuid.UUID) (TransportInstance, bool) {
	ctx := context.Background()
	data, err := r.client.Get(ctx, instanceMetaKey(id)).Bytes()
	if err != nil {
		return TransportInstance{}, false
	}
	inst, err := unmarshalInstanceMetadata(data)
	if err != nil {
		return TransportInstance{}, false
	}
	return inst, true
}

func (r *InstanceRegistry) loadCharacters(id uuid.UUID) []CharacterEntry {
	ctx := context.Background()
	charMap, err := r.client.HGetAll(ctx, instanceCharsKey(id)).Result()
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
	key := instanceRouteSetKey(tenantId, route.Id())

	members, err := r.client.SMembers(ctx, key).Result()
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
			count, err := r.client.HLen(ctx, instanceCharsKey(id)).Result()
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
	inst := NewTransportInstance(instanceId, route.Id(), tenantId, boardingUntil, arrivalAt)
	r.storeMetadata(inst)
	return inst
}

// AddCharacter adds a character to an instance.
// Returns whether the instance was found and the new character count.
func (r *InstanceRegistry) AddCharacter(instanceId uuid.UUID, entry CharacterEntry) (bool, int) {
	ctx := context.Background()
	exists, err := r.client.Exists(ctx, instanceMetaKey(instanceId)).Result()
	if err != nil || exists == 0 {
		return false, 0
	}
	data, _ := json.Marshal(entry)
	_ = r.client.HSet(ctx, instanceCharsKey(instanceId), fmt.Sprintf("%d", entry.CharacterId), data).Err()
	count, _ := r.client.HLen(ctx, instanceCharsKey(instanceId)).Result()
	return true, int(count)
}

// RemoveCharacter removes a character from an instance.
// Returns true if the instance is now empty.
func (r *InstanceRegistry) RemoveCharacter(instanceId uuid.UUID, characterId uint32) bool {
	ctx := context.Background()
	_ = r.client.HDel(ctx, instanceCharsKey(instanceId), fmt.Sprintf("%d", characterId)).Err()
	count, err := r.client.HLen(ctx, instanceCharsKey(instanceId)).Result()
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
	data, err := marshalInstanceMetadata(inst)
	if err != nil {
		return false
	}
	_ = r.client.Set(context.Background(), instanceMetaKey(instanceId), data, 0).Err()
	return true
}

// ReleaseInstance removes an instance from all indices and deletes its data.
func (r *InstanceRegistry) ReleaseInstance(instanceId uuid.UUID) {
	ctx := context.Background()
	inst, ok := r.loadMetadata(instanceId)
	if !ok {
		return
	}
	_ = r.client.SRem(ctx, instanceRouteSetKey(inst.tenantId, inst.routeId), instanceId.String()).Err()
	_ = r.client.SRem(ctx, allInstancesKey, instanceId.String()).Err()
	_ = r.client.Del(ctx, instanceMetaKey(instanceId)).Err()
	_ = r.client.Del(ctx, instanceCharsKey(instanceId)).Err()
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
	return r.filterInstances(func(inst TransportInstance) bool {
		return true
	})
}

// GetStuck returns instances exceeding the given max lifetime.
func (r *InstanceRegistry) GetStuck(now time.Time, maxLifetime time.Duration) []TransportInstance {
	return r.filterInstances(func(inst TransportInstance) bool {
		return now.Sub(inst.createdAt) > maxLifetime
	})
}

// GetInstancesByRoute returns all instances for a given tenant and route.
func (r *InstanceRegistry) GetInstancesByRoute(tenantId, routeId uuid.UUID) []TransportInstance {
	ctx := context.Background()
	key := instanceRouteSetKey(tenantId, routeId)
	members, err := r.client.SMembers(ctx, key).Result()
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
	ctx := context.Background()
	members, err := r.client.SMembers(ctx, allInstancesKey).Result()
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
