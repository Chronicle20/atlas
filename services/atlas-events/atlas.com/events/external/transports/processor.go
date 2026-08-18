package transports

import (
	"context"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

type Processor interface {
	GetRoute(routeId uuid.UUID) (RestModel, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) GetRoute(routeId uuid.UUID) (RestModel, error) {
	return requests.Provider[RestModel, RestModel](p.l, p.ctx)(requestRoute(p.ctx, routeId), Extract)()
}

func Extract(m RestModel) (RestModel, error) {
	return m, nil
}

// VoyageUnderway reports whether the route is still on the given voyage.
// "Still underway" requires BOTH state == in_transit AND the SAME voyage
// identity (design §7.4): a route that has since departed on its NEXT trip
// is in_transit again, and comparing state alone would wrongly report our
// voyage as ongoing.
func VoyageUnderway(rm RestModel, voyageId uuid.UUID) bool {
	if rm.State != "in_transit" {
		return false
	}
	return rm.VoyageID == voyageId.String()
}
