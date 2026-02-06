package config

import (
	"atlas-transports/instance"
	"time"

	_map "github.com/Chronicle20/atlas-constants/map"
)

type InstanceRouteRestModel struct {
	Id                    string `json:"-"`
	Name                  string `json:"name"`
	StartMapId            _map.Id `json:"startMapId"`
	TransitMapId          _map.Id `json:"transitMapId"`
	DestinationMapId      _map.Id `json:"destinationMapId"`
	Capacity              uint32  `json:"capacity"`
	BoardingWindowSeconds uint32  `json:"boardingWindowSeconds"`
	TravelDurationSeconds uint32  `json:"travelDurationSeconds"`
	TransitMessage        string  `json:"transitMessage"`
}

func (r InstanceRouteRestModel) GetID() string {
	return r.Id
}

func (r *InstanceRouteRestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

func (r InstanceRouteRestModel) GetName() string {
	return "instance-routes"
}

func ExtractRoute(r InstanceRouteRestModel) (instance.RouteModel, error) {
	return instance.NewRouteBuilder(r.Name).
		SetStartMapId(r.StartMapId).
		SetTransitMapId(r.TransitMapId).
		SetDestinationMapId(r.DestinationMapId).
		SetCapacity(r.Capacity).
		SetBoardingWindow(time.Duration(r.BoardingWindowSeconds) * time.Second).
		SetTravelDuration(time.Duration(r.TravelDurationSeconds) * time.Second).
		SetTransitMessage(r.TransitMessage).
		Build()
}
