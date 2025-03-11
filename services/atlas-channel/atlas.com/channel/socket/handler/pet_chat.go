package handler

import (
	_map "atlas-channel/map"
	"atlas-channel/pet"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const PetChatHandle = "PetChatHandle"

func PetChatHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
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
		p, err := pet.GetById(l)(ctx)(petId)
		if err != nil {
			return
		}
		if p.OwnerId() != s.CharacterId() {
			return
		}
		_ = _map.ForSessionsInMap(l)(ctx)(s.Map(), session.Announce(l)(ctx)(wp)(writer.PetChat)(writer.PetChatBody(p, nType, nAction, msg, false)))
	}
}
