package ui

import (
	"context"

	atlas_packet "github.com/Chronicle20/atlas-packet"
	"github.com/sirupsen/logrus"
)

type UiWindow string

const (
	UiWindowItem                 UiWindow = "ITEM"
	UiWindowEquipment            UiWindow = "EQUIPMENT"
	UiWindowStatistics           UiWindow = "STATISTICS"
	UiWindowSkills               UiWindow = "SKILLS"
	UiWindowKeyboard             UiWindow = "KEYBOARD"
	UiWindowQuest                UiWindow = "QUEST"
	UiWindowMonsterBook          UiWindow = "MONSTER_BOOK"
	UiWindowCharacterInformation UiWindow = "CHARACTER_INFORMATION"
	UiWindowGuildBbs             UiWindow = "GUILD_BBS"
	UiWindowMonsterCarnival      UiWindow = "MONSTER_CARNIVAL"
	UiWindowEnergyBar            UiWindow = "ENERGY_BAR"
	UiWindowPartySearch          UiWindow = "PARTY_SEARCH"
	UiWindowItemMaker            UiWindow = "ITEM_MAKER"
	UiWindowRanking              UiWindow = "RANKING"
	UiWindowFamily               UiWindow = "FAMILY"
	UiWindowFamilyPedigree       UiWindow = "FAMILY_PEDIGREE"
	UiWindowOperatorBoard        UiWindow = "OPERATOR_BOARD"
	UiWindowOperatorBoardState   UiWindow = "OPERATOR_BOARD_STATE"
	UiWindowMedalQuest           UiWindow = "MEDAL_MEDAL_QUEST"
	UiWindowWebEvent             UiWindow = "WEB_EVENT"
	UiWindowSkillsEx             UiWindow = "SKILLS_EX"
)

func UiOpenBody(window UiWindow) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := atlas_packet.ResolveCode(l, options, "operations", string(window))
			return NewUiOpen(mode).Encode(l, ctx)(options)
		}
	}
}
