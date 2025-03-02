package writer

import (
	"atlas-channel/character"
	"github.com/Chronicle20/atlas-socket/response"
)

const CharacterChatWhisper = "CharacterChatWhisper"

type WhisperMode byte

const (
	WhisperModeSend    = WhisperMode(0x0A)
	WhisperModeReceive = WhisperMode(0x12)
)

func CharacterChatWhisperSendResultBody(target character.Model, success bool) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteByte(byte(WhisperModeSend))
		w.WriteAsciiString(target.Name())
		w.WriteBool(success)
		return w.Bytes()
	}
}

func CharacterChatWhisperReceiptBody(from character.Model, channelId byte, message string) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteByte(byte(WhisperModeReceive))
		w.WriteAsciiString(from.Name())
		w.WriteByte(channelId)
		w.WriteBool(from.Gm())
		w.WriteAsciiString(message)
		return w.Bytes()
	}
}
