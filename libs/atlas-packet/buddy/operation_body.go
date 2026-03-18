package buddy

import (
	"context"

	"github.com/Chronicle20/atlas-constants/channel"
	atlas_packet "github.com/Chronicle20/atlas-packet"
	"github.com/Chronicle20/atlas-packet/buddy/clientbound"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

const (
	BuddyOperationUpdate                 = "UPDATE"
	BuddyOperationBuddyUpdate            = "BUDDY_UPDATE"
	BuddyOperationInvite                 = "INVITE"
	BuddyOperationUnknown1               = "UNKNOWN_1"
	BuddyOperationErrorListFull          = "BUDDY_LIST_FULL"
	BuddyOperationErrorOtherListFull     = "OTHER_BUDDY_LIST_FULL"
	BuddyOperationErrorAlreadyBuddy      = "ALREADY_BUDDY"
	BuddyOperationErrorCannotBuddyGm     = "CANNOT_BUDDY_GM"
	BuddyOperationErrorCharacterNotFound = "CHARACTER_NOT_FOUND"
	BuddyOperationErrorUnknownError      = "UNKNOWN_ERROR"
	BuddyOperationErrorUnknownError2     = "UNKNOWN_ERROR_2"
	BuddyOperationUnknown2               = "UNKNOWN_2"
	BuddyOperationErrorUnknownError3     = "UNKNOWN_ERROR_3"
	BuddyOperationBuddyChannelChange     = "BUDDY_CHANNEL_CHANGE"
	BuddyOperationCapacityUpdate         = "CAPACITY_CHANGE"
	BuddyOperationErrorUnknownError4     = "UNKNOWN_ERROR_4"
)

func BuddyInviteBody(actorId uint32, originatorId uint32, originatorName string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", BuddyOperationInvite, func(mode byte) packet.Encoder {
		return clientbound.NewBuddyInvite(mode, actorId, originatorId, originatorName)
	})
}

func BuddyListUpdateBody(buddies []clientbound.BuddyEntry) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", BuddyOperationUpdate, func(mode byte) packet.Encoder {
		return clientbound.NewBuddyListUpdate(mode, buddies)
	})
}

func BuddyUpdateBody(characterId uint32, group string, characterName string, channelId int8, inShop bool) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", BuddyOperationBuddyUpdate, func(mode byte) packet.Encoder {
		return clientbound.NewBuddyUpdate(mode, characterId, characterName, group, channel.Id(channelId), inShop)
	})
}

func BuddyErrorBody(errorCode string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	hasExtra := errorCode == BuddyOperationErrorUnknownError
	return atlas_packet.WithResolvedCode("operations", errorCode, func(mode byte) packet.Encoder {
		return clientbound.NewBuddyError(mode, hasExtra)
	})
}

func BuddyChannelChangeBody(characterId uint32, channelId int8) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", BuddyOperationBuddyChannelChange, func(mode byte) packet.Encoder {
		return clientbound.NewBuddyChannelChange(mode, characterId, channelId)
	})
}

func BuddyCapacityUpdateBody(capacity byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", BuddyOperationCapacityUpdate, func(mode byte) packet.Encoder {
		return clientbound.NewBuddyCapacityUpdate(mode, capacity)
	})
}
