package session

import (
	session3 "atlas-login/kafka/message/account/session"
	"atlas-login/kafka/producer"
	session2 "atlas-login/kafka/producer/account/session"
	"context"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	Create(sessionId uuid.UUID, accountId uint32, accountName string, password string, ipAddress string) error
	Destroy(sessionId uuid.UUID, accountId uint32)
	UpdateState(sessionId uuid.UUID, accountId uint32, state uint8, params interface{}) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	mp  producer.Provider
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
		mp:  producer.ProviderImpl(l)(ctx),
	}
	return p
}

func (p *ProcessorImpl) Create(sessionId uuid.UUID, accountId uint32, accountName string, password string, ipAddress string) error {
	return p.mp(session3.EnvCommandTopic)(session2.CreateCommandProvider(sessionId, accountId, accountName, password, ipAddress))
}

func (p *ProcessorImpl) Destroy(sessionId uuid.UUID, accountId uint32) {
	p.l.Debugf("Destroying session for account [%d].", accountId)
	_ = p.mp(session3.EnvCommandTopic)(session2.LogoutCommandProvider(sessionId, accountId))
}

func (p *ProcessorImpl) UpdateState(sessionId uuid.UUID, accountId uint32, state uint8, params interface{}) error {
	return p.mp(session3.EnvCommandTopic)(session2.ProgressStateCommandProvider(sessionId, accountId, state, params))
}
