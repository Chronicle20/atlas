package writer

import (
	"atlas-channel/socket/model"
	"strconv"

	"github.com/Chronicle20/atlas-constants/skill"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterEffect = "CharacterEffect"
const CharacterEffectForeign = "CharacterEffectForeign"

const (
	CharacterEffectLevelUp                          = "LEVEL_UP"
	CharacterEffectSkillUse                         = "SKILL_USE"
	CharacterEffectSkillAffected                    = "SKILL_AFFECTED"
	CharacterEffectQuest                            = "QUEST"
	CharacterEffectPet                              = "PET"
	CharacterEffectSkillSpecial                     = "SKILL_SPECIAL"
	CharacterEffectProtectOnDieItemUse              = "PROTECT_ON_DIE_ITEM_USE"
	CharacterEffectPlayPortalSoundEffect            = "PLAY_PORTAL_SOUND_EFFECT"
	CharacterEffectJobChanged                       = "JOB_CHANGED"
	CharacterEffectQuestComplete                    = "QUEST_COMPLETE"
	CharacterEffectIncDecHPEffect                   = "INC_DEC_HP_EFFECT"
	CharacterEffectBuffItemEffect                   = "BUFF_ITEM_EFFECT"
	CharacterEffectShowIntroEffect                  = "SHOW_INTRO"
	CharacterEffectMonsterBookCardGet               = "MONSTER_BOOK_CARD_GET"
	CharacterEffectLotteryUse                       = "LOTTERY_USE"
	CharacterEffectItemLevelUp                      = "ITEM_LEVEL_UP"
	CharacterEffectItemMaker                        = "ITEM_MAKER"
	CharacterEffectShowInfo                         = "SHOW_INFO"
	CharacterEffectReservedEffect                   = "RESERVED_EFFECT"
	CharacterEffectBuff                             = "BUFF"
	CharacterEffectConsumeEffect                    = "CONSUME_EFFECT"
	CharacterEffectUpgradeTombItemUse               = "UPGRADE_TOMB_ITEM_USE"
	CharacterEffectBattlefieldItemUse               = "BATTLEFIELD_ITEM_USE"
	CharacterEffectAvatarOriented                   = "AVATAR_ORIENTED"
	CharacterEffectIncubatorUse                     = "INCUBATOR_USE"
	CharacterEffectPlaySoundWithMuteBackgroundMusic = "PLAY_SOUND_WITH_MUTE_BACKGROUND_MUSIC"
	CharacterEffectSoulStoneUse                     = "SOUL_STONE_USE"
	CharacterEffectDeliveryQuestItemUse             = "DELIVERY_QUEST_ITEM_USE"
	CharacterEffectRepeatEffectRemove               = "REPEAT_REPEAT_EFFECT"
	CharacterEffectEvolutionRing                    = "EVOLUTION_RING"

	PetEffectLevelUp   = byte(0)
	PetEffectDisappear = byte(1)
)

func CharacterLevelUpEffectBody(l logrus.FieldLogger) func() BodyProducer {
	return func() BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCharacterEffect(l)(options, CharacterEffectLevelUp))
			return w.Bytes()
		}
	}
}

func CharacterLevelUpEffectForeignBody(l logrus.FieldLogger) func(characterId uint32) BodyProducer {
	return func(characterId uint32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(characterId)
			return CharacterLevelUpEffectBody(l)()(w, options)
		}
	}
}

func CharacterSkillUseEffectBody(l logrus.FieldLogger) func(skillId uint32, characterLevel byte, skillLevel byte, darkForceEffect bool, createOrDeleteDragon bool, left bool) BodyProducer {
	return func(skillId uint32, characterLevel byte, skillLevel byte, darkForceEffect bool, createOrDeleteDragon bool, left bool) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCharacterEffect(l)(options, CharacterEffectSkillUse))
			w.WriteInt(skillId)
			w.WriteByte(characterLevel)
			w.WriteByte(skillLevel)
			if skill.Id(skillId) == skill.DarkKnightBerserkId {
				w.WriteBool(darkForceEffect)
			}
			if skill.Id(skillId) == skill.EvanStage8DragonFuryId {
				w.WriteBool(createOrDeleteDragon)
			}
			if skill.Is(skill.Id(skillId), skill.HeroMonsterMagnetId, skill.PaladinMonsterMagnetId, skill.DarkKnightMonsterMagnetId) {
				w.WriteBool(left)
			}
			return w.Bytes()
		}
	}
}

