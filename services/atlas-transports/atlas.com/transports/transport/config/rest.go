package config

import (
	"atlas-transports/transport"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// RouteRestModel is the JSON:API resource for routes
type RouteRestModel struct {
	Id                     string        `json:"-"`
	Uuid                   string        `json:"uuid"`
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
	return r.Id
}

// SetID sets the resource ID
func (r *RouteRestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

// GetName returns the resource name
func (r RouteRestModel) GetName() string {
	return "routes"
}

// routeResourceName is the atlas-tenants configuration resource these
// routes come from; it must match TransformRoute's value exactly.
const routeResourceName = "routes"

// ExtractRouteFor builds the mapper requests.DrainProvider applies to
// each fetched route. See the instance/config twin for why identity has
// to be tenant-derived rather than freshly minted.
func ExtractRouteFor(l logrus.FieldLogger, t tenant.Model) func(RouteRestModel) (transport.Model, error) {
	return func(r RouteRestModel) (transport.Model, error) {
		builder := transport.NewBuilder(r.Name).
			SetId(resolveRouteId(l, t, r.Id, r.Uuid)).
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
}

// resolveRouteId prefers the uuid atlas-tenants supplies and otherwise
// derives the identical value locally. The fallback exists for a
// staggered rollout where atlas-transports is up before atlas-tenants
// serves the attribute; because both sides call tenant.DerivedId with the
// same inputs, the two paths can never disagree.
func resolveRouteId(l logrus.FieldLogger, t tenant.Model, slug string, raw string) uuid.UUID {
	if raw != "" {
		if parsed, err := uuid.Parse(raw); err == nil {
			return parsed
		}
		l.Warnf("Route [%s] has unparseable uuid [%s] for tenant [%s]; deriving locally.", slug, raw, t.Id())
	} else {
		l.Warnf("Route [%s] has no uuid attribute for tenant [%s]; deriving locally.", slug, t.Id())
	}
	return tenant.DerivedId(t.Id(), routeResourceName, slug)
}

// VesselRestModel is the JSON:API resource for vessels
type VesselRestModel struct {
	Id              string        `json:"-"`
	Name            string        `json:"name"`
	RouteAID        string        `json:"routeAID"`
	RouteBID        string        `json:"routeBID"`
	TurnaroundDelay time.Duration `json:"turnaroundDelay"`
}

// GetID returns the resource ID
func (v VesselRestModel) GetID() string {
	return v.Id
}

// SetID sets the resource ID
func (v *VesselRestModel) SetID(idStr string) error {
	v.Id = idStr
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
