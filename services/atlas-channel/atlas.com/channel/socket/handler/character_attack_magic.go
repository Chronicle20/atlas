package handler

import (
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	packetmodel "github.com/Chronicle20/atlas-packet/model"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const CharacterMagicAttackHandle = "CharacterMagicAttackHandle"

func CharacterMagicAttackHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		at := packetmodel.NewAttackInfo(packetmodel.AttackTypeMagic)
		at.Decode(l, ctx)(r, readerOptions)
		l.Debugf("Character [%d] is attempting a magic attack.", s.CharacterId())
		err := processAttack(l)(ctx)(wp)(*at)(s)
		if err != nil {
			l.WithError(err).Errorf("Unable to completely process character [%d] magic attack.", s.CharacterId())
		}
	}
}