func CharacterSkillUseEffectForeignBody(l logrus.FieldLogger) func(characterId uint32, skillId uint32, characterLevel byte, skillLevel byte, darkForceEffect bool, createOrDeleteDragon bool, left bool) BodyProducer {
	return func(characterId uint32, skillId uint32, characterLevel byte, skillLevel byte, darkForceEffect bool, createOrDeleteDragon bool, left bool) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(characterId)
			return CharacterSkillUseEffectBody(l)(skillId, characterLevel, skillLevel, darkForceEffect, createOrDeleteDragon, left)(w, options)
		}
	}
}

func CharacterSkillAffectedEffectBody(l logrus.FieldLogger) func(skillId uint32, skillLevel byte) BodyProducer {
	return func(skillId uint32, skillLevel byte) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCharacterEffect(l)(options, CharacterEffectSkillAffected))
			w.WriteInt(skillId)
			w.WriteByte(skillLevel)
			return w.Bytes()
		}
	}
}

func CharacterSkillAffectedEffectForeignBody(l logrus.FieldLogger) func(characterId uint32, skillId uint32, skillLevel byte) BodyProducer {
	return func(characterId uint32, skillId uint32, skillLevel byte) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(characterId)
			return CharacterSkillAffectedEffectBody(l)(skillId, skillLevel)(w, options)
		}
	}
}

func CharacterQuestEffectBody(l logrus.FieldLogger) func(message string, rewards []model.QuestReward, nEffect uint32) BodyProducer {
	return func(message string, rewards []model.QuestReward, nEffect uint32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCharacterEffect(l)(options, CharacterEffectQuest))
			w.WriteByte(byte(len(rewards)))
			if len(rewards) == 0 {
				w.WriteAsciiString(message)
				w.WriteInt(nEffect)
			} else {
				for _, r := range rewards {
					w.WriteInt(r.ItemId())
					w.WriteInt32(r.Amount())
				}
			}
			return w.Bytes()
		}
	}
}

func CharacterQuestEffectForeignBody(l logrus.FieldLogger) func(characterId uint32, message string, rewards []model.QuestReward, nEffect uint32) BodyProducer {
	return func(characterId uint32, message string, rewards []model.QuestReward, nEffect uint32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(characterId)
			return CharacterQuestEffectBody(l)(message, rewards, nEffect)(w, options)
		}
	}
}

func CharacterPetEffectBody(l logrus.FieldLogger) func(petIndex byte, effectType byte) BodyProducer {
	return func(petIndex byte, effectType byte) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCharacterEffect(l)(options, CharacterEffectPet))
			w.WriteByte(effectType)
			w.WriteByte(petIndex)
			return w.Bytes()
		}
	}
}

func CharacterPetEffectForeignBody(l logrus.FieldLogger) func(characterId uint32, petIndex byte, effectType byte) BodyProducer {
	return func(characterId uint32, petIndex byte, effectType byte) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(characterId)
			return CharacterPetEffectBody(l)(petIndex, effectType)(w, options)
		}
	}
}

func CharacterSkillSpecialEffectBody(l logrus.FieldLogger) func(skillId uint32) BodyProducer {
	return func(skillId uint32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCharacterEffect(l)(options, CharacterEffectSkillSpecial))
			w.WriteInt(skillId)
			return w.Bytes()
		}
	}
}

func CharacterSkillSpecialEffectForeignBody(l logrus.FieldLogger) func(characterId uint32, skillId uint32) BodyProducer {
	return func(characterId uint32, skillId uint32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(characterId)
			return CharacterSkillSpecialEffectBody(l)(skillId)(w, options)
		}
	}
}

