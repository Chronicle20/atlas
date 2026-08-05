package handler

import (
	"atlas-channel/report"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	reportsb "github.com/Chronicle20/atlas/libs/atlas-packet/report/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

func ClaimRequestHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, ro map[string]interface{}) {
	return func(s session.Model, r *request.Reader, ro map[string]interface{}) {
		p := reportsb.ClaimRequest{}
		p.Decode(l, ctx)(r, ro)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		err := report.NewProcessor(l, ctx).Claim(s.CharacterId(), s.WorldId(), s.ChannelId(), p.TargetName(), p.ReasonType(), p.Description(), p.IsChatClaim(), p.ChatLog())
		if err != nil {
			l.WithError(err).Errorf("Unable to submit claim report from character [%d].", s.CharacterId())
		}
	}
}
