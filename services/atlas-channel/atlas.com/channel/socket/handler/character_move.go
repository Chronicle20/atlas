package handler

import (
	"atlas-channel/movement"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-packet/character/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

func CharacterMoveHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := serverbound.Move{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		_ = movement.NewProcessor(l, ctx, wp).ForCharacter(s.Field(), s.CharacterId(), p.MovementData())
	}
}
