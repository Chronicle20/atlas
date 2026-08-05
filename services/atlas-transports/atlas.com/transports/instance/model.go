package instance

import (
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// RouteModel is the domain model for an instance transport route
type RouteModel struct {
	id                uuid.UUID
	name              string
	startMapId        _map.Id
	transitMapIds     []_map.Id
	destinationMapId  _map.Id
	capacity          uint32
	boardingWindow    time.Duration
	travelDuration    time.Duration
	transitMessage    string
	effectItemIds     []item.Id
	forcedReturnMapId _map.Id
}

func (m RouteModel) Id() uuid.UUID {
	return m.id
}

func (m RouteModel) Name() string {
	return m.name
}

func (m RouteModel) StartMapId() _map.Id {
	return m.startMapId
}

func (m RouteModel) TransitMapIds() []_map.Id {
	result := make([]_map.Id, len(m.transitMapIds))
	copy(result, m.transitMapIds)
	return result
}

func (m RouteModel) HasTransitMap(mapId _map.Id) bool {
	for _, id := range m.transitMapIds {
		if id == mapId {
			return true
		}
	}
	return false
}

func (m RouteModel) DestinationMapId() _map.Id {
	return m.destinationMapId
}

func (m RouteModel) Capacity() uint32 {
	return m.capacity
}

func (m RouteModel) BoardingWindow() time.Duration {
	return m.boardingWindow
}

func (m RouteModel) TravelDuration() time.Duration {
	return m.travelDuration
}

func (m RouteModel) TransitMessage() string {
	return m.transitMessage
}

// EffectItemIds are the consumable item ids whose effects this route applies
// on boarding and cancels on every terminal path. Empty means the route
// applies no effects. The copy matches TransitMapIds: reads never hand out
// the backing array, which is what makes the model immutable in practice.
func (m RouteModel) EffectItemIds() []item.Id {
	result := make([]item.Id, len(m.effectItemIds))
	copy(result, m.effectItemIds)
	return result
}

// ForcedReturnMapId is the map TickArrival warps to when the travel timer
// expires. Zero means "not set" — deliver to destinationMapId instead. It
// mirrors the client's own Map.wz info/forcedReturn node, which is only
// meaningful on maps that also carry a timeLimit.
func (m RouteModel) ForcedReturnMapId() _map.Id {
	return m.forcedReturnMapId
}

func (m RouteModel) MaxLifetime() time.Duration {
	return 2 * (m.boardingWindow + m.travelDuration)
}

// CharacterEntry tracks a character and their field context within an instance
type CharacterEntry struct {
	CharacterId uint32
	WorldId     world.Id
	ChannelId   channel.Id
}

// TransportInstance represents an active instance of a transport route
type TransportInstance struct {
	instanceId    uuid.UUID
	routeId       uuid.UUID
	tenantId      uuid.UUID
	characters    []CharacterEntry
	state         InstanceState
	boardingUntil time.Time
	arrivalAt     time.Time
	createdAt     time.Time
}

func NewTransportInstance(instanceId uuid.UUID, routeId uuid.UUID, tenantId uuid.UUID, boardingUntil time.Time, arrivalAt time.Time) TransportInstance {
	return TransportInstance{
		instanceId:    instanceId,
		routeId:       routeId,
		tenantId:      tenantId,
		characters:    make([]CharacterEntry, 0),
		state:         Boarding,
		boardingUntil: boardingUntil,
		arrivalAt:     arrivalAt,
		createdAt:     time.Now(),
	}
}

func (i TransportInstance) InstanceId() uuid.UUID {
	return i.instanceId
}

func (i TransportInstance) RouteId() uuid.UUID {
	return i.routeId
}

func (i TransportInstance) TenantId() uuid.UUID {
	return i.tenantId
}

func (i TransportInstance) Characters() []CharacterEntry {
	result := make([]CharacterEntry, len(i.characters))
	copy(result, i.characters)
	return result
}

func (i TransportInstance) CharacterCount() int {
	return len(i.characters)
}

func (i TransportInstance) State() InstanceState {
	return i.state
}

func (i TransportInstance) BoardingUntil() time.Time {
	return i.boardingUntil
}

func (i TransportInstance) ArrivalAt() time.Time {
	return i.arrivalAt
}

func (i TransportInstance) CreatedAt() time.Time {
	return i.createdAt
}

func (i TransportInstance) HasCharacter(characterId uint32) bool {
	for _, c := range i.characters {
		if c.CharacterId == characterId {
			return true
		}
	}
	return false
}
