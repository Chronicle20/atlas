package handler

import (
	"atlas-channel/character"
	"atlas-channel/invite"
	invite2 "atlas-channel/kafka/message/invite"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const (
	GuildInviteRejectHandle = "GuildInviteRejectHandle"
)

func GuildInviteRejectHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		unk := r.ReadByte()
		from := r.ReadAsciiString()
		l.Debugf("Rejecting guild invite from [%s]. unk [%d].", from, unk)

		cs, err := character.NewProcessor(l, ctx).GetByName(from)
		if err != nil {
			l.WithError(err).Errorf("Unable to locate character by name [%s]. Invite will be stuck", from)
			return
		}

		err = invite.NewProcessor(l, ctx).Reject(s.CharacterId(), s.WorldId(), invite2.InviteTypeGuild, cs.Id())
		if err != nil {
			l.WithError(err).Errorf("Unable to issue invite rejection command for character [%d].", s.CharacterId())
		}
	}
}
