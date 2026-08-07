package config

import (
	"atlas-transports/instance"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// resourceName is the atlas-tenants configuration resource these routes
// come from. It is one half of the derived-id name, so it must match the
// value atlas-tenants uses in TransformInstanceRoute exactly.
const resourceName = "instance-routes"

type InstanceRouteRestModel struct {
	Id                    string    `json:"-"`
	Uuid                  string    `json:"uuid"`
	Name                  string    `json:"name"`
	StartMapId            _map.Id   `json:"startMapId"`
	TransitMapIds         []_map.Id `json:"transitMapIds"`
	DestinationMapId      _map.Id   `json:"destinationMapId"`
	Capacity              uint32    `json:"capacity"`
	BoardingWindowSeconds uint32    `json:"boardingWindowSeconds"`
	TravelDurationSeconds uint32    `json:"travelDurationSeconds"`
	TransitMessage        string    `json:"transitMessage"`
	// EffectItemIds and ForcedReturnMapId are optional. atlas-tenants omits
	// them for routes that declare neither, which decodes to nil/0 — the
	// "no effects, deliver to destination" default.
	EffectItemIds     []item.Id `json:"effectItemIds"`
	ForcedReturnMapId _map.Id   `json:"forcedReturnMapId"`
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

// ExtractRouteFor builds the mapper requests.DrainProvider applies to each
// fetched route. It is tenant-aware because the route's identity is
// derived from the tenant: without a stable id the builder mints a fresh
// uuid.New() on every load, so each replica and each restart writes
// another full copy into the shared Redis registry.
func ExtractRouteFor(l logrus.FieldLogger, t tenant.Model) func(InstanceRouteRestModel) (instance.RouteModel, error) {
	return func(r InstanceRouteRestModel) (instance.RouteModel, error) {
		return instance.NewRouteBuilder(r.Name).
			SetId(resolveRouteId(l, t, r.Id, r.Uuid)).
			SetStartMapId(r.StartMapId).
			SetTransitMapIds(r.TransitMapIds).
			SetDestinationMapId(r.DestinationMapId).
			SetCapacity(r.Capacity).
			SetBoardingWindow(time.Duration(r.BoardingWindowSeconds) * time.Second).
			SetTravelDuration(time.Duration(r.TravelDurationSeconds) * time.Second).
			SetTransitMessage(r.TransitMessage).
			SetEffectItemIds(r.EffectItemIds).
			SetForcedReturnMapId(r.ForcedReturnMapId).
			Build()
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
		l.Warnf("Instance route [%s] has unparseable uuid [%s] for tenant [%s]; deriving locally.", slug, raw, t.Id())
	} else {
		l.Warnf("Instance route [%s] has no uuid attribute for tenant [%s]; deriving locally.", slug, t.Id())
	}
	return tenant.DerivedId(t.Id(), resourceName, slug)
}
