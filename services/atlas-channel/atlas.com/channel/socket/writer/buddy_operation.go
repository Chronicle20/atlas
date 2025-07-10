package writer

import (
	"atlas-channel/buddylist/buddy"
	"atlas-channel/socket/model"
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"strconv"
)

const (
	BuddyOperation                       = "BuddyOperation"
	BuddyOperationInvite                 = "INVITE"
	BuddyOperationUpdate                 = "UPDATE"
	BuddyOperationErrorListFull          = "BUDDY_LIST_FULL"
	BuddyOperationErrorOtherListFull     = "OTHER_BUDDY_LIST_FULL"
	BuddyOperationErrorAlreadyBuddy      = "ALREADY_BUDDY"
	BuddyOperationErrorCannotBuddyGm     = "CANNOT_BUDDY_GM"
	BuddyOperationErrorCharacterNotFound = "CHARACTER_NOT_FOUND"
	BuddyOperationErrorUnknownError      = "UNKNOWN_ERROR"
	BuddyOperationBuddyUpdate            = "BUDDY_UPDATE"
	BuddyOperationBuddyChannelChange     = "BUDDY_CHANNEL_CHANGE"
	BuddyOperationCapacityUpdate         = "CAPACITY_CHANGE"
)

func BuddyInviteBody(l logrus.FieldLogger, t tenant.Model) func(actorId uint32, originatorId uint32, originatorName string) BodyProducer {
	return func(actorId uint32, originatorId uint32, originatorName string) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getBuddyOperation(l)(options, BuddyOperationInvite))
			w.WriteInt(originatorId)
			w.WriteAsciiString(originatorName)

			b := model.Buddy{
				FriendId:    actorId,
				FriendName:  originatorName,
				Flag:        0,
				ChannelId:   channel.Id(0),
				FriendGroup: "Default Group",
			}
			b.Encode(l, t, options)(w)
			w.WriteByte(0) // 0 no, 1 true m_aInShop
			return w.Bytes()
		}
	}
}

func BuddyListUpdateBody(l logrus.FieldLogger, t tenant.Model) func(buddies []buddy.Model) BodyProducer {
	return func(buddies []buddy.Model) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
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
				m.Encode(l, t, options)(w)
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

func BuddyUpdateBody(l logrus.FieldLogger, t tenant.Model) func(characterId uint32, group string, characterName string, channelId int8, inShop bool) BodyProducer {
	return func(characterId uint32, group string, characterName string, channelId int8, inShop bool) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getBuddyOperation(l)(options, BuddyOperationBuddyUpdate))
			w.WriteInt(characterId)
			m := model.Buddy{
				FriendId:    characterId,
				FriendName:  characterName,
				Flag:        0,
				ChannelId:   channel.Id(channelId),
				FriendGroup: group,
			}
			m.Encode(l, t, options)(w)
			w.WriteBool(inShop)
			return w.Bytes()
		}
	}
}

func BuddyErrorBody(l logrus.FieldLogger) func(errorCode string) BodyProducer {
	return func(errorCode string) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getBuddyOperation(l)(options, errorCode))
			if errorCode == BuddyOperationErrorUnknownError {
				w.WriteByte(0)
			}
			return w.Bytes()
		}
	}
}

func BuddyChannelChangeBody(l logrus.FieldLogger) func(characterId uint32, channelId int8) BodyProducer {
	return func(characterId uint32, channelId int8) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getBuddyOperation(l)(options, BuddyOperationBuddyChannelChange))
			w.WriteInt(characterId)
			w.WriteByte(0) // TODO m_aInShop
			w.WriteInt32(int32(channelId))
			return w.Bytes()
		}
	}
}

func BuddyCapacityUpdateBody(l logrus.FieldLogger) func(capacity byte) BodyProducer {
	return func(capacity byte) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
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
