package handler

import (
	"atlas-channel/character"
	"atlas-channel/invite"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	invite2 "github.com/Chronicle20/atlas-constants/invite"
	guildsb "github.com/Chronicle20/atlas-packet/guild/serverbound"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func GuildInviteRejectHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := guildsb.InviteReject{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		cs, err := character.NewProcessor(l, ctx).GetByName(p.From())
		if err != nil {
			l.WithError(err).Errorf("Unable to locate character by name [%s]. Invite will be stuck", p.From())
			return
		}

		err = invite.NewProcessor(l, ctx).Reject(s.CharacterId(), s.WorldId(), string(invite2.TypeGuild), cs.Id())
		if err != nil {
			l.WithError(err).Errorf("Unable to issue invite rejection command for character [%d].", s.CharacterId())
		}
	}
}
