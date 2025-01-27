package writer

import (
	"atlas-channel/socket/model"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"strconv"
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

func CharacterStatusMessageDropPickUpItemUnavailableBody(l logrus.FieldLogger) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		// Written to right side of screen.
		w.WriteByte(getCharacterStatusMessageOperation(l)(options, CharacterStatusMessageOperationDropPickUp))
		w.WriteInt8(-2)
		return w.Bytes()
	}
}

func CharacterStatusMessageDropPickUpInventoryFullBody(l logrus.FieldLogger) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		// Written to right side of screen.
		w.WriteByte(getCharacterStatusMessageOperation(l)(options, CharacterStatusMessageOperationDropPickUp))
		w.WriteInt8(-3)
		return w.Bytes()
	}
}

func CharacterStatusMessageOperationDropPickUpStackableItemBody(l logrus.FieldLogger) func(itemId uint32, amount uint32) BodyProducer {
	return func(itemId uint32, amount uint32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			// Written to right side of screen.
			w.WriteByte(getCharacterStatusMessageOperation(l)(options, CharacterStatusMessageOperationDropPickUp))
			w.WriteInt8(0)
			w.WriteInt(itemId)
			w.WriteInt(amount)
			return w.Bytes()
		}
	}
}

func CharacterStatusMessageOperationDropPickUpUnStackableItemBody(l logrus.FieldLogger) func(itemId uint32) BodyProducer {
	return func(itemId uint32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			// Written to right side of screen.
			w.WriteByte(getCharacterStatusMessageOperation(l)(options, CharacterStatusMessageOperationDropPickUp))
			w.WriteInt8(2)
			w.WriteInt(itemId)
			return w.Bytes()
		}
	}
}

func CharacterStatusMessageOperationDropPickUpMesoBody(l logrus.FieldLogger) func(partial bool, amount uint32, internetCafeBonus uint16) BodyProducer {
	return func(partial bool, amount uint32, internetCafeBonus uint16) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			// Written to right side of screen.
			w.WriteByte(getCharacterStatusMessageOperation(l)(options, CharacterStatusMessageOperationDropPickUp))
			w.WriteInt8(1)
			w.WriteBool(partial)
			w.WriteInt(amount)
			w.WriteShort(internetCafeBonus)
			return w.Bytes()
		}
	}
}

func CharacterStatusMessageOperationForfeitQuestRecordBody(l logrus.FieldLogger) func(questId uint16) BodyProducer {
	return func(questId uint16) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCharacterStatusMessageOperation(l)(options, CharacterStatusMessageOperationQuestRecord))
			w.WriteShort(questId)
			w.WriteByte(0)
			return w.Bytes()
		}
	}
}

func CharacterStatusMessageOperationUpdateQuestRecordBody(l logrus.FieldLogger) func(questId uint16, info string) BodyProducer {
	return func(questId uint16, info string) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCharacterStatusMessageOperation(l)(options, CharacterStatusMessageOperationQuestRecord))
			w.WriteShort(questId)
			w.WriteByte(1)
			w.WriteAsciiString(info)
			w.WriteByte(0)  // TODO ??
			w.WriteInt32(0) // TODO ??
			return w.Bytes()
		}
	}
}

func CharacterStatusMessageOperationCompleteQuestRecordBody(l logrus.FieldLogger) func(questId uint16) BodyProducer {
	return func(questId uint16) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCharacterStatusMessageOperation(l)(options, CharacterStatusMessageOperationQuestRecord))
			w.WriteShort(questId)
			w.WriteByte(2)
			w.WriteLong(0) // TODO completion time
			return w.Bytes()
		}
	}
}

func CharacterStatusMessageOperationCashItemExpireBody(l logrus.FieldLogger) func(itemId uint32) BodyProducer {
	return func(itemId uint32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			// Written in chat as a pink message.
			w.WriteByte(getCharacterStatusMessageOperation(l)(options, CharacterStatusMessageOperationCashItemExpire))
			w.WriteInt(itemId)
			return w.Bytes()
		}
	}
}

func CharacterStatusMessageOperationIncreaseExperienceBody(l logrus.FieldLogger, t tenant.Model) func(c model.IncreaseExperienceConfig) BodyProducer {
	return func(c model.IncreaseExperienceConfig) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCharacterStatusMessageOperation(l)(options, CharacterStatusMessageOperationIncreaseExperience))
			w.WriteBool(c.White)
			w.WriteInt32(c.Amount)
			w.WriteBool(c.InChat)
			w.WriteInt32(c.MonsterBookBonus)
			w.WriteByte(c.MobEventBonusPercentage)
			w.WriteByte(c.PartyBonusPercentage)
			w.WriteInt32(c.WeddingBonusEXP)
			if c.MobEventBonusPercentage > 0 {
				w.WriteByte(c.PlayTimeHour)
			}
			if c.InChat {
				w.WriteByte(c.QuestBonusRate)
				if c.QuestBonusRate > 0 {
					w.WriteByte(c.QuestBonusRemainCount)
				}
			}
			w.WriteByte(c.PartyBonusEventRate)
			w.WriteInt32(c.PartyBonusExp)
			w.WriteInt32(c.ItemBonusEXP)
			w.WriteInt32(c.PremiumIPExp)
			w.WriteInt32(c.RainbowWeekEventEXP)

			if t.Region() == "GMS" && t.MajorVersion() >= 95 {
				w.WriteInt32(c.PartyEXPRingEXP)
				w.WriteInt32(c.CakePieEventBonus)
			}
			return w.Bytes()
		}
	}
}

