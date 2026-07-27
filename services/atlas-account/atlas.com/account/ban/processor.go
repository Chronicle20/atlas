package ban

import (
	"context"

	"github.com/sirupsen/logrus"
)

type Processor interface {
	CheckBan(ip string, hwid string, accountId uint32) (CheckRestModel, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) CheckBan(ip string, hwid string, accountId uint32) (CheckRestModel, error) {
	return requestCheckBan(ip, hwid, accountId)(p.l, p.ctx)
}
