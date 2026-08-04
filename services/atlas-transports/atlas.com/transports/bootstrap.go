package main

import (
	"atlas-transports/instance"
	"atlas-transports/transport"

	"github.com/sirupsen/logrus"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// The four interfaces below are the narrow slices of the config and
// registry processors the reconcilers actually use. Declaring them here
// (rather than depending on the full Processor interfaces) keeps the
// reconcilers unit-testable without hand-writing large fakes.
type scheduledLoader interface {
	LoadConfigurationsForTenant(t tenant.Model) ([]transport.Model, []transport.SharedVesselModel, error)
}

type instanceLoader interface {
	LoadConfigurationsForTenant(t tenant.Model) ([]instance.RouteModel, error)
}

type scheduledRegistry interface {
	ClearTenant() int
	AddTenant(routes []transport.Model, sharedVessels []transport.SharedVesselModel) error
}

type instanceRegistry interface {
	ClearTenant() int
	AddTenant(routes []instance.RouteModel)
}

// reconcileScheduled replaces the tenant's scheduled-route registry
// contents with exactly what configuration says — but only after a
// successful load.
//
// Load-then-clear-then-add is load-bearing, not stylistic. Route ids are
// now derived and therefore stable, so clear-then-add converges the
// registry to exactly the configured set on every restart — which is
// what purges the duplicate entries an earlier uuid.New() per load left
// behind. Clearing before a load that then fails would instead wipe a
// healthy registry, so a load error skips the whole reconcile.
func reconcileScheduled(l logrus.FieldLogger, t tenant.Model, loader scheduledLoader, registry scheduledRegistry) error {
	routes, vessels, err := loader.LoadConfigurationsForTenant(t)
	if err != nil {
		l.WithError(err).Errorf("Failed to load configurations for tenant [%s]; leaving the scheduled route registry untouched.", t.Id())
		return err
	}
	registry.ClearTenant()
	return registry.AddTenant(routes, vessels)
}

// reconcileInstance is reconcileScheduled's twin for instance transports.
func reconcileInstance(l logrus.FieldLogger, t tenant.Model, loader instanceLoader, registry instanceRegistry) error {
	routes, err := loader.LoadConfigurationsForTenant(t)
	if err != nil {
		l.WithError(err).Errorf("Failed to load instance route configurations for tenant [%s]; leaving the instance route registry untouched.", t.Id())
		return err
	}
	registry.ClearTenant()
	registry.AddTenant(routes)
	return nil
}