func CharacterProtectOnDieItemUseEffectBody(l logrus.FieldLogger) func(safetyCharm bool, usesRemaining byte, days byte, itemId uint32) BodyProducer {
	return func(safetyCharm bool, usesRemaining byte, days byte, itemId uint32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCharacterEffect(l)(options, CharacterEffectProtectOnDieItemUse))
			w.WriteBool(safetyCharm)
			w.WriteByte(usesRemaining)
			w.WriteByte(days)
			if !safetyCharm {
				w.WriteInt(itemId)
			}
			return w.Bytes()
		}
	}
}

func CharacterProtectOnDieItemUseEffectForeignBody(l logrus.FieldLogger) func(characterId uint32, safetyCharm bool, usesRemaining byte, days byte, itemId uint32) BodyProducer {
	return func(characterId uint32, safetyCharm bool, usesRemaining byte, days byte, itemId uint32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(characterId)
			return CharacterProtectOnDieItemUseEffectBody(l)(safetyCharm, usesRemaining, days, itemId)(w, options)
		}
	}
}

func CharacterPlayPortalSoundEffectEffectBody(l logrus.FieldLogger) func() BodyProducer {
	return func() BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCharacterEffect(l)(options, CharacterEffectPlayPortalSoundEffect))
			return w.Bytes()
		}
	}
}

func CharacterPlayPortalSoundEffectEffectForeignBody(l logrus.FieldLogger) func(characterId uint32) BodyProducer {
	return func(characterId uint32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(characterId)
			return CharacterPlayPortalSoundEffectEffectBody(l)()(w, options)
		}
	}
}

func CharacterJobChangedEffectBody(l logrus.FieldLogger) func() BodyProducer {
	return func() BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCharacterEffect(l)(options, CharacterEffectJobChanged))
			return w.Bytes()
		}
	}
}

func CharacterJobChangedEffectForeignBody(l logrus.FieldLogger) func(characterId uint32) BodyProducer {
	return func(characterId uint32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(characterId)
			return CharacterJobChangedEffectBody(l)()(w, options)
		}
	}
}

func CharacterQuestCompleteEffectBody(l logrus.FieldLogger) func() BodyProducer {
	return func() BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCharacterEffect(l)(options, CharacterEffectQuestComplete))
			return w.Bytes()
		}
	}
}

func CharacterQuestCompleteEffectForeignBody(l logrus.FieldLogger) func(characterId uint32) BodyProducer {
	return func(characterId uint32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(characterId)
			return CharacterQuestCompleteEffectBody(l)()(w, options)
		}
	}
}

// TODO this will crash for some reason
func CharacterIncDecHPEffectBody(l logrus.FieldLogger) func(delta int8) BodyProducer {
	return func(delta int8) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCharacterEffect(l)(options, CharacterEffectIncDecHPEffect))
			w.WriteInt8(delta)
			return w.Bytes()
		}
	}
}

// TODO this will crash for some reason
func CharacterIncDecHPEffectForeignBody(l logrus.FieldLogger) func(characterId uint32, delta int8) BodyProducer {
	return func(characterId uint32, delta int8) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(characterId)
			return CharacterIncDecHPEffectBody(l)(delta)(w, options)
		}
	}
}

func CharacterBuffItemEffectBody(l logrus.FieldLogger) func(itemId uint32) BodyProducer {
	return func(itemId uint32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCharacterEffect(l)(options, CharacterEffectBuffItemEffect))
			w.WriteInt(itemId)
			return w.Bytes()
		}
	}
}

func CharacterBuffItemEffectForeignBody(l logrus.FieldLogger) func(characterId uint32, itemId uint32) BodyProducer {
	return func(characterId uint32, itemId uint32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(characterId)
			return CharacterBuffItemEffectBody(l)(itemId)(w, options)
		}
	}
}

func CharacterShowIntroEffectBody(l logrus.FieldLogger) func(message string) BodyProducer {
	return func(message string) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCharacterEffect(l)(options, CharacterEffectShowIntroEffect))
			w.WriteAsciiString(message)
			return w.Bytes()
		}
	}
}

