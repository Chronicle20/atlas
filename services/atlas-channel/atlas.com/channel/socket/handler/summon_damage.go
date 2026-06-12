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

		_ = summoncmd.NewProcessor(l, ctx).Damage(s.Field(), p.Oid(), s.CharacterId(), int32(p.Damage()), p.MonsterIdFrom())
	}
}
