package handler

import (
	"atlas-channel/character"
	"atlas-channel/invite"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	invite2 "github.com/Chronicle20/atlas-constants/invite"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const (
	PartyInviteRejectHandle = "PartyInviteRejectHandle"
)

func PartyInviteRejectHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		unk := r.ReadByte()
		from := r.ReadAsciiString()
		l.Debugf("Rejecting party invite from [%s]. unk [%d].", from, unk)

		cs, err := character.NewProcessor(l, ctx).GetByName(from)
		if err != nil {
			l.WithError(err).Errorf("Unable to locate character by name [%s]. Invite will be stuck", from)
			return
		}

		err = invite.NewProcessor(l, ctx).Reject(s.CharacterId(), s.WorldId(), string(invite2.TypeParty), cs.Id())
		if err != nil {
			l.WithError(err).Errorf("Unable to issue invite rejection command for character [%d].", s.CharacterId())
		}
	}
}
