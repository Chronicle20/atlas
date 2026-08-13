package report

import (
	report2 "atlas-channel/kafka/message/report"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
)

// ClaimCostMesos is the fee a successful claim costs the reporter. The client
// has a dedicated CLAIM_RESULT mode for the affordability failure
// (NOT_ENOUGH_MESOS, "You do not have enough mesos to report"), so the cost is
// a first-class part of the claim flow rather than a silent debit.
//
// The charge is applied only after atlas-ban confirms the report was created:
// a claim rejected for an unknown target or an exhausted quota is free.
const ClaimCostMesos int32 = 300

// ClaimFeeActorType labels the meso adjustment in atlas-character's
// meso-changed status event, the same way skill costs label theirs "SKILL".
const ClaimFeeActorType = "REPORT"

// Processor defines report submission operations available to the channel.
type Processor interface {
	// Sue submits a /-command report. Legacy versions supply accusedId;
	// v95 supplies subCommand (forwarded as the accused name). The ban
	// service resolves whichever half is missing.
	Sue(reporterId uint32, worldId world.Id, channelId channel.Id, accusedId uint32, subCommand string, flag byte, reason string) error
	// Claim submits a CUIClaim report window submission.
	Claim(reporterId uint32, worldId world.Id, channelId channel.Id, targetName string, reasonType byte, description string, chatClaim bool, chatLog string) error
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) Sue(reporterId uint32, worldId world.Id, channelId channel.Id, accusedId uint32, subCommand string, flag byte, reason string) error {
	p.l.Debugf("Character [%d] sues [%d/%s] flag [%d].", reporterId, accusedId, subCommand, flag)
	return producer.ProviderImpl(p.l)(p.ctx)(report2.EnvCommandTopic)(sueCommandProvider(reporterId, worldId, channelId, accusedId, subCommand, flag, reason))
}

func (p *ProcessorImpl) Claim(reporterId uint32, worldId world.Id, channelId channel.Id, targetName string, reasonType byte, description string, chatClaim bool, chatLog string) error {
	p.l.Debugf("Character [%d] claims against [%s] type [%d] chatClaim [%t].", reporterId, targetName, reasonType, chatClaim)
	return producer.ProviderImpl(p.l)(p.ctx)(report2.EnvCommandTopic)(claimCommandProvider(reporterId, worldId, channelId, targetName, reasonType, description, chatClaim, chatLog))
}