func CharacterShowIntroEffectForeignBody(l logrus.FieldLogger) func(characterId uint32, message string) BodyProducer {
	return func(characterId uint32, message string) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(characterId)
			return CharacterShowIntroEffectBody(l)(message)(w, options)
		}
	}
}

func CharacterMonsterBookCardGetEffectBody(l logrus.FieldLogger) func() BodyProducer {
	return func() BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCharacterEffect(l)(options, CharacterEffectMonsterBookCardGet))
			return w.Bytes()
		}
	}
}

func CharacterMonsterBookCardGetEffectForeignBody(l logrus.FieldLogger) func(characterId uint32) BodyProducer {
	return func(characterId uint32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(characterId)
			return CharacterMonsterBookCardGetEffectBody(l)()(w, options)
		}
	}
}

func CharacterLotteryUseEffectBody(l logrus.FieldLogger) func(itemId uint32, success bool, message string) BodyProducer {
	return func(itemId uint32, success bool, message string) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCharacterEffect(l)(options, CharacterEffectLotteryUse))
			w.WriteInt(itemId)
			w.WriteBool(success)
			if success {
				w.WriteAsciiString(message)
			}
			return w.Bytes()
		}
	}
}

func CharacterLotteryUseEffectForeignBody(l logrus.FieldLogger) func(characterId uint32, itemId uint32, success bool, message string) BodyProducer {
	return func(characterId uint32, itemId uint32, success bool, message string) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(characterId)
			return CharacterLotteryUseEffectBody(l)(itemId, success, message)(w, options)
		}
	}
}

func CharacterItemLevelUpEffectBody(l logrus.FieldLogger) func() BodyProducer {
	return func() BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCharacterEffect(l)(options, CharacterEffectItemLevelUp))
			return w.Bytes()
		}
	}
}

func CharacterItemLevelUpEffectForeignBody(l logrus.FieldLogger) func(characterId uint32) BodyProducer {
	return func(characterId uint32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(characterId)
			return CharacterItemLevelUpEffectBody(l)()(w, options)
		}
	}
}

func CharacterItemMakerEffectBody(l logrus.FieldLogger) func(state uint32) BodyProducer {
	return func(state uint32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCharacterEffect(l)(options, CharacterEffectItemMaker))
			w.WriteInt(state)
			return w.Bytes()
		}
	}
}

func CharacterItemMakerEffectForeignBody(l logrus.FieldLogger) func(characterId uint32, state uint32) BodyProducer {
	return func(characterId uint32, state uint32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(characterId)
			return CharacterItemMakerEffectBody(l)(state)(w, options)
		}
	}
}

func CharacterShowInfoEffectBody(l logrus.FieldLogger) func(path string) BodyProducer {
	return func(path string) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCharacterEffect(l)(options, CharacterEffectShowInfo))
			w.WriteAsciiString(path)
			w.WriteInt(1) // not used
			return w.Bytes()
		}
	}
}

func CharacterShowInfoEffectForeignBody(l logrus.FieldLogger) func(characterId uint32, path string) BodyProducer {
	return func(characterId uint32, path string) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(characterId)
			return CharacterShowInfoEffectBody(l)(path)(w, options)
		}
	}
}

func CharacterReservedEffectBody(l logrus.FieldLogger) func(message string) BodyProducer {
	return func(message string) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCharacterEffect(l)(options, CharacterEffectReservedEffect))
			w.WriteAsciiString(message)
			return w.Bytes()
		}
	}
}

func CharacterReservedEffectForeignBody(l logrus.FieldLogger) func(characterId uint32, message string) BodyProducer {
	return func(characterId uint32, message string) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(characterId)
			return CharacterReservedEffectBody(l)(message)(w, options)
		}
	}
}

func CharacterConsumeEffectBody(l logrus.FieldLogger) func(itemId uint32) BodyProducer {
	return func(itemId uint32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCharacterEffect(l)(options, CharacterEffectConsumeEffect))
			w.WriteInt(itemId)
			return w.Bytes()
		}
	}
}

