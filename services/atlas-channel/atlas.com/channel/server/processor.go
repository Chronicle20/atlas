package server

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	Register(t tenant.Model, ch channel.Model, ipAddress string, port int) Model
	GetAll() []Model
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

func (p *ProcessorImpl) Register(t tenant.Model, ch channel.Model, ipAddress string, port int) Model {
	m := Model{
		tenant:    t,
		ch:        ch,
		ipAddress: ipAddress,
		port:      port,
	}
	getRegistry().Register(m)
	return m
}

func (p *ProcessorImpl) GetAll() []Model {
	return getRegistry().GetAll()
}
