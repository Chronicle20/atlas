package conversation

import (
	"atlas-npc-conversations/conversation/script"
	registry2 "atlas-npc-conversations/conversation/script/registry"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

func Start(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, npcId uint32, characterId uint32) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, npcId uint32, characterId uint32) error {
		t := tenant.MustFromContext(ctx)
		return func(worldId byte, channelId byte, mapId uint32, npcId uint32, characterId uint32) error {
			l.Debugf("Starting conversation with NPC [%d] with character [%d] in map [%d].", npcId, characterId, mapId)
			pctx, err := GetRegistry().GetPreviousContext(t, characterId)
			if err == nil {
				l.Debugf("Previous conversation between character [%d] and npc [%d] exists, avoiding starting new conversation with [%d].", characterId, pctx.ctx.NPCId, npcId)
				return errors.New("another conversation exists")
			}

			s, err := registry2.GetRegistry().GetScript(t, npcId)
			if err != nil {
				l.Errorf("Script for npc [%d] is not implemented.", npcId)
				return errors.New("not implemented")
			}

			sctx := script.Context{
				WorldId:     worldId,
				ChannelId:   channelId,
				CharacterId: characterId,
				MapId:       mapId,
				NPCId:       npcId,
			}
			ns := (*s).Initial(l)(ctx)(sctx)

			if ns != nil {
				GetRegistry().SetContext(t, characterId, sctx, ns)
			} else {
				GetRegistry().ClearContext(t, characterId)
			}

			return nil
		}
	}

}
