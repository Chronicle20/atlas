package writer

import (
	"context"

	atlas_packet "github.com/Chronicle20/atlas-packet"
	uipkt "github.com/Chronicle20/atlas-packet/ui"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

type UiWindow string

const (
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

func UiOpenBody(window UiWindow) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getUiWindowMode(l)(options, window)
			return uipkt.NewUiOpen(mode).Encode(l, ctx)(options)
		}
	}
}

func getUiWindowMode(l logrus.FieldLogger) func(options map[string]interface{}, key UiWindow) byte {
	return func(options map[string]interface{}, key UiWindow) byte {
		return atlas_packet.ResolveCode(l, options, "operations", string(key))
	}
}
