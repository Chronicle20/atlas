package controller

import (
	"atlas-channel/data/npc"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	npcpkt "github.com/Chronicle20/atlas/libs/atlas-packet/npc/clientbound"
)

// AnnounceGrant sends the controller grant (OnNpcChangeController flag 1)
// for npcObjectId to characterId's session, if present on this pod.
func AnnounceGrant(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(f field.Model, characterId uint32, npcObjectId uint32) error {
	return func(f field.Model, characterId uint32, npcObjectId uint32) error {
		return session.NewProcessor(l, ctx).IfPresentByCharacterId(f.Channel())(characterId, func(s session.Model) error {
			n, err := npc.NewProcessor(l, ctx).GetInMapByObjectId(f.MapId(), npcObjectId)
			if err != nil {
				l.WithError(err).Warnf("Unable to load NPC [%d] for controller grant to [%d].", npcObjectId, characterId)
				return err
			}
			l.Debugf("Granting NPC [%d] control to character [%d] in field [%s].", npcObjectId, characterId, f.Id())
			return session.Announce(l)(ctx)(wp)(npcpkt.NpcSpawnRequestControllerWriter)(npcpkt.NewNpcSpawnRequestController(n.Id(), n.Template(), n.X(), n.CY(), int32(n.F()), n.Fh(), n.RX0(), n.RX1(), true).Encode)(s)
		})
	}
}

// AnnounceRevoke sends the remove-controller arm (flag 0) for npcObjectId
// to s — the client demotes the NPC to remote control (FR-6.1).
func AnnounceRevoke(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, npcObjectId uint32) error {
	return func(s session.Model, npcObjectId uint32) error {
		l.Debugf("Revoking NPC [%d] control from character [%d].", npcObjectId, s.CharacterId())
		return session.Announce(l)(ctx)(wp)(npcpkt.NpcSpawnRequestControllerWriter)(npcpkt.NewNpcRemoveController(npcObjectId).Encode)(s)
	}
}
