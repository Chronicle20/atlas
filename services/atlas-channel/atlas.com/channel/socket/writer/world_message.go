package writer

import (
	"atlas-channel/character/inventory/item"
	"atlas-channel/npc"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"strconv"
)

type WorldMessageMode string

const (
	WorldMessage = "WorldMessage"

	WorldMessageNotice         = WorldMessageMode("NOTICE")
	WorldMessagePopUp          = WorldMessageMode("POP_UP")
	WorldMessageMegaphone      = WorldMessageMode("MEGAPHONE")
	WorldMessageSuperMegaphone = WorldMessageMode("SUPER_MEGAPHONE")
	WorldMessageTopScroll      = WorldMessageMode("TOP_SCROLL")
	WorldMessagePinkText       = WorldMessageMode("PINK_TEXT")
	WorldMessageBlueText       = WorldMessageMode("BLUE_TEXT")
	WorldMessageNPC            = WorldMessageMode("NPC")
	WorldMessageItemMegaphone  = WorldMessageMode("ITEM_MEGAPHONE")
	WorldMessageUnk2           = WorldMessageMode("UNKNOWN_2")
	WorldMessageMultiMegaphone = WorldMessageMode("MULTI_MEGAPHONE")
	WorldMessageWeather        = WorldMessageMode("WEATHER")
	WorldMessageGachapon       = WorldMessageMode("GACHAPON")
	WorldMessageUnk3           = WorldMessageMode("UNKNOWN_3")
	WorldMessageUnk4           = WorldMessageMode("UNKNOWN_4")
	WorldMessageUnk5           = WorldMessageMode("UNKNOWN_5")
	WorldMessageUnk6           = WorldMessageMode("UNKNOWN_6")
	WorldMessageUnk7           = WorldMessageMode("UNKNOWN_7")
	WorldMessageUnk8           = WorldMessageMode("UNKNOWN_8") // present in v95+
)

func WorldMessageBody(l logrus.FieldLogger, t tenant.Model) func(mode WorldMessageMode, messages []string, channel byte, npc *npc.Model, i *item.Model) BodyProducer {
	return func(mode WorldMessageMode, messages []string, channel byte, npc *npc.Model, i *item.Model) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			clearTopScroll := true
			whisperIcon := false
			weatherItemId := uint32(0)

			if len(messages) > 1 && mode != WorldMessageMultiMegaphone {
				l.Warnf("Client will only relay a maximum of 1 message in this mode.")
			} else if len(messages) > 3 {
				l.Warnf("Client will only relay a maximum of 3 messages in a multi megaphone.")
			}

			modeByte := getWorldMessageMode(l)(options, mode)
			w.WriteByte(modeByte)
			if mode == WorldMessageTopScroll {
				w.WriteBool(!clearTopScroll)
			}
			w.WriteAsciiString(messages[0])
			if mode == WorldMessageSuperMegaphone {
				w.WriteByte(channel)
				w.WriteBool(whisperIcon)
			} else if mode == WorldMessageBlueText {
				if i == nil {
					w.WriteInt(0)
				} else {
					w.WriteInt(uint32(i.Slot()))
				}
			} else if mode == WorldMessageNPC {
				if npc == nil {
					l.Warnf("NPC should be provided for NPC mode.")
					w.WriteInt(0)
				} else {
					w.WriteInt(npc.Id())
				}
			} else if mode == WorldMessageItemMegaphone {
				w.WriteByte(channel)
				w.WriteBool(whisperIcon)
				if i == nil {
					w.WriteByte(0)
				} else {
					w.WriteByte(1)
					_ = WriteItemInfo(t)(w, true)(*i)
				}
			} else if mode == WorldMessageUnk2 {
				w.WriteByte(channel)
			} else if mode == WorldMessageMultiMegaphone {
				w.WriteByte(byte(len(messages)))
				for _, m := range messages[1:] {
					w.WriteAsciiString(m)
				}
				w.WriteByte(channel)
				w.WriteBool(whisperIcon)
			} else if mode == WorldMessageWeather {
				w.WriteInt(weatherItemId)
			} else if mode == WorldMessageGachapon {
				w.WriteInt(0)                 // ?
				w.WriteAsciiString("Henesys") // town name
				if i == nil {
					l.Warnf("Received item should be provided for Gachapon mode.")
					_ = WriteItemInfo(t)(w, true)(item.Model{})
				} else {
					_ = WriteItemInfo(t)(w, true)(*i)
				}
			} else if mode == WorldMessageUnk3 || mode == WorldMessageUnk4 {
				w.WriteAsciiString("") // character name
				if i == nil {
					l.Warnf("Received item should be provided for Gachapon mode.")
					_ = WriteItemInfo(t)(w, true)(item.Model{})
				} else {
					_ = WriteItemInfo(t)(w, true)(*i)
				}
			} else if mode == WorldMessageUnk7 {
				w.WriteInt(0) // item id
			} else if mode == WorldMessageUnk8 {
				w.WriteByte(channel)
				w.WriteBool(whisperIcon)
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

		var code interface{}
		if code, ok = codes[string(key)]; !ok {
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
