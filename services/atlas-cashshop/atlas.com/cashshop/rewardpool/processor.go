package rewardpool

import (
	"context"
	"errors"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// ErrPoolMissing means no cash-surprise pool is configured for this box
// template id (404). ErrPoolEmpty means the pool exists but has no eligible
// entries (409, task-207 FR-3.7). They are distinct so the open path can log
// POOL_MISSING vs POOL_EMPTY — the client sees the same bare FAILED arm
// either way, so the log is the only place the difference survives.
var (
	ErrPoolMissing = errors.New("no cash-surprise pool configured for box template")
	ErrPoolEmpty   = errors.New("cash-surprise pool has no eligible entries")
)

type Model struct {
	itemId      uint32
	quantity    uint32
	commodityId uint32
}

func (m Model) ItemId() uint32      { return m.itemId }
func (m Model) Quantity() uint32    { return m.quantity }
func (m Model) CommodityId() uint32 { return m.commodityId }

type Processor interface {
	SelectReward(boxTemplateId uint32) (Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) SelectReward(boxTemplateId uint32) (Model, error) {
	rm, err := requestSelectReward(boxTemplateId)(p.l, p.ctx)
	if err != nil {
		return Model{}, classifySelectError(err)
	}
	return Model{itemId: rm.ItemId, quantity: rm.Quantity, commodityId: rm.CommodityId}, nil
}

// classifySelectError distinguishes the two *configuration* faults from
// everything else. An infrastructure fault must NOT be reported as a
// misconfigured pool — the operator would go looking in the wrong place.
func classifySelectError(err error) error {
	if errors.Is(err, requests.ErrNotFound) {
		return errors.Join(ErrPoolMissing, err)
	}
	if errors.Is(err, requests.ErrConflict) {
		return errors.Join(ErrPoolEmpty, err)
	}
	return err
}
