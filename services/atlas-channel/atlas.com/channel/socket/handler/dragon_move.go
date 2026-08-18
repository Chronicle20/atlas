package handler

import (
	dragoncmd "atlas-channel/dragon"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-packet/dragon/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// DragonMoveHandleFunc decodes an inbound MOVE_DRAGON packet and emits a
// COMMAND_TOPIC_DRAGON MOVE command keyed on the SENDING SESSION's character id.
//
// The packet carries no identity field at all (CVecCtrlDragon::EndUpdateActive
// writes only the CMovePath blob), so the session IS the identity: there is no
// id to reconcile and no cross-character spoofing surface. atlas-dragons drops
// the command if the sender has no dragon; it never creates one as a side
// effect.
func DragonMoveHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := serverbound.Move{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		if err := dragoncmd.NewProcessor(l, ctx).Move(s.Field(), s.CharacterId(), p.StartX(), p.StartY(), 0, p.RawMovement()); err != nil {
			// Non-fatal: a failed move relay must not kill the session. Logged
			// so a persistent failure (e.g. Redis errors surfaced from
			// atlas-dragons) is visible instead of silently dropped.
			l.WithError(err).Errorf("Unable to relay dragon move for character [%d].", s.CharacterId())
		}
	}
}
