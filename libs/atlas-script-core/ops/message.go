package ops

import saga "github.com/Chronicle20/atlas/libs/atlas-saga"

const opSendMessage = "send_message"

// SendMessage builds a SendMessage step. It backs both the `send_message`
// (npc-conversations) and `drop_message` (map/reactor/portal actions) script
// operations — FR-14 keeps both dispatch names valid.
//
// Parameters:
//   - message      (required) the message text.
//   - messageType  (optional) one of "NOTICE", "POP_UP", "PINK_TEXT",
//     "BLUE_TEXT". `type` is accepted as an alias; `messageType` wins when both
//     are present. Numeric "5" maps to "PINK_TEXT" and "6" to "BLUE_TEXT" for
//     either key (carried forward from reactor-actions). Defaults to
//     "PINK_TEXT" when absent.
func SendMessage(p map[string]string, r Resolver, t Target, characterId uint32) (Step, error) {
	message, err := requiredString(p, r, characterId, opSendMessage, "message")
	if err != nil {
		return Step{}, err
	}

	messageType := "PINK_TEXT"
	key := ""
	if _, ok := p["messageType"]; ok {
		key = "messageType"
	} else if _, ok := p["type"]; ok {
		key = "type"
	}
	if key != "" {
		raw, err := requiredString(p, r, characterId, opSendMessage, key)
		if err != nil {
			return Step{}, err
		}
		switch raw {
		case "5":
			messageType = "PINK_TEXT"
		case "6":
			messageType = "BLUE_TEXT"
		default:
			messageType = raw
		}
	}

	return newStep(saga.SendMessage, saga.SendMessagePayload{
		CharacterId: characterId,
		WorldId:     t.Field().WorldId(),
		ChannelId:   t.Field().ChannelId(),
		MessageType: messageType,
		Message:     message,
	}), nil
}
