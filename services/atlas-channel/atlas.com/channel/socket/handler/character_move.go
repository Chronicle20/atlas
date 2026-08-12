package handler

import (
	"atlas-channel/character/chakra"
	"atlas-channel/movement"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-packet/character/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func CharacterMoveHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := serverbound.Move{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		// Movement cancels a pending Chakra heal (PRD FR-5.1). This is a
		// server-authority measure, not a simulation of client behaviour:
		// CUserLocal::IsImmovable returns true for the whole window, so an
		// authentic client physically cannot walk, jump, climb or rope and
		// never triggers this (design §3.7). It closes the crafted-client
		// hole where a player kites through the window collecting a free
		// heal and a damage factor. MP is not refunded because none was
		// spent — the generic cost block only runs on USE_SKILL.
		if chakra.GetRegistry().Clear(tenant.MustFromContext(ctx), s.CharacterId()) {
			l.Debugf("Chakra recovery window for character [%d] interrupted by movement; pending heal cancelled.", s.CharacterId())
		}

		_ = movement.NewProcessor(l, ctx, wp).ForCharacter(s.Field(), s.CharacterId(), p.MovementData())
	}
}
