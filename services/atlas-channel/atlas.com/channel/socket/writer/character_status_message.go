package writer

import (
	"atlas-channel/socket/model"
	"context"
	"strconv"
	"time"

	charpkt "github.com/Chronicle20/atlas-packet/character"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

const (
	CharacterStatusMessage                            = "CharacterStatusMessage"
	CharacterStatusMessageOperationDropPickUp         = "DROP_PICK_UP"
	CharacterStatusMessageOperationQuestRecord        = "QUEST_RECORD"
	CharacterStatusMessageOperationCashItemExpire     = "CASH_ITEM_EXPIRE"
	CharacterStatusMessageOperationIncreaseExperience = "INCREASE_EXPERIENCE"
	CharacterStatusMessageOperationIncreaseSkillPoint = "INCREASE_SKILL_POINT"
	CharacterStatusMessageOperationIncreaseFame       = "INCREASE_FAME"
	CharacterStatusMessageOperationIncreaseMeso       = "INCREASE_MESO"
	CharacterStatusMessageOperationIncreaseGuildPoint = "INCREASE_GUILD_POINT"
	CharacterStatusMessageOperationGiveBuff           = "GIVE_BUFF"
	CharacterStatusMessageOperationGeneralItemExpire  = "GENERAL_ITEM_EXPIRE"
	CharacterStatusMessageOperationSystemMessage      = "SYSTEM_MESSAGE"
	CharacterStatusMessageOperationQuestRecordEx      = "QUEST_RECORD_EX"
	CharacterStatusMessageOperationItemProtectExpire  = "ITEM_PROTECT_EXPIRE"
	CharacterStatusMessageOperationItemExpireReplace  = "ITEM_EXPIRE_REPLACE"
	CharacterStatusMessageOperationSkillExpire        = "SKILL_EXPIRE"
)

func CharacterStatusMessageDropPickUpItemUnavailableBody() packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterStatusMessageOperation(l)(options, CharacterStatusMessageOperationDropPickUp)
			return charpkt.NewStatusMessageDropPickUpItemUnavailable(mode).Encode(l, ctx)(options)
		}
	}
}

func CharacterStatusMessageDropPickUpInventoryFullBody() packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterStatusMessageOperation(l)(options, CharacterStatusMessageOperationDropPickUp)
			return charpkt.NewStatusMessageDropPickUpInventoryFull(mode).Encode(l, ctx)(options)
		}
	}
}

func CharacterStatusMessageOperationDropPickUpStackableItemBody(itemId uint32, amount uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterStatusMessageOperation(l)(options, CharacterStatusMessageOperationDropPickUp)
			return charpkt.NewStatusMessageDropPickUpStackableItem(mode, itemId, amount).Encode(l, ctx)(options)
		}
	}
}

func CharacterStatusMessageOperationDropPickUpUnStackableItemBody(itemId uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterStatusMessageOperation(l)(options, CharacterStatusMessageOperationDropPickUp)
			return charpkt.NewStatusMessageDropPickUpUnStackableItem(mode, itemId).Encode(l, ctx)(options)
		}
	}
}

func CharacterStatusMessageOperationDropPickUpMesoBody(partial bool, amount uint32, internetCafeBonus uint16) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterStatusMessageOperation(l)(options, CharacterStatusMessageOperationDropPickUp)
			return charpkt.NewStatusMessageDropPickUpMeso(mode, partial, amount, internetCafeBonus).Encode(l, ctx)(options)
		}
	}
}

func CharacterStatusMessageOperationForfeitQuestRecordBody(questId uint16) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterStatusMessageOperation(l)(options, CharacterStatusMessageOperationQuestRecord)
			return charpkt.NewStatusMessageForfeitQuestRecord(mode, questId).Encode(l, ctx)(options)
		}
	}
}

func CharacterStatusMessageOperationUpdateQuestRecordBody(questId uint16, info string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterStatusMessageOperation(l)(options, CharacterStatusMessageOperationQuestRecord)
			return charpkt.NewStatusMessageUpdateQuestRecord(mode, questId, info).Encode(l, ctx)(options)
		}
	}
}

func CharacterStatusMessageOperationCompleteQuestRecordBody(questId uint16, completedAt time.Time) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterStatusMessageOperation(l)(options, CharacterStatusMessageOperationQuestRecord)
			return charpkt.NewStatusMessageCompleteQuestRecord(mode, questId, completedAt).Encode(l, ctx)(options)
		}
	}
}

func CharacterStatusMessageOperationCashItemExpireBody(itemId uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterStatusMessageOperation(l)(options, CharacterStatusMessageOperationCashItemExpire)
			return charpkt.NewStatusMessageCashItemExpire(mode, itemId).Encode(l, ctx)(options)
		}
	}
}

