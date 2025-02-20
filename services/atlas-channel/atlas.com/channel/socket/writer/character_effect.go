package writer

import (
	"github.com/Chronicle20/atlas-constants/skill"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
	"strconv"
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
	CharacterEffectSquibEffect                      = "SQUIB_EFFECT"
	CharacterEffectMonsterBookCardGet               = "MONSTER_BOOK_CARD_GET"
	CharacterEffectLotteryUse                       = "LOTTERY_USE"
	CharacterEffectItemLevelUp                      = "ITEM_LEVEL_UP"
	CharacterEffectItemMaker                        = "ITEM_MAKER"
	CharacterEffectExpItemConsumed                  = "EXP_ITEM_CONSUMED"
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
