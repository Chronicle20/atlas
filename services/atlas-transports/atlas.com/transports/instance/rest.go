package instance

import (
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// RouteRestModel is the JSON:API resource for an instance transport route.
//
// BoardingWindow and TravelDuration are time.Duration and therefore serialise
// as integer nanosecond counts. They are retained unchanged for existing
// consumers; new consumers read the unit-explicit *Seconds fields.
type RouteRestModel struct {
	ID                    uuid.UUID     `json:"-"`
	Name                  string        `json:"name"`
	StartMapId            _map.Id       `json:"startMapId"`
	TransitMapIds         []_map.Id     `json:"transitMapIds"`
	DestinationMapId      _map.Id       `json:"destinationMapId"`
	Capacity              uint32        `json:"capacity"`
	BoardingWindow        time.Duration `json:"boardingWindow"`
	TravelDuration        time.Duration `json:"travelDuration"`
	BoardingWindowSeconds uint32        `json:"boardingWindowSeconds"`
	TravelDurationSeconds uint32        `json:"travelDurationSeconds"`
	EffectItemIds         []item.Id     `json:"effectItemIds"`
	ForcedReturnMapId     _map.Id       `json:"forcedReturnMapId"`
}

func (r RouteRestModel) GetID() string {
	return r.ID.String()
}

func (r *RouteRestModel) SetID(idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return err
	}
	r.ID = id
	return nil
}

func (r RouteRestModel) GetName() string {
	return "instance-routes"
}

func TransformRoute(m RouteModel) (RouteRestModel, error) {
	return RouteRestModel{
		ID:                    m.Id(),
		Name:                  m.Name(),
		StartMapId:            m.StartMapId(),
		TransitMapIds:         m.TransitMapIds(),
		DestinationMapId:      m.DestinationMapId(),
		Capacity:              m.Capacity(),
		BoardingWindow:        m.BoardingWindow(),
		TravelDuration:        m.TravelDuration(),
		BoardingWindowSeconds: uint32(m.BoardingWindow().Seconds()),
		TravelDurationSeconds: uint32(m.TravelDuration().Seconds()),
		EffectItemIds:         m.EffectItemIds(),
		ForcedReturnMapId:     m.ForcedReturnMapId(),
	}, nil
}

type InstanceStatusRestModel struct {
	ID            uuid.UUID `json:"-"`
	RouteId       uuid.UUID `json:"routeId"`
	State         string    `json:"state"`
	Characters    int       `json:"characters"`
	BoardingUntil string    `json:"boardingUntil"`
	ArrivalAt     string    `json:"arrivalAt"`
	// CreatedAt is the instance's creation instant. The stuck-timeout sweep
	// force-warps on now - createdAt exceeding the route's MaxLifetime, so a
	// client that wants to warn ahead of that has to compare the same
	// quantity rather than infer it from boardingUntil.
	CreatedAt string `json:"createdAt"`
}

func (r InstanceStatusRestModel) GetID() string {
	return r.ID.String()
}

func (r *InstanceStatusRestModel) SetID(idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return err
	}
	r.ID = id
	return nil
}

func (r InstanceStatusRestModel) GetName() string {
	return "instance-status"
}

func TransformInstanceStatus(inst TransportInstance) (InstanceStatusRestModel, error) {
	stateStr := "boarding"
	if inst.State() == InTransit {
		stateStr = "in_transit"
	}
	return InstanceStatusRestModel{
		ID:            inst.InstanceId(),
		RouteId:       inst.RouteId(),
		State:         stateStr,
		Characters:    inst.CharacterCount(),
		BoardingUntil: inst.BoardingUntil().Format(time.RFC3339),
		ArrivalAt:     inst.ArrivalAt().Format(time.RFC3339),
		CreatedAt:     inst.CreatedAt().Format(time.RFC3339),
	}, nil
}

type StartTransportRestModel struct {
	ID          string     `json:"-"`
	CharacterId uint32     `json:"characterId"`
	WorldId     world.Id   `json:"worldId"`
	ChannelId   channel.Id `json:"channelId"`
}

func (r StartTransportRestModel) GetID() string {
	return r.ID
}

func (r *StartTransportRestModel) SetID(idStr string) error {
	r.ID = idStr
	return nil
}

func (r StartTransportRestModel) GetName() string {
	return "start-transport"
}