func CharacterStatusMessageOperationIncreaseExperienceBody(c model.IncreaseExperienceConfig) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterStatusMessageOperation(l)(options, CharacterStatusMessageOperationIncreaseExperience)
			return charpkt.NewStatusMessageIncreaseExperience(mode,
				c.White, c.Amount, c.InChat, c.MonsterBookBonus,
				c.MobEventBonusPercentage, c.PartyBonusPercentage, c.WeddingBonusEXP, c.PlayTimeHour,
				c.QuestBonusRate, c.QuestBonusRemainCount, c.PartyBonusEventRate, c.PartyBonusExp,
				c.ItemBonusEXP, c.PremiumIPExp, c.RainbowWeekEventEXP, c.PartyEXPRingEXP, c.CakePieEventBonus,
			).Encode(l, ctx)(options)
		}
	}
}

func CharacterStatusMessageOperationIncreaseSkillPointBody(jobId uint16, amount byte) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterStatusMessageOperation(l)(options, CharacterStatusMessageOperationIncreaseSkillPoint)
			return charpkt.NewStatusMessageIncreaseSkillPoint(mode, jobId, amount).Encode(l, ctx)(options)
		}
	}
}

func CharacterStatusMessageOperationIncreaseFameBody(amount int32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterStatusMessageOperation(l)(options, CharacterStatusMessageOperationIncreaseFame)
			return charpkt.NewStatusMessageIncreaseFame(mode, amount).Encode(l, ctx)(options)
		}
	}
}

func CharacterStatusMessageOperationIncreaseMesoBody(amount int32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterStatusMessageOperation(l)(options, CharacterStatusMessageOperationIncreaseMeso)
			return charpkt.NewStatusMessageIncreaseMeso(mode, amount).Encode(l, ctx)(options)
		}
	}
}

func CharacterStatusMessageOperationIncreaseGuildPointBody(amount int32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterStatusMessageOperation(l)(options, CharacterStatusMessageOperationIncreaseGuildPoint)
			return charpkt.NewStatusMessageIncreaseGuildPoint(mode, amount).Encode(l, ctx)(options)
		}
	}
}

func CharacterStatusMessageOperationGiveBuffBody(itemId uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterStatusMessageOperation(l)(options, CharacterStatusMessageOperationGiveBuff)
			return charpkt.NewStatusMessageGiveBuff(mode, itemId).Encode(l, ctx)(options)
		}
	}
}

func CharacterStatusMessageOperationGeneralItemExpireBody(itemIds []uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterStatusMessageOperation(l)(options, CharacterStatusMessageOperationGeneralItemExpire)
			return charpkt.NewStatusMessageGeneralItemExpire(mode, itemIds).Encode(l, ctx)(options)
		}
	}
}

func CharacterStatusMessageOperationSystemMessageBody(message string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterStatusMessageOperation(l)(options, CharacterStatusMessageOperationSystemMessage)
			return charpkt.NewStatusMessageSystemMessage(mode, message).Encode(l, ctx)(options)
		}
	}
}

func CharacterStatusMessageOperationQuestRecordExBody(questId uint16, info string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterStatusMessageOperation(l)(options, CharacterStatusMessageOperationQuestRecordEx)
			return charpkt.NewStatusMessageQuestRecordEx(mode, questId, info).Encode(l, ctx)(options)
		}
	}
}

func CharacterStatusMessageOperationItemProtectExpireBody(itemIds []uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterStatusMessageOperation(l)(options, CharacterStatusMessageOperationItemProtectExpire)
			return charpkt.NewStatusMessageItemProtectExpire(mode, itemIds).Encode(l, ctx)(options)
		}
	}
}

func CharacterStatusMessageOperationItemExpireReplaceBody(messages []string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterStatusMessageOperation(l)(options, CharacterStatusMessageOperationItemExpireReplace)
			return charpkt.NewStatusMessageItemExpireReplace(mode, messages).Encode(l, ctx)(options)
		}
	}
}

func CharacterStatusMessageOperationSkillExpireBody(skillIds []uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterStatusMessageOperation(l)(options, CharacterStatusMessageOperationSkillExpire)
			return charpkt.NewStatusMessageSkillExpire(mode, skillIds).Encode(l, ctx)(options)
		}
	}
}

func getCharacterStatusMessageOperation(l logrus.FieldLogger) func(options map[string]interface{}, key string) byte {
	return func(options map[string]interface{}, key string) byte {
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
		if code, ok = codes[key]; !ok {
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
