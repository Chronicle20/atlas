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

// SummonDamageHandleFunc decodes an inbound SUMMON_DAMAGE packet and emits a
// COMMAND_TOPIC_SUMMON DAMAGE command. atlas-summons verifies the summon
// exists, decrements its HP by the reported amount (destroying it at zero), and
// emits a DAMAGED event that the channel rebroadcasts to other sessions in the
// map.
func SummonDamageHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := serverbound.Damage{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		// p.SummonId() is the owner cid on v83/v87 and the server summon id on v95;
		// atlas-summons reconciles via GetByOwner(senderCharacterId) when the id misses.
		// MonsterIdFrom is a mob TEMPLATE id (the client sends dwTemplateID, not an oid).
		_ = summoncmd.NewProcessor(l, ctx).Damage(s.Field(), p.SummonId(), s.CharacterId(), int32(p.Damage()), p.MonsterIdFrom())
	}
}
