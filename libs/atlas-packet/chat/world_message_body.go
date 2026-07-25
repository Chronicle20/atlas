package chat

import (
	"context"

	"github.com/sirupsen/logrus"

	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	"github.com/Chronicle20/atlas/libs/atlas-packet/chat/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

type WorldMessageMode string

const (
	WorldMessageMegaphone      WorldMessageMode = "MEGAPHONE"
	WorldMessageSuperMegaphone WorldMessageMode = "SUPER_MEGAPHONE"
	WorldMessageItemMegaphone  WorldMessageMode = "ITEM_MEGAPHONE"
	WorldMessageMultiMegaphone WorldMessageMode = "MULTI_MEGAPHONE"
)

func WorldMessageMegaphoneBody(message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(WorldMessageMegaphone), func(mode byte) packet.Encoder {
		return clientbound.NewWorldMessageMegaphone(mode, message)
	})
}

func WorldMessageSuperMegaphoneBody(message string, channelId byte, whispersOn bool) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(WorldMessageSuperMegaphone), func(mode byte) packet.Encoder {
		return clientbound.NewWorldMessageSuperMegaphone(mode, message, channelId, whispersOn)
	})
}

func WorldMessageItemMegaphoneBody(message string, channelId byte, whispersOn bool, item *model.Asset) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(WorldMessageItemMegaphone), func(mode byte) packet.Encoder {
		return clientbound.NewWorldMessageItemMegaphone(mode, message, channelId, whispersOn, item)
	})
}

func WorldMessageMultiMegaphoneBody(messages []string, channelId byte, whispersOn bool) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(WorldMessageMultiMegaphone), func(mode byte) packet.Encoder {
		return clientbound.NewWorldMessageMultiMegaphone(mode, messages, channelId, whispersOn)
	})
}
