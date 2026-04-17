package handler

import (
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const CharacterTouchAttackHandle = "CharacterTouchAttackHandle"

func CharacterTouchAttackHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		at := packetmodel.NewAttackInfo(packetmodel.AttackTypeEnergy)
		at.Decode(l, ctx)(r, readerOptions)
		l.Debugf("Character [%d] is attempting a touch attack.", s.CharacterId())
		err := processAttack(l)(ctx)(wp)(*at)(s)
		if err != nil {
			l.WithError(err).Errorf("Unable to completely process character [%d] touch attack.", s.CharacterId())
		}
	}
}
