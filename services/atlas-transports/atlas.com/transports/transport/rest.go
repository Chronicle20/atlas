package transport

import (
	"time"
)

// RestModel is the JSON:API resource for a transport route
type RestModel struct {
	ID                string        `json:"-"`
	Name              string        `json:"name"`
	StartMapID        uint32        `json:"startMapId"`
	StagingMapID      uint32        `json:"stagingMapId"`
	EnRouteMapID      uint32        `json:"enRouteMapId"`
	DestinationMapID  uint32        `json:"destinationMapId"`
	CycleInterval     time.Duration `json:"cycleInterval"`
}

// GetID returns the resource ID
func (r RestModel) GetID() string {
	return r.ID
}

// SetID sets the resource ID
func (r *RestModel) SetID(id string) error {
	r.ID = id
	return nil
}

// GetName returns the resource name
func (r RestModel) GetName() string {
	return "route"
}

// Transform converts a Model to a RestModel
func Transform(m Model) (RestModel, error) {
	return RestModel{
		ID:                m.Id(),
		Name:              m.Name(),
		StartMapID:        m.StartMapID(),
		StagingMapID:      m.StagingMapID(),
		EnRouteMapID:      m.EnRouteMapID(),
		DestinationMapID:  m.DestinationMapID(),
		CycleInterval:     m.CycleInterval(),
	}, nil
}

// RouteStateRestModel is the JSON:API resource for a route state
type RouteStateRestModel struct {
	ID            string    `json:"-"`
	Status        string    `json:"status"`
	NextDeparture time.Time `json:"nextDeparture"`
	BoardingEnds  time.Time `json:"boardingEnds"`
}

// GetID returns the resource ID
func (r RouteStateRestModel) GetID() string {
	return r.ID
}

// SetID sets the resource ID
func (r *RouteStateRestModel) SetID(id string) error {
	r.ID = id
	return nil
}

// GetName returns the resource name
func (r RouteStateRestModel) GetName() string {
	return "route-state"
}

// TransformState converts a RouteStateModel to a RouteStateRestModel
func TransformState(m RouteStateModel) (RouteStateRestModel, error) {
	return RouteStateRestModel{
		ID:            m.RouteID(),
		Status:        string(m.Status()),
		NextDeparture: m.NextDeparture(),
		BoardingEnds:  m.BoardingEnds(),
	}, nil
}

// TripScheduleRestModel is the JSON:API resource for a trip schedule
type TripScheduleRestModel struct {
	ID            string    `json:"-"`
	BoardingOpen  time.Time `json:"boardingOpen"`
	BoardingClosed time.Time `json:"boardingClosed"`
	Departure     time.Time `json:"departure"`
	Arrival       time.Time `json:"arrival"`
}

// GetID returns the resource ID
func (r TripScheduleRestModel) GetID() string {
	return r.ID
}

// SetID sets the resource ID
func (r *TripScheduleRestModel) SetID(id string) error {
	r.ID = id
	return nil
}

// GetName returns the resource name
func (r TripScheduleRestModel) GetName() string {
	return "trip-schedule"
}

// TransformSchedule converts a TripScheduleModel to a TripScheduleRestModel
func TransformSchedule(m TripScheduleModel) (TripScheduleRestModel, error) {
	return TripScheduleRestModel{
		ID:             m.TripID(),
		BoardingOpen:   m.BoardingOpen(),
		BoardingClosed: m.BoardingClosed(),
		Departure:      m.Departure(),
		Arrival:        m.Arrival(),
	}, nil
}
