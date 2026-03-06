package writer

import (
	"atlas-channel/buddylist/buddy"
	"atlas-channel/socket/model"
	"context"
	"strconv"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const (
	BuddyOperation                       = "BuddyOperation"
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
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getBuddyOperation(l)(options, BuddyOperationInvite))
			w.WriteInt(originatorId)
			w.WriteAsciiString(originatorName)

			b := model.Buddy{
				FriendId:    actorId,
				FriendName:  originatorName,
				Flag:        0,
				ChannelId:   0,
				FriendGroup: "Default Group",
			}
			w.WriteByteArray(b.Encoder(l, ctx)(options))
			w.WriteByte(0) // 0 no, 1 true m_aInShop
			return w.Bytes()
		}
	}
}

func BuddyListUpdateBody(buddies []buddy.Model) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getBuddyOperation(l)(options, BuddyOperationUpdate))
			w.WriteByte(byte(len(buddies)))
			for _, b := range buddies {
				m := model.Buddy{
					FriendId:    b.CharacterId(),
					FriendName:  b.Name(),
					Flag:        0,
					ChannelId:   channel.Id(b.ChannelId()),
					FriendGroup: b.Group(),
				}
				w.WriteByteArray(m.Encoder(l, ctx)(options))
			}
			for _, b := range buddies {
				if b.InShop() {
					w.WriteInt(1)
				} else {
					w.WriteInt(0)
				}
			}
			return w.Bytes()
		}
	}
}

func BuddyUpdateBody(characterId uint32, group string, characterName string, channelId int8, inShop bool) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getBuddyOperation(l)(options, BuddyOperationBuddyUpdate))
			w.WriteInt(characterId)
			m := model.Buddy{
				FriendId:    characterId,
				FriendName:  characterName,
				Flag:        0,
				ChannelId:   channel.Id(channelId),
				FriendGroup: group,
			}
			w.WriteByteArray(m.Encoder(l, ctx)(options))
			w.WriteBool(inShop)
			return w.Bytes()
		}
	}
}

func BuddyErrorBody(errorCode string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getBuddyOperation(l)(options, errorCode))
			if errorCode == BuddyOperationErrorUnknownError {
				w.WriteByte(0)
			}
			return w.Bytes()
		}
	}
}

func BuddyChannelChangeBody(characterId uint32, channelId int8) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getBuddyOperation(l)(options, BuddyOperationBuddyChannelChange))
			w.WriteInt(characterId)
			w.WriteByte(0) // TODO m_aInShop
			w.WriteInt32(int32(channelId))
			return w.Bytes()
		}
	}
}

func BuddyCapacityUpdateBody(capacity byte) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getBuddyOperation(l)(options, BuddyOperationCapacityUpdate))
			w.WriteByte(capacity)
			return w.Bytes()
		}
	}
}

func getBuddyOperation(l logrus.FieldLogger) func(options map[string]interface{}, key string) byte {
	return func(options map[string]interface{}, key string) byte {
		var genericCodes interface{}
		var ok bool
		if genericCodes, ok = options["operations"]; !ok {
			l.Errorf("Code [%s] not configured for use. Defaulting to 99 which will likely cause a client crash.", key)
			return 99
		}

		var codes map[string]interface{}
		if codes, ok = genericCodes.(map[string]interface{}); !ok {
			l.Errorf("Code [%s] not configured for use. Defaulting to 99 which will likely cause a client crash.", key)
			return 99
		}

		var code interface{}
		if code, ok = codes[key]; !ok {
			l.Errorf("Code [%s] not configured for use. Defaulting to 99 which will likely cause a client crash.", key)
			return 99
		}

		op, err := strconv.ParseUint(code.(string), 0, 16)
		if err != nil {
			l.Errorf("Code [%s] not configured for use. Defaulting to 99 which will likely cause a client crash.", key)
			return 99
		}
		return byte(op)
	}
}
