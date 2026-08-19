package transport

import (
	"time"

	"github.com/google/uuid"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// voyageNamespace scopes voyage-id derivation. Generated once and frozen: it is
// part of the wire contract, because atlas-events matches VOYAGE_ARRIVED to
// VOYAGE_DEPARTED by id equality alone. Changing it orphans every in-flight
// occurrence.
var voyageNamespace = uuid.MustParse("6f3a1b2c-9d4e-4a71-8c55-0b7f2d9e4a10")

// VoyageId derives the durable identity of one trip of one route on one day
// (FR-V1, FR-V5). It is a pure function of facts the service already holds, not
// stored state, which is what makes it survive an atlas-transports restart, a
// Redis flush of the route registry, and two replicas deriving independently
// (design §7.1). departedAt is truncated to the calendar day in its own
// location — ComputeSchedule emits at most one row per tripId per day and
// Evaluate selects at most one in-transit trip, so the day is a sufficient
// discriminator.
func VoyageId(t tenant.Model, routeId uuid.UUID, tripId uuid.UUID, departedAt time.Time) uuid.UUID {
	key := t.Id().String() + "|" + routeId.String() + "|" + tripId.String() + "|" + departedAt.Format("2006-01-02")
	return uuid.NewSHA1(voyageNamespace, []byte(key))
}
