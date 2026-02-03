package handler

import (
	"atlas-channel/message"
	"atlas-channel/pet"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const PetChatHandle = "PetChatHandle"

func PetChatHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		petId := r.ReadUint64()
		updateTime := uint32(0)
		if t.Region() == "GMS" && t.MajorVersion() > 83 {
			updateTime = r.ReadUint32()
		}
		nType := r.ReadByte()
		nAction := r.ReadByte()
		msg := r.ReadAsciiString()
		l.Debugf("Character [%d] received message [%s] from pet [%d]. updateTime [%d], nType [%d], nAction [%d].", s.CharacterId(), msg, petId, updateTime, nType, nAction)
		p, err := pet.NewProcessor(l, ctx).GetById(uint32(petId))
		if err != nil {
			return
		}
		if p.OwnerId() != s.CharacterId() {
			return
		}
		_ = message.NewProcessor(l, ctx).PetChat(s.Field(), petId, msg, s.CharacterId(), p.Slot(), nType, nAction, false)
	}
}
