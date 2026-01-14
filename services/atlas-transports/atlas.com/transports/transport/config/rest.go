package config

import (
	"atlas-transports/transport"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/google/uuid"
	"time"
)

// RouteRestModel is the JSON:API resource for routes
type RouteRestModel struct {
	Id                     uuid.UUID     `json:"-"`
	Name                   string        `json:"name"`
	StartMapId             _map.Id       `json:"startMapId"`
	StagingMapId           _map.Id       `json:"stagingMapId"`
	EnRouteMapIds          []_map.Id     `json:"enRouteMapIds"`
	DestinationMapId       _map.Id       `json:"destinationMapId"`
	ObservationMapId       _map.Id       `json:"observationMapId"`
	BoardingWindowDuration time.Duration `json:"boardingWindowDuration"`
	PreDepartureDuration   time.Duration `json:"preDepartureDuration"`
	TravelDuration         time.Duration `json:"travelDuration"`
	CycleInterval          time.Duration `json:"cycleInterval"`
}

// GetID returns the resource ID
func (r RouteRestModel) GetID() string {
	return r.Id.String()
}

// SetID sets the resource ID
func (r *RouteRestModel) SetID(idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return err
	}
	r.Id = id
	return nil
}

// GetName returns the resource name
func (r RouteRestModel) GetName() string {
	return "routes"
}

// ExtractRoute converts a RouteRestModel to a transport.Model
func ExtractRoute(r RouteRestModel) (transport.Model, error) {
	builder := transport.NewBuilder(r.Name).
		SetId(r.Id).
		SetStartMapId(r.StartMapId).
		SetStagingMapId(r.StagingMapId).
		SetDestinationMapId(r.DestinationMapId).
		SetObservationMapId(r.ObservationMapId).
		SetBoardingWindowDuration(r.BoardingWindowDuration * time.Minute).
		SetPreDepartureDuration(r.PreDepartureDuration * time.Minute).
		SetTravelDuration(r.TravelDuration * time.Minute).
		SetCycleInterval(r.CycleInterval * time.Minute)

	for _, mapId := range r.EnRouteMapIds {
		builder.AddEnRouteMapId(mapId)
	}

	return builder.Build()
}

// VesselRestModel is the JSON:API resource for vessels
type VesselRestModel struct {
	Id              uuid.UUID     `json:"-"`
	Name            string        `json:"name"`
	RouteAID        uuid.UUID     `json:"routeAID"`
	RouteBID        uuid.UUID     `json:"routeBID"`
	TurnaroundDelay time.Duration `json:"turnaroundDelay"`
}

// GetID returns the resource ID
func (v VesselRestModel) GetID() string {
	return v.Id.String()
}

// SetID sets the resource ID
func (v *VesselRestModel) SetID(idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return err
	}
	v.Id = id
	return nil
}

// GetName returns the resource name
func (v VesselRestModel) GetName() string {
	return "vessels"
}

// ExtractVessel converts a VesselRestModel to a transport.SharedVesselModel
func ExtractVessel(v VesselRestModel) (transport.SharedVesselModel, error) {
	return transport.NewSharedVesselBuilder().
		SetId(v.Id).
		SetName(v.Name).
		SetRouteAID(v.RouteAID).
		SetRouteBID(v.RouteBID).
		SetTurnaroundDelay(v.TurnaroundDelay * time.Second).
		Build()
}
