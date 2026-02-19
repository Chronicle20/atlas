package writer

import (
	"fmt"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type WorldMessageMode string

const (
	// WorldMessage CWvsContext::OnBroadcastMsg
	WorldMessage = "WorldMessage"

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

func WorldMessageNoticeBody(l logrus.FieldLogger) func(message string) BodyProducer {
	return func(message string) BodyProducer {
		return WorldMessageBody(l)(WorldMessageNotice, []string{message}, 0, false, "", NoOpOperator)
	}
}

func WorldMessagePopUpBody(l logrus.FieldLogger) func(message string) BodyProducer {
	return func(message string) BodyProducer {
		return WorldMessageBody(l)(WorldMessagePopUp, []string{message}, 0, false, "", NoOpOperator)
	}
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

func WorldMessageMegaphoneBody(l logrus.FieldLogger) func(medal string, characterName string, message string) BodyProducer {
	return func(medal string, characterName string, message string) BodyProducer {
		actualMessage := decorateMegaphoneMessage(medal, characterName, message)
		return WorldMessageBody(l)(WorldMessageMegaphone, []string{actualMessage}, 0, false, "", NoOpOperator)
	}
}

func WorldMessageSuperMegaphoneBody(l logrus.FieldLogger) func(medal string, characterName string, message string, channelId channel.Id, whispersOn bool) BodyProducer {
	return func(medal string, characterName string, message string, channelId channel.Id, whispersOn bool) BodyProducer {
		actualMessage := decorateMegaphoneMessage(medal, characterName, message)
		return WorldMessageBody(l)(WorldMessageSuperMegaphone, []string{actualMessage}, channelId, whispersOn, "", NoOpOperator)
	}
}

func WorldMessageTopScrollBody(l logrus.FieldLogger) func(message string) BodyProducer {
	return func(message string) BodyProducer {
		return WorldMessageBody(l)(WorldMessageTopScroll, []string{message}, 0, false, "", NoOpOperator)
	}
}

func WorldMessageClearTopScrollBody(l logrus.FieldLogger) func() BodyProducer {
	return func() BodyProducer {
		return WorldMessageBody(l)(WorldMessageTopScroll, []string{""}, 0, false, "", NoOpOperator)
	}
}

func WorldMessagePinkTextBody(l logrus.FieldLogger) func(medal string, characterName string, message string) BodyProducer {
	return func(medal string, characterName string, message string) BodyProducer {
		actualMessage := decorateMegaphoneMessage(medal, characterName, message)
		return WorldMessageBody(l)(WorldMessagePinkText, []string{actualMessage}, 0, false, "", NoOpOperator)
	}
}

func WorldMessageBlueTextBody(l logrus.FieldLogger) func(medal string, characterName string, message string) BodyProducer {
	return func(medal string, characterName string, message string) BodyProducer {
		actualMessage := decorateMegaphoneMessage(medal, characterName, message)
		return WorldMessageBody(l)(WorldMessageBlueText, []string{actualMessage}, 0, false, "", ItemIdOperator(0))
	}
}

func WorldMessageBlueTextWithItemBody(l logrus.FieldLogger) func(medal string, characterName string, message string, itemId uint32) BodyProducer {
	return func(medal string, characterName string, message string, itemId uint32) BodyProducer {
		actualMessage := decorateMegaphoneMessage(medal, characterName, message)
		return WorldMessageBody(l)(WorldMessageBlueText, []string{actualMessage}, 0, false, "", ItemIdOperator(itemId))
	}
}

func WorldMessageNPCBody(l logrus.FieldLogger) func(message string, npcId uint32) BodyProducer {
	return func(message string, npcId uint32) BodyProducer {
		return WorldMessageBody(l)(WorldMessageBlueText, []string{message}, 0, false, "", NPCIdOperator(l)(npcId))
	}
}

func WorldMessageItemMegaphoneBody(l logrus.FieldLogger) func(medal string, characterName string, message string, channelId channel.Id, operator model.Operator[*response.Writer], whispersOn bool) BodyProducer {
	return func(medal string, characterName string, message string, channelId channel.Id, operator model.Operator[*response.Writer], whispersOn bool) BodyProducer {
		actualMessage := decorateMegaphoneMessage(medal, characterName, message)
		return WorldMessageBody(l)(WorldMessageItemMegaphone, []string{actualMessage}, channelId, whispersOn, "", operator)
	}
}

func WorldMessageYellowMegaphoneBody(l logrus.FieldLogger) func(medal string, characterName string, message string, channelId channel.Id) BodyProducer {
	return func(medal string, characterName string, message string, channelId channel.Id) BodyProducer {
		actualMessage := decorateMegaphoneMessage(medal, characterName, message)
		return WorldMessageBody(l)(WorldMessageYellowMegaphone, []string{actualMessage}, channelId, false, "", NoOpOperator)
	}
}

func WorldMessageMultiMegaphoneBody(l logrus.FieldLogger) func(medal string, characterName string, messages []string, channelId channel.Id, whispersOn bool) BodyProducer {
	return func(medal string, characterName string, messages []string, channelId channel.Id, whispersOn bool) BodyProducer {
		actualMessages := make([]string, 0)
		for _, m := range messages {
			actualMessages = append(actualMessages, decorateMegaphoneMessage(medal, characterName, m))
		}
		return WorldMessageBody(l)(WorldMessageMultiMegaphone, actualMessages, channelId, whispersOn, "", NoOpOperator)
	}
}

func WorldMessageGachaponMegaphoneBody(l logrus.FieldLogger) func(medal string, characterName string, channelId channel.Id, townName string, operator model.Operator[*response.Writer]) BodyProducer {
	return func(medal string, characterName string, channelId channel.Id, townName string, operator model.Operator[*response.Writer]) BodyProducer {
		actualMessage := decorateNameForMessage(medal, characterName)
		return WorldMessageBody(l)(WorldMessageGachapon, []string{actualMessage}, channelId, false, townName, operator)
	}
}

func NoOpOperator(_ *response.Writer) error {
	return nil
}

func ItemIdOperator(itemId uint32) func(w *response.Writer) error {
	return func(w *response.Writer) error {
		w.WriteInt(itemId)
		return nil
	}
}

func SlotAsIntOperator(slot int16) model.Operator[*response.Writer] {
	return func(w *response.Writer) error {
		w.WriteInt32(int32(slot))
		return nil
	}
}

func NPCIdOperator(l logrus.FieldLogger) func(npcId uint32) model.Operator[*response.Writer] {
	return func(npcId uint32) model.Operator[*response.Writer] {
		return func(w *response.Writer) error {
			if npcId == 0 {
				l.Warnf("NPC should be provided for NPC mode.")
				w.WriteInt(9010000)
			} else {
				w.WriteInt(npcId)
			}
			return nil
		}
	}
}

func WorldMessageBody(l logrus.FieldLogger) func(mode WorldMessageMode, messages []string, channel channel.Id, whispersOn bool, townName string, operator model.Operator[*response.Writer]) BodyProducer {
	return func(mode WorldMessageMode, messages []string, channel channel.Id, whispersOn bool, townName string, operator model.Operator[*response.Writer]) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			weatherItemId := uint32(0)

			if len(messages) > 1 && mode != WorldMessageMultiMegaphone {
				l.Warnf("Client will only relay a maximum of 1 message in this mode.")
			} else if len(messages) > 3 {
				l.Warnf("Client will only relay a maximum of 3 messages in a multi megaphone.")
			}

			modeByte := getWorldMessageMode(l)(options, mode)
			w.WriteByte(modeByte)
			if mode == WorldMessageTopScroll {
				if len(messages[0]) == 0 {
					w.WriteBool(false)
				} else {
					w.WriteBool(true)
				}
			}
			w.WriteAsciiString(messages[0])
			if mode == WorldMessageSuperMegaphone {
				w.WriteByte(byte(channel))
				w.WriteBool(whispersOn)
			} else if mode == WorldMessageBlueText {
				_ = operator(w)
			} else if mode == WorldMessageNPC {
				_ = operator(w)
			} else if mode == WorldMessageItemMegaphone {
				w.WriteByte(byte(channel))
				w.WriteBool(whispersOn)
				w.WriteBool(true)
				_ = operator(w)
			} else if mode == WorldMessageYellowMegaphone {
				w.WriteByte(byte(channel))
			} else if mode == WorldMessageMultiMegaphone {
				w.WriteByte(byte(len(messages)))
				for _, m := range messages[1:] {
					w.WriteAsciiString(m)
				}
				w.WriteByte(byte(channel))
				w.WriteBool(whispersOn)
			} else if mode == WorldMessageWeather {
				w.WriteInt(weatherItemId)
			} else if mode == WorldMessageGachapon {
				w.WriteInt(0)                // ?
				w.WriteAsciiString(townName) // town name
				_ = operator(w)
			} else if mode == WorldMessageUnk3 || mode == WorldMessageUnk4 {
				w.WriteAsciiString("doo") // character name
				_ = operator(w)
			} else if mode == WorldMessageUnk7 {
				w.WriteInt(0) // item id
			} else if mode == WorldMessageUnk8 {
				w.WriteByte(byte(channel))
				w.WriteBool(whispersOn)
			}
			return w.Bytes()
		}
	}
}

func getWorldMessageMode(l logrus.FieldLogger) func(options map[string]interface{}, key WorldMessageMode) byte {
	return func(options map[string]interface{}, key WorldMessageMode) byte {
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

		op, ok := codes[string(key)].(float64)
		if !ok {
			l.Errorf("Code [%s] not configured for use. Defaulting to 99 which will likely cause a client crash.", key)
			return 99
		}
		return byte(op)
	}
}
