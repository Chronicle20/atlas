package transport

import (
	"encoding/json"
	"time"

	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/google/uuid"
)

type tripScheduleJSON struct {
	TripId         uuid.UUID `json:"tripId"`
	RouteId        uuid.UUID `json:"routeId"`
	BoardingOpen   time.Time `json:"boardingOpen"`
	BoardingClosed time.Time `json:"boardingClosed"`
	Departure      time.Time `json:"departure"`
	Arrival        time.Time `json:"arrival"`
}

func (m TripScheduleModel) MarshalJSON() ([]byte, error) {
	return json.Marshal(tripScheduleJSON{
		TripId:         m.tripId,
		RouteId:        m.routeId,
		BoardingOpen:   m.boardingOpen,
		BoardingClosed: m.boardingClosed,
		Departure:      m.departure,
		Arrival:        m.arrival,
	})
}

func (m *TripScheduleModel) UnmarshalJSON(data []byte) error {
	var j tripScheduleJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	m.tripId = j.TripId
	m.routeId = j.RouteId
	m.boardingOpen = j.BoardingOpen
	m.boardingClosed = j.BoardingClosed
	m.departure = j.Departure
	m.arrival = j.Arrival
	return nil
}

type modelJSON struct {
	Id                     uuid.UUID           `json:"id"`
	Name                   string              `json:"name"`
	StartMapId             _map.Id             `json:"startMapId"`
	StagingMapId           _map.Id             `json:"stagingMapId"`
	EnRouteMapIds          []_map.Id           `json:"enRouteMapIds"`
	DestinationMapId       _map.Id             `json:"destinationMapId"`
	ObservationMapId       _map.Id             `json:"observationMapId"`
	State                  RouteState          `json:"state"`
	Schedule               []TripScheduleModel `json:"schedule"`
	BoardingWindowDuration time.Duration       `json:"boardingWindowDuration"`
	PreDepartureDuration   time.Duration       `json:"preDepartureDuration"`
	TravelDuration         time.Duration       `json:"travelDuration"`
	CycleInterval          time.Duration       `json:"cycleInterval"`
}

func (m Model) MarshalJSON() ([]byte, error) {
	return json.Marshal(modelJSON{
		Id:                     m.id,
		Name:                   m.name,
		StartMapId:             m.startMapId,
		StagingMapId:           m.stagingMapId,
		EnRouteMapIds:          m.enRouteMapIds,
		DestinationMapId:       m.destinationMapId,
		ObservationMapId:       m.observationMapId,
		State:                  m.state,
		Schedule:               m.schedule,
		BoardingWindowDuration: m.boardingWindowDuration,
		PreDepartureDuration:   m.preDepartureDuration,
		TravelDuration:         m.travelDuration,
		CycleInterval:          m.cycleInterval,
	})
}

func (m *Model) UnmarshalJSON(data []byte) error {
	var j modelJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	m.id = j.Id
	m.name = j.Name
	m.startMapId = j.StartMapId
	m.stagingMapId = j.StagingMapId
	m.enRouteMapIds = j.EnRouteMapIds
	m.destinationMapId = j.DestinationMapId
	m.observationMapId = j.ObservationMapId
	m.state = j.State
	m.schedule = j.Schedule
	m.boardingWindowDuration = j.BoardingWindowDuration
	m.preDepartureDuration = j.PreDepartureDuration
	m.travelDuration = j.TravelDuration
	m.cycleInterval = j.CycleInterval
	return nil
}
