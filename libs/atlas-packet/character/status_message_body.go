package character

import (
	"context"
	"time"

	atlas_packet "github.com/Chronicle20/atlas-packet"
	"github.com/Chronicle20/atlas-packet/character/clientbound"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

func CharacterStatusMessageDropPickUpItemUnavailableBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", "DROP_PICK_UP", func(mode byte) packet.Encoder {
		return clientbound.NewStatusMessageDropPickUpItemUnavailable(mode)
	})
}

func CharacterStatusMessageDropPickUpInventoryFullBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", "DROP_PICK_UP", func(mode byte) packet.Encoder {
		return clientbound.NewStatusMessageDropPickUpInventoryFull(mode)
	})
}

func CharacterStatusMessageOperationDropPickUpStackableItemBody(itemId uint32, amount uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", "DROP_PICK_UP", func(mode byte) packet.Encoder {
		return clientbound.NewStatusMessageDropPickUpStackableItem(mode, itemId, amount)
	})
}

func CharacterStatusMessageOperationDropPickUpUnStackableItemBody(itemId uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", "DROP_PICK_UP", func(mode byte) packet.Encoder {
		return clientbound.NewStatusMessageDropPickUpUnStackableItem(mode, itemId)
	})
}

func CharacterStatusMessageOperationDropPickUpMesoBody(partial bool, amount uint32, internetCafeBonus uint16) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", "DROP_PICK_UP", func(mode byte) packet.Encoder {
		return clientbound.NewStatusMessageDropPickUpMeso(mode, partial, amount, internetCafeBonus)
	})
}

func CharacterStatusMessageOperationForfeitQuestRecordBody(questId uint16) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", "QUEST_RECORD", func(mode byte) packet.Encoder {
		return clientbound.NewStatusMessageForfeitQuestRecord(mode, questId)
	})
}

func CharacterStatusMessageOperationUpdateQuestRecordBody(questId uint16, info string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", "QUEST_RECORD", func(mode byte) packet.Encoder {
		return clientbound.NewStatusMessageUpdateQuestRecord(mode, questId, info)
	})
}

func CharacterStatusMessageOperationCompleteQuestRecordBody(questId uint16, completedAt time.Time) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", "QUEST_RECORD", func(mode byte) packet.Encoder {
		return clientbound.NewStatusMessageCompleteQuestRecord(mode, questId, completedAt)
	})
}

func CharacterStatusMessageOperationCashItemExpireBody(itemId uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", "CASH_ITEM_EXPIRE", func(mode byte) packet.Encoder {
		return clientbound.NewStatusMessageCashItemExpire(mode, itemId)
	})
}

func CharacterStatusMessageOperationIncreaseExperienceBody(white bool, amount int32, inChat bool, monsterBookBonus int32,
	mobEventBonusPercentage byte, partyBonusPercentage byte, weddingBonusEXP int32, playTimeHour byte,
	questBonusRate byte, questBonusRemainCount byte, partyBonusEventRate byte, partyBonusExp int32,
	itemBonusEXP int32, premiumIPExp int32, rainbowWeekEventEXP int32, partyEXPRingEXP int32, cakePieEventBonus int32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", "INCREASE_EXPERIENCE", func(mode byte) packet.Encoder {
		return clientbound.NewStatusMessageIncreaseExperience(mode,
			white, amount, inChat, monsterBookBonus,
			mobEventBonusPercentage, partyBonusPercentage, weddingBonusEXP, playTimeHour,
			questBonusRate, questBonusRemainCount, partyBonusEventRate, partyBonusExp,
			itemBonusEXP, premiumIPExp, rainbowWeekEventEXP, partyEXPRingEXP, cakePieEventBonus,
		)
	})
}

func CharacterStatusMessageOperationIncreaseSkillPointBody(jobId uint16, amount byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", "INCREASE_SKILL_POINT", func(mode byte) packet.Encoder {
		return clientbound.NewStatusMessageIncreaseSkillPoint(mode, jobId, amount)
	})
}

func CharacterStatusMessageOperationIncreaseFameBody(amount int32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", "INCREASE_FAME", func(mode byte) packet.Encoder {
		return clientbound.NewStatusMessageIncreaseFame(mode, amount)
	})
}

func CharacterStatusMessageOperationIncreaseMesoBody(amount int32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", "INCREASE_MESO", func(mode byte) packet.Encoder {
		return clientbound.NewStatusMessageIncreaseMeso(mode, amount)
	})
}

func CharacterStatusMessageOperationIncreaseGuildPointBody(amount int32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", "INCREASE_GUILD_POINT", func(mode byte) packet.Encoder {
		return clientbound.NewStatusMessageIncreaseGuildPoint(mode, amount)
	})
}

func CharacterStatusMessageOperationGiveBuffBody(itemId uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", "GIVE_BUFF", func(mode byte) packet.Encoder {
		return clientbound.NewStatusMessageGiveBuff(mode, itemId)
	})
}

func CharacterStatusMessageOperationGeneralItemExpireBody(itemIds []uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", "GENERAL_ITEM_EXPIRE", func(mode byte) packet.Encoder {
		return clientbound.NewStatusMessageGeneralItemExpire(mode, itemIds)
	})
}

func CharacterStatusMessageOperationSystemMessageBody(message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", "SYSTEM_MESSAGE", func(mode byte) packet.Encoder {
		return clientbound.NewStatusMessageSystemMessage(mode, message)
	})
}

func CharacterStatusMessageOperationQuestRecordExBody(questId uint16, info string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", "QUEST_RECORD_EX", func(mode byte) packet.Encoder {
		return clientbound.NewStatusMessageQuestRecordEx(mode, questId, info)
	})
}

func CharacterStatusMessageOperationItemProtectExpireBody(itemIds []uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", "ITEM_PROTECT_EXPIRE", func(mode byte) packet.Encoder {
		return clientbound.NewStatusMessageItemProtectExpire(mode, itemIds)
	})
}

func CharacterStatusMessageOperationItemExpireReplaceBody(messages []string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", "ITEM_EXPIRE_REPLACE", func(mode byte) packet.Encoder {
		return clientbound.NewStatusMessageItemExpireReplace(mode, messages)
	})
}

func CharacterStatusMessageOperationSkillExpireBody(skillIds []uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", "SKILL_EXPIRE", func(mode byte) packet.Encoder {
		return clientbound.NewStatusMessageSkillExpire(mode, skillIds)
	})
}
