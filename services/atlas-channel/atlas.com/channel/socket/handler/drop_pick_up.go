package handler

import (
	"atlas-channel/drop"
	"atlas-channel/party"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const DropPickUpHandle = "DropPickUpHandle"

func DropPickUpHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		fieldKey := r.ReadByte()
		updateTime := r.ReadUint32()
		x := r.ReadInt16()
		y := r.ReadInt16()
		dropId := r.ReadUint32()
		crc := r.ReadUint32()
		l.Debugf("Character [%d] is attempting to pick up drop [%d] at [%d,%d]. FieldKey [%d], UpdateTime [%d], crc [%d].", s.CharacterId(), dropId, x, y, fieldKey, updateTime, crc)

		var partyId uint32
		p, err := party.NewProcessor(l, ctx).GetByMemberId(s.CharacterId())
		if err == nil {
			partyId = p.Id()
		}

		_ = drop.NewProcessor(l, ctx).RequestReservation(s.Field(), dropId, s.CharacterId(), partyId, x, y, -1)
	}
}
