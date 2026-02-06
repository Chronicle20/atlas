package instance

import (
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

// RouteModel is the domain model for an instance transport route
type RouteModel struct {
	id               uuid.UUID
	name             string
	startMapId       _map.Id
	transitMapId     _map.Id
	destinationMapId _map.Id
	capacity         uint32
	boardingWindow   time.Duration
	travelDuration   time.Duration
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

func (m RouteModel) TransitMapId() _map.Id {
	return m.transitMapId
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
