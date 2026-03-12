package handler

import (
	"atlas-channel/message"
	"atlas-channel/pet"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	pet2 "github.com/Chronicle20/atlas-packet/pet"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func PetChatHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		pk := pet2.ChatRequest{}
		pk.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", pk.Operation(), pk.String())
		p, err := pet.NewProcessor(l, ctx).GetById(uint32(pk.PetId()))
		if err != nil {
			return
		}
		if p.OwnerId() != s.CharacterId() {
			return
		}
		_ = message.NewProcessor(l, ctx).PetChat(s.Field(), pk.PetId(), pk.Msg(), s.CharacterId(), p.Slot(), pk.NType(), pk.NAction(), false)
	}
}
