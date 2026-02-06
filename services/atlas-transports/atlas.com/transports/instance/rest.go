package instance

import (
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

type RouteRestModel struct {
	ID               uuid.UUID     `json:"-"`
	Name             string        `json:"name"`
	StartMapId       _map.Id       `json:"startMapId"`
	TransitMapId     _map.Id       `json:"transitMapId"`
	DestinationMapId _map.Id       `json:"destinationMapId"`
	Capacity         uint32        `json:"capacity"`
	BoardingWindow   time.Duration `json:"boardingWindow"`
	TravelDuration   time.Duration `json:"travelDuration"`
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
		ID:               m.Id(),
		Name:             m.Name(),
		StartMapId:       m.StartMapId(),
		TransitMapId:     m.TransitMapId(),
		DestinationMapId: m.DestinationMapId(),
		Capacity:         m.Capacity(),
		BoardingWindow:   m.BoardingWindow(),
		TravelDuration:   m.TravelDuration(),
	}, nil
}

type InstanceStatusRestModel struct {
	ID            uuid.UUID `json:"-"`
	RouteId       uuid.UUID `json:"routeId"`
	State         string    `json:"state"`
	Characters    int       `json:"characters"`
	BoardingUntil string    `json:"boardingUntil"`
	ArrivalAt     string    `json:"arrivalAt"`
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

func TransformInstanceStatus(inst *TransportInstance) (InstanceStatusRestModel, error) {
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
