package transport

import (
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

// RouteRestModel represents an instance transport route from atlas-transports
type RouteRestModel struct {
	ID   uuid.UUID `json:"-"`
	Name string    `json:"name"`
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

// StartTransportRestModel represents the request body for starting a transport
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
