package messenger

import (
	"context"

	atlas_packet "github.com/Chronicle20/atlas-packet"
	"github.com/Chronicle20/atlas-packet/model"
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

func MessengerOperationAddBody(position byte, avatar model.Avatar, name string, channelId byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MessengerOperationModeAdd, func(mode byte) packet.Encoder {
		return NewMessengerAdd(mode, position, avatar, name, channelId)
	})
}

func MessengerOperationJoinBody(position byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MessengerOperationModeJoin, func(mode byte) packet.Encoder {
		return NewMessengerJoin(mode, position)
	})
}

func MessengerOperationRemoveBody(position byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MessengerOperationModeRemove, func(mode byte) packet.Encoder {
		return NewMessengerRemove(mode, position)
	})
}

func MessengerOperationInviteBody(fromName string, messengerId uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MessengerOperationModeRequestInvite, func(mode byte) packet.Encoder {
		return NewMessengerRequestInvite(mode, fromName, messengerId)
	})
}

func MessengerOperationInviteSentBody(message string, success bool) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MessengerOperationModeInviteSent, func(mode byte) packet.Encoder {
		return NewMessengerInviteSent(mode, message, success)
	})
}

func MessengerOperationInviteDeclinedBody(message string, declineMode byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MessengerOperationModeInviteDeclined, func(mode byte) packet.Encoder {
		return NewMessengerInviteDeclined(mode, message, declineMode)
	})
}

func MessengerOperationChatBody(message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MessengerOperationModeChat, func(mode byte) packet.Encoder {
		return NewMessengerChat(mode, message)
	})
}

func MessengerOperationUpdateBody(position byte, avatar model.Avatar, name string, channelId byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MessengerOperationModeUpdate, func(mode byte) packet.Encoder {
		return NewMessengerUpdate(mode, position, avatar, name, channelId)
	})
}
