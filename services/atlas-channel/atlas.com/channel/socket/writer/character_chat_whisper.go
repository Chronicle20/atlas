package writer

import (
	"atlas-channel/character"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-socket/response"
)

const CharacterChatWhisper = "CharacterChatWhisper"

type WhisperMode byte

type WhisperFindResultMode byte

const (
	WhisperModeSend                  = WhisperMode(0x0A)
	WhisperModeReceive               = WhisperMode(0x12)
	WhisperModeFindResult            = WhisperMode(0x09)
	WhisperModeBuddyWindowFindResult = WhisperMode(0x48)
	WhisperModeUnk1                  = WhisperMode(0x8A)
	WhisperModeError                 = WhisperMode(0x22)
	WhisperModeWeather               = WhisperMode(0x92)

	WhisperFindResultModeError            = WhisperFindResultMode(0)
	WhisperFindResultModeMap              = WhisperFindResultMode(1)
	WhisperFindResultModeCashShop         = WhisperFindResultMode(2)
	WhisperFindResultModeDifferentChannel = WhisperFindResultMode(3)
	WhisperFindResultModeUnable2          = WhisperFindResultMode(4)
)

func CharacterChatWhisperFindResultInCashShopBody(mode WhisperMode, targetName string) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteByte(byte(mode))
		w.WriteAsciiString(targetName)
		w.WriteByte(byte(WhisperFindResultModeCashShop))
		w.WriteInt32(-1)
		return w.Bytes()
	}
}

func CharacterChatWhisperFindResultInMapBody(mode WhisperMode, target character.Model, mapId _map.Id) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteByte(byte(mode))
		w.WriteAsciiString(target.Name())
		w.WriteByte(byte(WhisperFindResultModeMap))
		w.WriteInt(uint32(mapId))
		if mode == WhisperModeFindResult {
			w.WriteInt32(int32(target.X()))
			w.WriteInt32(int32(target.Y()))
		}
		return w.Bytes()
	}
}

func CharacterChatWhisperFindResultInOtherChannelBody(mode WhisperMode, targetName string, channelId channel.Id) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteByte(byte(mode))
		w.WriteAsciiString(targetName)
		w.WriteByte(byte(WhisperFindResultModeDifferentChannel))
		w.WriteInt(uint32(channelId))
		return w.Bytes()
	}
}

func CharacterChatWhisperFindResultErrorBody(mode WhisperMode, targetName string) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteByte(byte(mode))
		w.WriteAsciiString(targetName)
		w.WriteByte(byte(WhisperFindResultModeError))
		w.WriteInt(0)
		return w.Bytes()
	}
}

func CharacterChatWhisperSendResultBody(target character.Model, success bool) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteByte(byte(WhisperModeSend))
		w.WriteAsciiString(target.Name())
		w.WriteBool(success)
		return w.Bytes()
	}
}

func CharacterChatWhisperSendFailureResultBody(targetName string, success bool) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteByte(byte(WhisperModeSend))
		w.WriteAsciiString(targetName)
		w.WriteBool(success)
		return w.Bytes()
	}
}

func CharacterChatWhisperReceiptBody(from character.Model, channelId channel.Id, message string) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteByte(byte(WhisperModeReceive))
		w.WriteAsciiString(from.Name())
		w.WriteByte(byte(channelId))
		w.WriteBool(from.Gm())
		w.WriteAsciiString(message)
		return w.Bytes()
	}
}

func CharacterChatWhisperErrorBody(targetName string, whispersDisabled bool) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteByte(byte(WhisperModeError))
		w.WriteAsciiString(targetName)
		w.WriteBool(!whispersDisabled)
		return w.Bytes()
	}
}

func CharacterChatWhisperWeatherBody(fromName string, message string) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteByte(byte(WhisperModeWeather))
		w.WriteAsciiString(fromName)
		w.WriteBool(true)
		w.WriteAsciiString(message)
		return w.Bytes()
	}
}
