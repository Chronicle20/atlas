package handler

import (
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	summoncmd "atlas-channel/summon"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-packet/summon/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// SummonMoveHandleFunc decodes an inbound MOVE_SUMMON packet and emits a
// COMMAND_TOPIC_SUMMON MOVE command. atlas-summons verifies ownership and
// rebroadcasts the raw movement blob to the rest of the map. The blob travels
// unchanged through the command and event, but is re-serialized at encode time
// for each RECEIVING client (see writer.SummonMoveBody): GMS v87 reads the
// per-element XOffset/YOffset pair this handler receives and must never be sent
// it back. The startPos carried in the packet seeds the persisted position; the
// movement blob is what other clients actually render.
func SummonMoveHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := serverbound.Move{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		// p.SummonId() is the owner cid on v83/v87 (the client has no oid; the
		// summon pool is cid-keyed) and the server summon id on v95. atlas-summons
		// reconciles: it tries the id, then falls back to GetByOwner(senderCharacterId).
		_ = summoncmd.NewProcessor(l, ctx).Move(s.Field(), p.SummonId(), s.CharacterId(), p.StartX(), p.StartY(), 0, p.RawMovement())
	}
}
