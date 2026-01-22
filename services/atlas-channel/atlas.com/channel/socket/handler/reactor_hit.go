package handler

import (
	"atlas-channel/reactor"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const ReactorHitHandle = "ReactorHitHandle"

func ReactorHitHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		oid := r.ReadUint32()
		isSkill := r.ReadUint32() == 1
		dwHitOption := r.ReadUint32()
		bMoveAction := dwHitOption & 1
		direction := (dwHitOption >> 1) & 1
		delay := r.ReadUint16()
		skillId := r.ReadUint32()
		l.Debugf("Character [%d] has hit reactor oid [%d]. isSkill [%t], bMoveAction [%d], direction [%d], delay [%d], skillId [%d].", s.CharacterId(), oid, isSkill, bMoveAction, direction, delay, skillId)

		stance := uint16(bMoveAction) | uint16(direction<<1)
		err := reactor.NewProcessor(l, ctx).Hit(s.Map(), oid, s.CharacterId(), stance, skillId)
		if err != nil {
			l.WithError(err).Errorf("Unable to send hit command for reactor [%d].", oid)
		}
	}
}
