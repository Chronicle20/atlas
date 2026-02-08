package gachapon

import (
	"context"

	"github.com/sirupsen/logrus"
)

type Processor interface {
	SelectReward(gachaponId string) (RewardRestModel, error)
	GetGachapon(gachaponId string) (GachaponRestModel, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

func (p *ProcessorImpl) SelectReward(gachaponId string) (RewardRestModel, error) {
	return SelectReward(p.l, p.ctx)(gachaponId)
}

func (p *ProcessorImpl) GetGachapon(gachaponId string) (GachaponRestModel, error) {
	return GetGachapon(p.l, p.ctx)(gachaponId)
}
