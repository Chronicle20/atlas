package writer

import (
	"atlas-channel/character"
	"context"
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
)

type MessengerOperationMode byte

const (
	MessengerOperation                   = "MessengerOperation"
	MessengerOperationModeAdd            = MessengerOperationMode(0)
	MessengerOperationModeJoin           = MessengerOperationMode(1)
	MessengerOperationModeRemove         = MessengerOperationMode(2)
	MessengerOperationModeRequestInvite  = MessengerOperationMode(3)
	MessengerOperationModeInviteSent     = MessengerOperationMode(4)
	MessengerOperationModeInviteDeclined = MessengerOperationMode(5)
	MessengerOperationModeChat           = MessengerOperationMode(6)
	MessengerOperationModeUpdate         = MessengerOperationMode(7)
)

func MessengerOperationAddBody(ctx context.Context) func(position byte, c character.Model, channelId channel.Id) BodyProducer {
	t := tenant.MustFromContext(ctx)
	return func(position byte, c character.Model, channelId channel.Id) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(byte(MessengerOperationModeAdd))
			w.WriteByte(position)
			WriteCharacterLook(t)(w, c, true)
			w.WriteAsciiString(c.Name())
			w.WriteByte(byte(channelId))
			w.WriteByte(0x00)
			return w.Bytes()
		}
	}
}

func MessengerOperationJoinBody(position byte) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteByte(byte(MessengerOperationModeJoin))
		w.WriteByte(position)
		return w.Bytes()
	}
}

func MessengerOperationRemoveBody(position byte) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteByte(byte(MessengerOperationModeRemove))
		w.WriteByte(position)
		return w.Bytes()
	}
}

func MessengerOperationInviteBody(fromName string, messengerId uint32) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteByte(byte(MessengerOperationModeRequestInvite))
		w.WriteAsciiString(fromName)
		w.WriteByte(0)
		w.WriteInt(messengerId)
		w.WriteByte(0)
		return w.Bytes()
	}
}

func MessengerOperationInviteSentBody(message string, success bool) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteByte(byte(MessengerOperationModeInviteSent))
		w.WriteAsciiString(message)
		w.WriteBool(success)
		return w.Bytes()
	}
}

func MessengerOperationInviteDeclinedBody(message string, mode byte) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteByte(byte(MessengerOperationModeInviteDeclined))
		w.WriteAsciiString(message)
		w.WriteByte(mode)
		return w.Bytes()
	}
}

func MessengerOperationChatBody(message string) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteByte(byte(MessengerOperationModeChat))
		w.WriteAsciiString(message)
		return w.Bytes()
	}
}

func MessengerOperationUpdateBody(ctx context.Context) func(position byte, c character.Model, channelId channel.Id) BodyProducer {
	t := tenant.MustFromContext(ctx)
	return func(position byte, c character.Model, channelId channel.Id) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(byte(MessengerOperationModeUpdate))
			w.WriteByte(position)
			WriteCharacterLook(t)(w, c, true)
			w.WriteAsciiString(c.Name())
			w.WriteByte(byte(channelId))
			w.WriteByte(0x00)
			return w.Bytes()
		}
	}
}
