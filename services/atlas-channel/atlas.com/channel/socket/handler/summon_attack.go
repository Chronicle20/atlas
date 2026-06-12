package handler

import (
	summon2 "atlas-channel/kafka/message/summon"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	summoncmd "atlas-channel/summon"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-packet/summon/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

// SummonAttackHandleFunc decodes an inbound SUMMON_ATTACK packet and emits a
// COMMAND_TOPIC_SUMMON ATTACK command. atlas-summons verifies ownership,
// credits the owner, clamps the per-target damage, and emits an ATTACKED event
// that the channel rebroadcasts to other sessions in the map.
func SummonAttackHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := serverbound.Attack{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		targets := make([]summon2.AttackTargetEntry, 0, len(p.Targets()))
		for _, t := range p.Targets() {
			targets = append(targets, summon2.AttackTargetEntry{MonsterId: t.MonsterOid(), Damage: t.Damage()})
		}

		_ = summoncmd.NewProcessor(l, ctx).Attack(s.Field(), p.Oid(), s.CharacterId(), p.Direction(), targets)
	}
}
