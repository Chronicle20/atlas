package handler

import (
	"atlas-channel/message"
	"atlas-channel/pet"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	pet2 "github.com/Chronicle20/atlas/libs/atlas-packet/pet/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

func PetChatHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		pk := pet2.ChatRequest{}
		pk.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", pk.Operation(), pk.String())
		// The wire value is the client's pet serial (GW_ItemSlotBase::liCashItemSN),
		// not the Atlas pet id — resolve it before anything downstream, which all
		// keys on the Atlas id.
		p, err := pet.NewProcessor(l, ctx).GetBySerialNumber(s.CharacterId(), pk.PetId())
		if err != nil {
			return
		}
		if p.OwnerId() != s.CharacterId() {
			return
		}
		_ = message.NewProcessor(l, ctx).PetChat(s.Field(), uint64(p.Id()), pk.Msg(), s.CharacterId(), p.Slot(), pk.NType(), pk.NAction(), false)
	}
}