func CharacterConsumeEffectForeignBody(l logrus.FieldLogger) func(characterId uint32, itemId uint32) BodyProducer {
	return func(characterId uint32, itemId uint32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(characterId)
			return CharacterConsumeEffectBody(l)(itemId)(w, options)
		}
	}
}

func CharacterUpgradeTombItemUseEffectBody(l logrus.FieldLogger) func(usesRemaining byte) BodyProducer {
	return func(usesRemaining byte) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCharacterEffect(l)(options, CharacterEffectUpgradeTombItemUse))
			w.WriteByte(usesRemaining)
			return w.Bytes()
		}
	}
}

func CharacterUpgradeTombItemUseEffectForeignBody(l logrus.FieldLogger) func(characterId uint32, usesRemaining byte) BodyProducer {
	return func(characterId uint32, usesRemaining byte) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(characterId)
			return CharacterUpgradeTombItemUseEffectBody(l)(usesRemaining)(w, options)
		}
	}
}

func CharacterBattlefieldItemUseEffectBody(l logrus.FieldLogger) func(message string) BodyProducer {
	return func(message string) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCharacterEffect(l)(options, CharacterEffectBattlefieldItemUse))
			w.WriteAsciiString(message)
			return w.Bytes()
		}
	}
}

func CharacterBattlefieldItemUseEffectForeignBody(l logrus.FieldLogger) func(characterId uint32, message string) BodyProducer {
	return func(characterId uint32, message string) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(characterId)
			return CharacterBattlefieldItemUseEffectBody(l)(message)(w, options)
		}
	}
}

func CharacterAvatarOrientedEffectBody(l logrus.FieldLogger) func(message string) BodyProducer {
	return func(message string) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCharacterEffect(l)(options, CharacterEffectAvatarOriented))
			w.WriteAsciiString(message)
			w.WriteInt(0) // unused
			return w.Bytes()
		}
	}
}

func CharacterAvatarOrientedEffectForeignBody(l logrus.FieldLogger) func(characterId uint32, message string) BodyProducer {
	return func(characterId uint32, message string) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(characterId)
			return CharacterAvatarOrientedEffectBody(l)(message)(w, options)
		}
	}
}

func CharacterIncubatorUseEffectBody(l logrus.FieldLogger) func(itemId uint32, message string) BodyProducer {
	return func(itemId uint32, message string) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCharacterEffect(l)(options, CharacterEffectIncubatorUse))
			w.WriteInt(itemId)
			w.WriteAsciiString(message)
			return w.Bytes()
		}
	}
}

func CharacterIncubatorUseEffectForeignBody(l logrus.FieldLogger) func(characterId uint32, itemId uint32, message string) BodyProducer {
	return func(characterId uint32, itemId uint32, message string) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(characterId)
			return CharacterIncubatorUseEffectBody(l)(itemId, message)(w, options)
		}
	}
}

func CharacterPlaySoundWithMuteBackgroundMusicEffectBody(l logrus.FieldLogger) func(songName string) BodyProducer {
	return func(songName string) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCharacterEffect(l)(options, CharacterEffectPlaySoundWithMuteBackgroundMusic))
			w.WriteAsciiString(songName)
			return w.Bytes()
		}
	}
}

func CharacterPlaySoundWithMuteBackgroundMusicEffectForeignBody(l logrus.FieldLogger) func(characterId uint32, songName string) BodyProducer {
	return func(characterId uint32, songName string) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(characterId)
			return CharacterPlaySoundWithMuteBackgroundMusicEffectBody(l)(songName)(w, options)
		}
	}
}

func CharacterSoulStoneUseEffectBody(l logrus.FieldLogger) func() BodyProducer {
	return func() BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCharacterEffect(l)(options, CharacterEffectSoulStoneUse))
			return w.Bytes()
		}
	}
}

func CharacterSoulStoneUseEffectForeignBody(l logrus.FieldLogger) func(characterId uint32) BodyProducer {
	return func(characterId uint32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(characterId)
			return CharacterSoulStoneUseEffectBody(l)()(w, options)
		}
	}
}

func getCharacterEffect(l logrus.FieldLogger) func(options map[string]interface{}, key string) byte {
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
