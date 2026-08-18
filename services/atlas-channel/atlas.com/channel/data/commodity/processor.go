package commodity

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// Processor resolves a cash-shop commodity SERIAL NUMBER (the identifier
// ShopOperationBuy* carry on the wire) to the item TEMPLATE id atlas-data's
// commodity catalog maps it to. Serial numbers and template ids are
// different identifier spaces (docs/tasks/task-227-cash-name-change-world-transfer/derivation.md);
// this is the resolution step between the two, mirroring
// services/atlas-cashshop/atlas.com/cashshop/cashshop/commodity, which hits
// the same atlas-data endpoint for the same reason.
type Processor interface {
	GetById(serialNumber uint32) (RestModel, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) GetById(serialNumber uint32) (RestModel, error) {
	return requests.Provider[RestModel, RestModel](p.l, p.ctx)(requestById(serialNumber), Extract)()
}

func Extract(m RestModel) (RestModel, error) {
	return m, nil
}
