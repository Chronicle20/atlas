package handler

import (
	"atlas-channel/character"
	"atlas-channel/report"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	reportcb "github.com/Chronicle20/atlas/libs/atlas-packet/report/clientbound"
	reportsb "github.com/Chronicle20/atlas/libs/atlas-packet/report/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// claimSubmitDeps isolates submitClaim from the REST client, the Kafka
// producer and the session registry, the same way damageMitigationDeps does
// for character damage.
type claimSubmitDeps struct {
	getCharacter func(characterId uint32) (character.Model, error)
	submitClaim  func(p reportsb.ClaimRequest) error
	notice       func(code writer.ClaimResultCode)
}

func ClaimRequestHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, ro map[string]interface{}) {
	return func(s session.Model, r *request.Reader, ro map[string]interface{}) {
		p := reportsb.ClaimRequest{}
		p.Decode(l, ctx)(r, ro)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		rp := report.NewProcessor(l, ctx)
		submitClaim(l, s.CharacterId(), p, claimSubmitDeps{
			getCharacter: character.NewProcessor(l, ctx).GetById(),
			submitClaim: func(p reportsb.ClaimRequest) error {
				return rp.Claim(s.CharacterId(), s.WorldId(), s.ChannelId(), p.TargetName(), p.ReasonType(), p.Description(), p.IsChatClaim(), p.ChatLog())
			},
			notice: func(code writer.ClaimResultCode) { announceClaimNotice(l, ctx, wp, s, code) },
		})
	}
}

// submitClaim gates the claim on the reporter being able to pay the fee
// before the create command is emitted, purely so they get the client's
// dedicated "not enough mesos" notice instead of a report that vanishes when
// the debit is refused. The authoritative guard is atlas-character's
// RequestChangeMeso, which refuses any adjustment leaving meso negative; the
// fee itself is charged only once atlas-ban confirms creation
// (kafka/consumer/report/consumer.go).
func submitClaim(l logrus.FieldLogger, characterId uint32, p reportsb.ClaimRequest, deps claimSubmitDeps) {
	c, err := deps.getCharacter(characterId)
	if err != nil {
		l.WithError(err).Errorf("Unable to resolve character [%d] submitting a claim.", characterId)
		deps.notice(writer.ClaimResultTryAgain)
		return
	}
	if c.Meso() < uint32(report.ClaimCostMesos) {
		l.Debugf("Character [%d] cannot afford the [%d] meso claim fee; holds [%d].", characterId, report.ClaimCostMesos, c.Meso())
		deps.notice(writer.ClaimResultNotEnoughMesos)
		return
	}
	if err = deps.submitClaim(p); err != nil {
		l.WithError(err).Errorf("Unable to submit claim report from character [%d].", characterId)
	}
}

// announceClaimNotice sends a bare-mode CLAIM_RESULT to the submitting
// session. A tenant whose socket config maps no ClaimResult writer is not an
// error — config presence IS the feature gate, exactly as the report status
// consumer treats it (kafka/consumer/report/consumer.go).
func announceClaimNotice(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, s session.Model, code writer.ClaimResultCode) {
	if _, err := wp(reportcb.ClaimResultWriter); err != nil {
		l.Debugf("Tenant configuration has no writer [%s] mapped; skipping claim notice [%s] for character [%d].", reportcb.ClaimResultWriter, code, s.CharacterId())
		return
	}
	if err := session.Announce(l)(ctx)(wp)(reportcb.ClaimResultWriter)(writer.ClaimResultNoticeBody(code))(s); err != nil {
		l.WithError(err).Errorf("Unable to deliver claim notice [%s] to character [%d].", code, s.CharacterId())
	}
}
