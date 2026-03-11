package character

import (
	"time"

	atlas_packet "github.com/Chronicle20/atlas-packet"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/sirupsen/logrus"
	"context"
)

func CharacterStatusMessageDropPickUpItemUnavailableBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", "DROP_PICK_UP", func(mode byte) packet.Encoder {
		return NewStatusMessageDropPickUpItemUnavailable(mode)
	})
}

func CharacterStatusMessageDropPickUpInventoryFullBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", "DROP_PICK_UP", func(mode byte) packet.Encoder {
		return NewStatusMessageDropPickUpInventoryFull(mode)
	})
}

func CharacterStatusMessageOperationDropPickUpStackableItemBody(itemId uint32, amount uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", "DROP_PICK_UP", func(mode byte) packet.Encoder {
		return NewStatusMessageDropPickUpStackableItem(mode, itemId, amount)
	})
}

func CharacterStatusMessageOperationDropPickUpUnStackableItemBody(itemId uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", "DROP_PICK_UP", func(mode byte) packet.Encoder {
		return NewStatusMessageDropPickUpUnStackableItem(mode, itemId)
	})
}

func CharacterStatusMessageOperationDropPickUpMesoBody(partial bool, amount uint32, internetCafeBonus uint16) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", "DROP_PICK_UP", func(mode byte) packet.Encoder {
		return NewStatusMessageDropPickUpMeso(mode, partial, amount, internetCafeBonus)
	})
}

func CharacterStatusMessageOperationForfeitQuestRecordBody(questId uint16) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", "QUEST_RECORD", func(mode byte) packet.Encoder {
		return NewStatusMessageForfeitQuestRecord(mode, questId)
	})
}

func CharacterStatusMessageOperationUpdateQuestRecordBody(questId uint16, info string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", "QUEST_RECORD", func(mode byte) packet.Encoder {
		return NewStatusMessageUpdateQuestRecord(mode, questId, info)
	})
}

func CharacterStatusMessageOperationCompleteQuestRecordBody(questId uint16, completedAt time.Time) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", "QUEST_RECORD", func(mode byte) packet.Encoder {
		return NewStatusMessageCompleteQuestRecord(mode, questId, completedAt)
	})
}

func CharacterStatusMessageOperationCashItemExpireBody(itemId uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", "CASH_ITEM_EXPIRE", func(mode byte) packet.Encoder {
		return NewStatusMessageCashItemExpire(mode, itemId)
	})
}

func CharacterStatusMessageOperationIncreaseExperienceBody(white bool, amount int32, inChat bool, monsterBookBonus int32,
	mobEventBonusPercentage byte, partyBonusPercentage byte, weddingBonusEXP int32, playTimeHour byte,
	questBonusRate byte, questBonusRemainCount byte, partyBonusEventRate byte, partyBonusExp int32,
	itemBonusEXP int32, premiumIPExp int32, rainbowWeekEventEXP int32, partyEXPRingEXP int32, cakePieEventBonus int32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", "INCREASE_EXPERIENCE", func(mode byte) packet.Encoder {
		return NewStatusMessageIncreaseExperience(mode,
			white, amount, inChat, monsterBookBonus,
			mobEventBonusPercentage, partyBonusPercentage, weddingBonusEXP, playTimeHour,
			questBonusRate, questBonusRemainCount, partyBonusEventRate, partyBonusExp,
			itemBonusEXP, premiumIPExp, rainbowWeekEventEXP, partyEXPRingEXP, cakePieEventBonus,
		)
	})
}

func CharacterStatusMessageOperationIncreaseSkillPointBody(jobId uint16, amount byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", "INCREASE_SKILL_POINT", func(mode byte) packet.Encoder {
		return NewStatusMessageIncreaseSkillPoint(mode, jobId, amount)
	})
}

func CharacterStatusMessageOperationIncreaseFameBody(amount int32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", "INCREASE_FAME", func(mode byte) packet.Encoder {
		return NewStatusMessageIncreaseFame(mode, amount)
	})
}

func CharacterStatusMessageOperationIncreaseMesoBody(amount int32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", "INCREASE_MESO", func(mode byte) packet.Encoder {
		return NewStatusMessageIncreaseMeso(mode, amount)
	})
}

func CharacterStatusMessageOperationIncreaseGuildPointBody(amount int32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", "INCREASE_GUILD_POINT", func(mode byte) packet.Encoder {
		return NewStatusMessageIncreaseGuildPoint(mode, amount)
	})
}

func CharacterStatusMessageOperationGiveBuffBody(itemId uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", "GIVE_BUFF", func(mode byte) packet.Encoder {
		return NewStatusMessageGiveBuff(mode, itemId)
	})
}

func CharacterStatusMessageOperationGeneralItemExpireBody(itemIds []uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", "GENERAL_ITEM_EXPIRE", func(mode byte) packet.Encoder {
		return NewStatusMessageGeneralItemExpire(mode, itemIds)
	})
}

func CharacterStatusMessageOperationSystemMessageBody(message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", "SYSTEM_MESSAGE", func(mode byte) packet.Encoder {
		return NewStatusMessageSystemMessage(mode, message)
	})
}

func CharacterStatusMessageOperationQuestRecordExBody(questId uint16, info string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", "QUEST_RECORD_EX", func(mode byte) packet.Encoder {
		return NewStatusMessageQuestRecordEx(mode, questId, info)
	})
}

func CharacterStatusMessageOperationItemProtectExpireBody(itemIds []uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", "ITEM_PROTECT_EXPIRE", func(mode byte) packet.Encoder {
		return NewStatusMessageItemProtectExpire(mode, itemIds)
	})
}

func CharacterStatusMessageOperationItemExpireReplaceBody(messages []string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", "ITEM_EXPIRE_REPLACE", func(mode byte) packet.Encoder {
		return NewStatusMessageItemExpireReplace(mode, messages)
	})
}

func CharacterStatusMessageOperationSkillExpireBody(skillIds []uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", "SKILL_EXPIRE", func(mode byte) packet.Encoder {
		return NewStatusMessageSkillExpire(mode, skillIds)
	})
}
