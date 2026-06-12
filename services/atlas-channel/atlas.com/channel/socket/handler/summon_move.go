package handler

import (
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	summoncmd "atlas-channel/summon"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-packet/summon/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

// SummonMoveHandleFunc decodes an inbound MOVE_SUMMON packet and emits a
// COMMAND_TOPIC_SUMMON MOVE command. atlas-summons verifies ownership and
// rebroadcasts the raw movement blob byte-faithfully to the rest of the map.
// The startPos carried in the packet seeds the persisted position; the raw
// movement blob is what other clients actually render.
func SummonMoveHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := serverbound.Move{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		_ = summoncmd.NewProcessor(l, ctx).Move(s.Field(), p.Oid(), s.CharacterId(), p.StartX(), p.StartY(), 0, p.RawMovement())
	}
}
