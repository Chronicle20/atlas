package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type UiWindow string

const (
	UiOpen                                = "UiOpen"
	UiWindowItem                 UiWindow = "ITEM"                  // 0
	UiWindowEquipment            UiWindow = "EQUIPMENT"             // 1
	UiWindowStatistics           UiWindow = "STATISTICS"            // 2
	UiWindowSkills               UiWindow = "SKILLS"                // 3
	UiWindowKeyboard             UiWindow = "KEYBOARD"              // 5
	UiWindowQuest                UiWindow = "QUEST"                 // 6
	UiWindowMonsterBook          UiWindow = "MONSTER_BOOK"          // 9
	UiWindowCharacterInformation UiWindow = "CHARACTER_INFORMATION" // 10
	UiWindowGuildBbs             UiWindow = "GUILD_BBS"             // 11
	UiWindowMonsterCarnival      UiWindow = "MONSTER_CARNIVAL"      // 18
	UiWindowEnergyBar            UiWindow = "ENERGY_BAR"            // 20
	UiWindowPartySearch          UiWindow = "PARTY_SEARCH"          // 22
	UiWindowItemMaker            UiWindow = "ITEM_MAKER"            // 23
	UiWindowRanking              UiWindow = "RANKING"               // 26
	UiWindowFamily               UiWindow = "FAMILY"                // 27
	UiWindowFamilyPedigree       UiWindow = "FAMILY_PEDIGREE"       // 28
	UiWindowOperatorBoard        UiWindow = "OPERATOR_BOARD"        // 29
	UiWindowOperatorBoardState   UiWindow = "OPERATOR_BOARD_STATE"  // 30
	UiWindowMedalQuest           UiWindow = "MEDAL_MEDAL_QUEST"     // 31
	UiWindowWebEvent             UiWindow = "WEB_EVENT"             // 32
	UiWindowSkillsEx             UiWindow = "SKILLS_EX"             // 33
)

func UiOpenBody(l logrus.FieldLogger) func(window UiWindow) BodyProducer {
	return func(window UiWindow) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getUiWindowMode(l)(options, window))
			return w.Bytes()
		}
	}
}

func getUiWindowMode(l logrus.FieldLogger) func(options map[string]interface{}, key UiWindow) byte {
	return func(options map[string]interface{}, key UiWindow) byte {
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
