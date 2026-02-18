package instance

import (
	"encoding/json"
	"time"

	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/google/uuid"
)

type routeModelJSON struct {
	Id               uuid.UUID     `json:"id"`
	Name             string        `json:"name"`
	StartMapId       _map.Id       `json:"startMapId"`
	TransitMapIds    []_map.Id     `json:"transitMapIds"`
	DestinationMapId _map.Id       `json:"destinationMapId"`
	Capacity         uint32        `json:"capacity"`
	BoardingWindow   time.Duration `json:"boardingWindow"`
	TravelDuration   time.Duration `json:"travelDuration"`
	TransitMessage   string        `json:"transitMessage"`
}

func (m RouteModel) MarshalJSON() ([]byte, error) {
	return json.Marshal(routeModelJSON{
		Id:               m.id,
		Name:             m.name,
		StartMapId:       m.startMapId,
		TransitMapIds:    m.transitMapIds,
		DestinationMapId: m.destinationMapId,
		Capacity:         m.capacity,
		BoardingWindow:   m.boardingWindow,
		TravelDuration:   m.travelDuration,
		TransitMessage:   m.transitMessage,
	})
}

func (m *RouteModel) UnmarshalJSON(data []byte) error {
	var j routeModelJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	m.id = j.Id
	m.name = j.Name
	m.startMapId = j.StartMapId
	m.transitMapIds = j.TransitMapIds
	m.destinationMapId = j.DestinationMapId
	m.capacity = j.Capacity
	m.boardingWindow = j.BoardingWindow
	m.travelDuration = j.TravelDuration
	m.transitMessage = j.TransitMessage
	return nil
}
