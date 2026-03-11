package writer

import (
	"atlas-channel/character"
	"atlas-channel/socket/model"
	"context"

	"github.com/Chronicle20/atlas-constants/channel"
	atlas_packet "github.com/Chronicle20/atlas-packet"
	messengerpkt "github.com/Chronicle20/atlas-packet/messenger"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

const (
	MessengerOperationModeAdd            = "ADD"
	MessengerOperationModeJoin           = "JOIN"
	MessengerOperationModeRemove         = "REMOVE"
	MessengerOperationModeRequestInvite  = "REQUEST_INVITE"
	MessengerOperationModeInviteSent     = "INVITE_SENT"
	MessengerOperationModeInviteDeclined = "INVITE_DECLINED"
	MessengerOperationModeChat           = "CHAT"
	MessengerOperationModeUpdate         = "UPDATE"
)

func MessengerOperationAddBody(position byte, c character.Model, channelId channel.Id) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getMessengerOperation(l)(options, MessengerOperationModeAdd)
			ava := model.NewFromCharacter(c, true)
			return messengerpkt.NewMessengerAdd(mode, position, ava, c.Name(), byte(channelId)).Encode(l, ctx)(options)
		}
	}
}

func MessengerOperationJoinBody(position byte) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getMessengerOperation(l)(options, MessengerOperationModeJoin)
			return messengerpkt.NewMessengerJoin(mode, position).Encode(l, ctx)(options)
		}
	}
}

func MessengerOperationRemoveBody(position byte) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getMessengerOperation(l)(options, MessengerOperationModeRemove)
			return messengerpkt.NewMessengerRemove(mode, position).Encode(l, ctx)(options)
		}
	}
}

func MessengerOperationInviteBody(fromName string, messengerId uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getMessengerOperation(l)(options, MessengerOperationModeRequestInvite)
			return messengerpkt.NewMessengerRequestInvite(mode, fromName, messengerId).Encode(l, ctx)(options)
		}
	}
}

func MessengerOperationInviteSentBody(message string, success bool) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getMessengerOperation(l)(options, MessengerOperationModeInviteSent)
			return messengerpkt.NewMessengerInviteSent(mode, message, success).Encode(l, ctx)(options)
		}
	}
}

func MessengerOperationInviteDeclinedBody(message string, mode byte) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			opMode := getMessengerOperation(l)(options, MessengerOperationModeInviteDeclined)
			return messengerpkt.NewMessengerInviteDeclined(opMode, message, mode).Encode(l, ctx)(options)
		}
	}
}

func MessengerOperationChatBody(message string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getMessengerOperation(l)(options, MessengerOperationModeChat)
			return messengerpkt.NewMessengerChat(mode, message).Encode(l, ctx)(options)
		}
	}
}

func MessengerOperationUpdateBody(position byte, c character.Model, channelId channel.Id) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getMessengerOperation(l)(options, MessengerOperationModeUpdate)
			ava := model.NewFromCharacter(c, true)
			return messengerpkt.NewMessengerUpdate(mode, position, ava, c.Name(), byte(channelId)).Encode(l, ctx)(options)
		}
	}
}

func getMessengerOperation(l logrus.FieldLogger) func(options map[string]interface{}, key string) byte {
	return func(options map[string]interface{}, key string) byte {
		return atlas_packet.ResolveCode(l, options, "operations", key)
	}
}
