package writer

import (
	"atlas-channel/character"
	"atlas-channel/socket/model"
	"context"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const (
	MessengerOperation                   = "MessengerOperation"
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
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getMessengerOperation(l)(options, MessengerOperationModeAdd))
			w.WriteByte(position)
			ava := model.NewFromCharacter(c, true)
			w.WriteByteArray(ava.Encoder(l, ctx)(options))
			w.WriteAsciiString(c.Name())
			w.WriteByte(byte(channelId))
			w.WriteByte(0x00)
			return w.Bytes()
		}
	}
}

func MessengerOperationJoinBody(position byte) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getMessengerOperation(l)(options, MessengerOperationModeJoin))
			w.WriteByte(position)
			return w.Bytes()
		}
	}
}

func MessengerOperationRemoveBody(position byte) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getMessengerOperation(l)(options, MessengerOperationModeRemove))
			w.WriteByte(position)
			return w.Bytes()
		}
	}
}

func MessengerOperationInviteBody(fromName string, messengerId uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getMessengerOperation(l)(options, MessengerOperationModeRequestInvite))
			w.WriteAsciiString(fromName)
			w.WriteByte(0)
			w.WriteInt(messengerId)
			w.WriteByte(0)
			return w.Bytes()
		}
	}
}

func MessengerOperationInviteSentBody(message string, success bool) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getMessengerOperation(l)(options, MessengerOperationModeInviteSent))
			w.WriteAsciiString(message)
			w.WriteBool(success)
			return w.Bytes()
		}
	}
}

func MessengerOperationInviteDeclinedBody(message string, mode byte) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getMessengerOperation(l)(options, MessengerOperationModeInviteDeclined))
			w.WriteAsciiString(message)
			w.WriteByte(mode)
			return w.Bytes()
		}
	}
}

func MessengerOperationChatBody(message string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getMessengerOperation(l)(options, MessengerOperationModeChat))
			w.WriteAsciiString(message)
			return w.Bytes()
		}
	}
}

func MessengerOperationUpdateBody(position byte, c character.Model, channelId channel.Id) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getMessengerOperation(l)(options, MessengerOperationModeUpdate))
			w.WriteByte(position)
			ava := model.NewFromCharacter(c, true)
			w.WriteByteArray(ava.Encoder(l, ctx)(options))
			w.WriteAsciiString(c.Name())
			w.WriteByte(byte(channelId))
			w.WriteByte(0x00)
			return w.Bytes()
		}
	}
}

func getMessengerOperation(l logrus.FieldLogger) func(options map[string]interface{}, key string) byte {
	return func(options map[string]interface{}, key string) byte {
		var genericCodes interface{}
		var ok bool
		if genericCodes, ok = options["operations"]; !ok {
			l.Errorf("Code [%s] not configured for use.", key)
			return 0
		}

		var codes map[string]interface{}
		if codes, ok = genericCodes.(map[string]interface{}); !ok {
			l.Errorf("Code [%s] not configured for use.", key)
			return 0
		}

		res, ok := codes[key].(float64)
		if !ok {
			l.Errorf("Code [%s] not configured for use.", key)
			return 0
		}
		return byte(res)
	}
}