func CharacterStatusMessageOperationIncreaseSkillPointBody(l logrus.FieldLogger) func(jobId uint16, amount byte) BodyProducer {
	return func(jobId uint16, amount byte) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCharacterStatusMessageOperation(l)(options, CharacterStatusMessageOperationIncreaseSkillPoint))
			w.WriteShort(jobId)
			w.WriteByte(amount)
			return w.Bytes()
		}
	}
}

func CharacterStatusMessageOperationIncreaseFameBody(l logrus.FieldLogger) func(amount int32) BodyProducer {
	return func(amount int32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			// Written in chat as a gray text.
			w.WriteByte(getCharacterStatusMessageOperation(l)(options, CharacterStatusMessageOperationIncreaseFame))
			w.WriteInt32(amount)
			return w.Bytes()
		}
	}
}

func CharacterStatusMessageOperationIncreaseMesoBody(l logrus.FieldLogger) func(amount int32) BodyProducer {
	return func(amount int32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			// Written in chat as a gray text.
			w.WriteByte(getCharacterStatusMessageOperation(l)(options, CharacterStatusMessageOperationIncreaseMeso))
			w.WriteInt32(amount)
			return w.Bytes()
		}
	}
}

func CharacterStatusMessageOperationIncreaseGuildPointBody(l logrus.FieldLogger) func(amount int32) BodyProducer {
	return func(amount int32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			// Written in chat as a gray text.
			w.WriteByte(getCharacterStatusMessageOperation(l)(options, CharacterStatusMessageOperationIncreaseGuildPoint))
			w.WriteInt32(amount)
			return w.Bytes()
		}
	}
}

func CharacterStatusMessageOperationGiveBuffBody(l logrus.FieldLogger) func(itemId uint32) BodyProducer {
	return func(itemId uint32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			// Written in chat as a gray text.
			w.WriteByte(getCharacterStatusMessageOperation(l)(options, CharacterStatusMessageOperationGiveBuff))
			w.WriteInt(itemId)
			return w.Bytes()
		}
	}
}

func CharacterStatusMessageOperationGeneralItemExpireBody(l logrus.FieldLogger) func(itemIds []uint32) BodyProducer {
	return func(itemIds []uint32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			// Written in chat as a pink text.
			w.WriteByte(getCharacterStatusMessageOperation(l)(options, CharacterStatusMessageOperationGeneralItemExpire))
			w.WriteByte(byte(len(itemIds)))
			for _, itemId := range itemIds {
				w.WriteInt(itemId)
			}
			return w.Bytes()
		}
	}
}

func CharacterStatusMessageOperationSystemMessageBody(l logrus.FieldLogger) func(message string) BodyProducer {
	return func(message string) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			// Written in chat as a pink text.
			w.WriteByte(getCharacterStatusMessageOperation(l)(options, CharacterStatusMessageOperationSystemMessage))
			w.WriteAsciiString(message)
			return w.Bytes()
		}
	}
}

func CharacterStatusMessageOperationQuestRecordExBody(l logrus.FieldLogger) func(questId uint16, info string) BodyProducer {
	return func(questId uint16, info string) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCharacterStatusMessageOperation(l)(options, CharacterStatusMessageOperationQuestRecordEx))
			w.WriteShort(questId)
			w.WriteAsciiString(info)
			return w.Bytes()
		}
	}
}

func CharacterStatusMessageOperationItemProtectExpireBody(l logrus.FieldLogger) func(itemIds []uint32) BodyProducer {
	return func(itemIds []uint32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			// Written in chat as a pink text.
			w.WriteByte(getCharacterStatusMessageOperation(l)(options, CharacterStatusMessageOperationItemProtectExpire))
			w.WriteByte(byte(len(itemIds)))
			for _, itemId := range itemIds {
				w.WriteInt(itemId)
			}
			return w.Bytes()
		}
	}
}

func CharacterStatusMessageOperationItemExpireReplaceBody(l logrus.FieldLogger) func(messages []string) BodyProducer {
	return func(messages []string) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			// Written in chat as a pink text.
			w.WriteByte(getCharacterStatusMessageOperation(l)(options, CharacterStatusMessageOperationItemExpireReplace))
			w.WriteByte(byte(len(messages)))
			for _, message := range messages {
				w.WriteAsciiString(message)
			}
			return w.Bytes()
		}
	}
}

func CharacterStatusMessageOperationSkillExpireBody(l logrus.FieldLogger) func(skillIds []uint32) BodyProducer {
	return func(skillIds []uint32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			// Written in chat as a pink text.
			w.WriteByte(getCharacterStatusMessageOperation(l)(options, CharacterStatusMessageOperationSkillExpire))
			w.WriteByte(byte(len(skillIds)))
			for _, skillId := range skillIds {
				w.WriteInt(skillId)
			}
			return w.Bytes()
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
