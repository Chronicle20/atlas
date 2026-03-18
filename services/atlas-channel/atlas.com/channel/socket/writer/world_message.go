package writer

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-constants/channel"
	atlas_packet "github.com/Chronicle20/atlas-packet"
	chatpkt "github.com/Chronicle20/atlas-packet/chat/clientbound"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

type WorldMessageMode string

const (
	// WorldMessage CWvsContext::OnBroadcastMsg

	WorldMessageNotice           = WorldMessageMode("NOTICE")
	WorldMessagePopUp            = WorldMessageMode("POP_UP")
	WorldMessageMegaphone        = WorldMessageMode("MEGAPHONE")
	WorldMessageSuperMegaphone   = WorldMessageMode("SUPER_MEGAPHONE")
	WorldMessageTopScroll        = WorldMessageMode("TOP_SCROLL")
	WorldMessagePinkText         = WorldMessageMode("PINK_TEXT")
	WorldMessageBlueText         = WorldMessageMode("BLUE_TEXT")
	WorldMessageNPC              = WorldMessageMode("NPC")
	WorldMessageItemMegaphone    = WorldMessageMode("ITEM_MEGAPHONE")
	WorldMessageYellowMegaphone  = WorldMessageMode("YELLOW_MEGAPHONE")
	WorldMessageMultiMegaphone   = WorldMessageMode("MULTI_MEGAPHONE")
	WorldMessageWeather          = WorldMessageMode("WEATHER")
	WorldMessageGachapon         = WorldMessageMode("GACHAPON")
	WorldMessageUnk3             = WorldMessageMode("UNKNOWN_3")
	WorldMessageUnk4             = WorldMessageMode("UNKNOWN_4")
	WorldMessageClipboardNotice1 = WorldMessageMode("CLIPBOARD_NOTICE_1")
	WorldMessageClipboardNotice2 = WorldMessageMode("CLIPBOARD_NOTICE_2")
	WorldMessageUnk7             = WorldMessageMode("UNKNOWN_7")
	WorldMessageUnk8             = WorldMessageMode("UNKNOWN_8") // present in v95+
)

func WorldMessageNoticeBody(message string) packet.Encode {
	return worldMessageBody(WorldMessageNotice, []string{message}, 0, false, "", 0, 0)
}

func WorldMessagePopUpBody(message string) packet.Encode {
	return worldMessageBody(WorldMessagePopUp, []string{message}, 0, false, "", 0, 0)
}

func decorateNameForMessage(medal string, characterName string) string {
	if len(medal) == 0 {
		return characterName
	}
	return fmt.Sprintf("<%s> %s", medal, characterName)
}

func decorateMegaphoneMessage(medal string, characterName string, message string) string {
	name := decorateNameForMessage(medal, characterName)
	if len(name) == 0 {
		return message
	}
	return fmt.Sprintf("%s : %s", name, message)
}

func WorldMessageTopScrollBody(message string) packet.Encode {
	return worldMessageBody(WorldMessageTopScroll, []string{message}, 0, false, "", 0, 0)
}

func WorldMessagePinkTextBody(medal string, characterName string, message string) packet.Encode {
	actualMessage := decorateMegaphoneMessage(medal, characterName, message)
	return worldMessageBody(WorldMessagePinkText, []string{actualMessage}, 0, false, "", 0, 0)
}

func WorldMessageBlueTextBody(medal string, characterName string, message string) packet.Encode {
	actualMessage := decorateMegaphoneMessage(medal, characterName, message)
	return worldMessageBody(WorldMessageBlueText, []string{actualMessage}, 0, false, "", 0, 0)
}

func WorldMessageGachaponMegaphoneBody(medal string, characterName string, channelId channel.Id, townName string, itemId uint32) packet.Encode {
	actualMessage := decorateNameForMessage(medal, characterName)
	return worldMessageBody(WorldMessageGachapon, []string{actualMessage}, channelId, false, townName, itemId, 0)
}

func worldMessageBody(mode WorldMessageMode, messages []string, channel channel.Id, whispersOn bool, townName string, itemId uint32, slot int32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			modeByte := getWorldMessageMode(l)(options, mode)

			switch mode {
			case WorldMessageNotice, WorldMessagePopUp, WorldMessageMegaphone, WorldMessagePinkText,
				WorldMessageClipboardNotice1, WorldMessageClipboardNotice2:
				return chatpkt.NewWorldMessageSimple(modeByte, messages[0]).Encode(l, ctx)(options)
			case WorldMessageTopScroll:
				return chatpkt.NewWorldMessageTopScroll(modeByte, messages[0]).Encode(l, ctx)(options)
			case WorldMessageSuperMegaphone:
				return chatpkt.NewWorldMessageSuperMegaphone(modeByte, messages[0], byte(channel), whispersOn).Encode(l, ctx)(options)
			case WorldMessageBlueText, WorldMessageNPC:
				return chatpkt.NewWorldMessageBlueText(modeByte, messages[0], itemId).Encode(l, ctx)(options)
			case WorldMessageItemMegaphone:
				return chatpkt.NewWorldMessageItemMegaphone(modeByte, messages[0], byte(channel), whispersOn, slot).Encode(l, ctx)(options)
			case WorldMessageYellowMegaphone:
				return chatpkt.NewWorldMessageYellowMegaphone(modeByte, messages[0], byte(channel)).Encode(l, ctx)(options)
			case WorldMessageMultiMegaphone:
				if len(messages) > 3 {
					l.Warnf("Client will only relay a maximum of 3 messages in a multi megaphone.")
				}
				return chatpkt.NewWorldMessageMultiMegaphone(modeByte, messages, byte(channel), whispersOn).Encode(l, ctx)(options)
			case WorldMessageGachapon:
				return chatpkt.NewWorldMessageGachapon(modeByte, messages[0], townName, itemId).Encode(l, ctx)(options)
			case WorldMessageWeather:
				return chatpkt.NewWorldMessageWeather(modeByte, messages[0], 0).Encode(l, ctx)(options)
			case WorldMessageUnk3, WorldMessageUnk4:
				return chatpkt.NewWorldMessageUnknown3(modeByte, messages[0], itemId).Encode(l, ctx)(options)
			case WorldMessageUnk7:
				return chatpkt.NewWorldMessageUnknown7(modeByte, messages[0]).Encode(l, ctx)(options)
			case WorldMessageUnk8:
				return chatpkt.NewWorldMessageUnknown8(modeByte, messages[0], byte(channel), whispersOn).Encode(l, ctx)(options)
			default:
				l.Warnf("Unhandled world message mode [%s].", mode)
				return nil
			}
		}
	}
}

func getWorldMessageMode(l logrus.FieldLogger) func(options map[string]interface{}, key WorldMessageMode) byte {
	return func(options map[string]interface{}, key WorldMessageMode) byte {
		return atlas_packet.ResolveCode(l, options, "operations", string(key))
	}
}
