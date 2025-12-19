package session

import (
	session2 "atlas-channel/kafka/message/account/session"
	"atlas-channel/kafka/producer"
	"context"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Processor interface defines the operations for session processing
type Processor interface {
	Destroy(sessionId uuid.UUID, accountId uint32)
	UpdateState(sessionId uuid.UUID, accountId uint32, state uint8, params interface{}) error
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	kp  producer.Provider
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
		kp:  producer.ProviderImpl(l)(ctx),
	}
	return p
}

func (p *ProcessorImpl) Destroy(sessionId uuid.UUID, accountId uint32) {
	p.l.Debugf("Destroying session for account [%d].", accountId)
	_ = p.kp(session2.EnvCommandTopic)(LogoutCommandProvider(sessionId, accountId))
}

func (p *ProcessorImpl) UpdateState(sessionId uuid.UUID, accountId uint32, state uint8, params interface{}) error {
	return p.kp(session2.EnvCommandTopic)(ProgressStateCommandProvider(sessionId, accountId, state, params))
}
