package writer

import (
	"atlas-channel/character"
	"context"

	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	chatpkt "github.com/Chronicle20/atlas-packet/chat"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/sirupsen/logrus"
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

func CharacterChatWhisperFindResultInCashShopBody(mode WhisperMode, targetName string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return chatpkt.NewWhisperFindResultCashShop(byte(mode), targetName).Encode(l, ctx)
	}
}

func CharacterChatWhisperFindResultInMapBody(mode WhisperMode, target character.Model, mapId _map.Id) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		if mode == WhisperModeFindResult {
			return chatpkt.NewWhisperFindResultMapWithXY(byte(mode), target.Name(), uint32(mapId), target.X(), target.Y()).Encode(l, ctx)
		}
		return chatpkt.NewWhisperFindResultMap(byte(mode), target.Name(), uint32(mapId)).Encode(l, ctx)
	}
}

func CharacterChatWhisperFindResultInOtherChannelBody(mode WhisperMode, targetName string, channelId channel.Id) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return chatpkt.NewWhisperFindResultChannel(byte(mode), targetName, uint32(channelId)).Encode(l, ctx)
	}
}

func CharacterChatWhisperFindResultErrorBody(mode WhisperMode, targetName string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return chatpkt.NewWhisperFindResultError(byte(mode), targetName).Encode(l, ctx)
	}
}

func CharacterChatWhisperSendResultBody(target character.Model, success bool) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return chatpkt.NewWhisperSendResult(byte(WhisperModeSend), target.Name(), success).Encode(l, ctx)
	}
}

func CharacterChatWhisperSendFailureResultBody(targetName string, success bool) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return chatpkt.NewWhisperSendResult(byte(WhisperModeSend), targetName, success).Encode(l, ctx)
	}
}

func CharacterChatWhisperReceiptBody(from character.Model, channelId channel.Id, message string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return chatpkt.NewWhisperReceive(byte(WhisperModeReceive), from.Name(), byte(channelId), from.Gm(), message).Encode(l, ctx)
	}
}

func CharacterChatWhisperErrorBody(targetName string, whispersDisabled bool) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return chatpkt.NewWhisperError(byte(WhisperModeError), targetName, !whispersDisabled).Encode(l, ctx)
	}
}

func CharacterChatWhisperWeatherBody(fromName string, message string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return chatpkt.NewWhisperWeather(byte(WhisperModeWeather), fromName, message).Encode(l, ctx)
	}
}
