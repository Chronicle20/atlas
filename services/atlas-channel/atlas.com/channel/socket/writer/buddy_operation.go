package writer

import (
	"atlas-channel/buddylist/buddy"
	"context"

	"github.com/Chronicle20/atlas-constants/channel"
	atlas_packet "github.com/Chronicle20/atlas-packet"
	buddypkt "github.com/Chronicle20/atlas-packet/buddy"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

const (
	BuddyOperationUpdate                 = "UPDATE"
	BuddyOperationBuddyUpdate            = "BUDDY_UPDATE"
	BuddyOperationInvite                 = "INVITE"
	BuddyOperationUnknown1               = "UNKNOWN_1" // same as UPDATE
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

func BuddyInviteBody(actorId uint32, originatorId uint32, originatorName string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getBuddyOperation(l)(options, BuddyOperationInvite)
			return buddypkt.NewBuddyInvite(mode, actorId, originatorId, originatorName).Encode(l, ctx)(options)
		}
	}
}

func BuddyListUpdateBody(buddies []buddy.Model) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getBuddyOperation(l)(options, BuddyOperationUpdate)
			entries := make([]buddypkt.BuddyEntry, 0, len(buddies))
			for _, b := range buddies {
				entries = append(entries, buddypkt.BuddyEntry{
					CharacterId: b.CharacterId(),
					Name:        b.Name(),
					ChannelId:   channel.Id(b.ChannelId()),
					Group:       b.Group(),
					InShop:      b.InShop(),
				})
			}
			return buddypkt.NewBuddyListUpdate(mode, entries).Encode(l, ctx)(options)
		}
	}
}

func BuddyUpdateBody(characterId uint32, group string, characterName string, channelId int8, inShop bool) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getBuddyOperation(l)(options, BuddyOperationBuddyUpdate)
			return buddypkt.NewBuddyUpdate(mode, characterId, characterName, group, channel.Id(channelId), inShop).Encode(l, ctx)(options)
		}
	}
}

func BuddyErrorBody(errorCode string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getBuddyOperation(l)(options, errorCode)
			hasExtra := errorCode == BuddyOperationErrorUnknownError
			return buddypkt.NewBuddyError(mode, hasExtra).Encode(l, ctx)(options)
		}
	}
}

func BuddyChannelChangeBody(characterId uint32, channelId int8) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getBuddyOperation(l)(options, BuddyOperationBuddyChannelChange)
			return buddypkt.NewBuddyChannelChange(mode, characterId, channelId).Encode(l, ctx)(options)
		}
	}
}

func BuddyCapacityUpdateBody(capacity byte) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getBuddyOperation(l)(options, BuddyOperationCapacityUpdate)
			return buddypkt.NewBuddyCapacityUpdate(mode, capacity).Encode(l, ctx)(options)
		}
	}
}

func getBuddyOperation(l logrus.FieldLogger) func(options map[string]interface{}, key string) byte {
	return func(options map[string]interface{}, key string) byte {
		return atlas_packet.ResolveCode(l, options, "operations", key)
	}
}
