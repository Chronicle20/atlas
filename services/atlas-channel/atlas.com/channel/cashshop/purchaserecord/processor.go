package purchaserecord

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// Processor interface defines the operations for purchase-record processing
type Processor interface {
	GetForAccountProvider(accountId uint32, serialNumber uint32) model.Provider[Model]
	GetForAccount(accountId uint32, serialNumber uint32) (Model, error)
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
	return p
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) GetForAccountProvider(accountId uint32, serialNumber uint32) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestByAccountIdAndSerialNumber(p.ctx, accountId, serialNumber), Extract)
}

func (p *ProcessorImpl) GetForAccount(accountId uint32, serialNumber uint32) (Model, error) {
	return p.GetForAccountProvider(accountId, serialNumber)()
}
