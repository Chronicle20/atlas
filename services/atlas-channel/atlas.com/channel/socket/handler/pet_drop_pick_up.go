package handler

import (
	"atlas-channel/drop"
	"atlas-channel/pet"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const PetDropPickUpHandle = "PetDropPickUpHandle"

func PetDropPickUpHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		petId := r.ReadUint64()
		fieldKey := r.ReadByte()
		updateTime := r.ReadUint32()
		x := r.ReadInt16()
		y := r.ReadInt16()
		dropId := r.ReadUint32()
		crc := r.ReadUint32()
		bPickupOthers := r.ReadBool()
		bSweepForDrop := r.ReadBool()
		bLongRange := r.ReadBool()
		ownerX := int16(0)
		ownerY := int16(0)
		posCrc := uint32(0)
		rectCrc := uint32(0)
		if t.Region() == "GMS" && t.MajorVersion() > 83 {
			if dropId%13 != 0 {
				ownerX = r.ReadInt16()
				ownerY = r.ReadInt16()
				posCrc = r.ReadUint32()
				rectCrc = r.ReadUint32()
			}
		}

		p, err := pet.NewProcessor(l, ctx).GetById(uint32(petId))
		if err != nil {
			l.WithError(err).Errorf("Unable to find pet [%d]", petId)
		}

		l.Debugf("Character [%d] pet [%d] attempting to pick up drop [%d]. fieldKey [%d], updateTime [%d], x [%d], y[%d], crc [%d], bPickupOthers [%t], bSweepForDrop [%t], bLongRange [%t], ownerX [%d], ownerY [%d], posCrc [%d], rectCrc[%d].", s.CharacterId(), petId, dropId, fieldKey, updateTime, x, y, crc, bPickupOthers, bSweepForDrop, bLongRange, ownerX, ownerY, posCrc, rectCrc)

		_ = drop.NewProcessor(l, ctx).RequestReservation(s.Map(), dropId, s.CharacterId(), x, y, p.Slot())
	}
}
