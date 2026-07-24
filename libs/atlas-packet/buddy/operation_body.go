package buddy

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	"github.com/Chronicle20/atlas/libs/atlas-packet/buddy/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
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

func BuddyInviteBody(actorId uint32, originatorId uint32, originatorName string, jobId uint32, level uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", BuddyOperationInvite, func(mode byte) packet.Encoder {
		return clientbound.NewBuddyInvite(mode, actorId, originatorId, originatorName, jobId, level)
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

func BuddyListFullBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", BuddyOperationErrorListFull, func(mode byte) packet.Encoder {
		return clientbound.NewListFull(mode)
	})
}

func BuddyOtherListFullBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", BuddyOperationErrorOtherListFull, func(mode byte) packet.Encoder {
		return clientbound.NewOtherListFull(mode)
	})
}

func BuddyAlreadyBuddyBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", BuddyOperationErrorAlreadyBuddy, func(mode byte) packet.Encoder {
		return clientbound.NewAlreadyBuddy(mode)
	})
}

func BuddyCannotBuddyGmBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", BuddyOperationErrorCannotBuddyGm, func(mode byte) packet.Encoder {
		return clientbound.NewCannotBuddyGm(mode)
	})
}

func BuddyCharacterNotFoundBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", BuddyOperationErrorCharacterNotFound, func(mode byte) packet.Encoder {
		return clientbound.NewCharacterNotFound(mode)
	})
}

func BuddyUnknownErrorBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", BuddyOperationErrorUnknownError, func(mode byte) packet.Encoder {
		return clientbound.NewUnknownError(mode)
	})
}

func BuddyUnknownError2Body() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", BuddyOperationErrorUnknownError2, func(mode byte) packet.Encoder {
		return clientbound.NewUnknownError2(mode)
	})
}

func BuddyUnknownError3Body() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", BuddyOperationErrorUnknownError3, func(mode byte) packet.Encoder {
		return clientbound.NewUnknownError3(mode)
	})
}

func BuddyUnknownError4Body() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", BuddyOperationErrorUnknownError4, func(mode byte) packet.Encoder {
		return clientbound.NewUnknownError4(mode)
	})
}
