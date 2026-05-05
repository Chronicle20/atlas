package monsterbook

import (
	mbmsg "atlas-channel/kafka/message/monsterbook"
	"atlas-channel/kafka/producer"
	"context"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

// Processor exposes monster book emissions from atlas-channel.
type Processor interface {
	RequestSetCover(characterId uint32, coverCardId uint32) error
}

// ProcessorImpl emits SET_COVER commands to the monster book service.
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
}

// NewProcessor builds a Processor bound to the request context's tenant.
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx, t: tenant.MustFromContext(ctx)}
}

// RequestSetCover emits a SET_COVER command keyed on the character.
func (p *ProcessorImpl) RequestSetCover(characterId uint32, coverCardId uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(mbmsg.EnvCommandTopic)(SetCoverCommandProvider(p.t.Id(), characterId, coverCardId))
}
