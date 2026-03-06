package writer

import (
	"atlas-channel/socket/model"
	"context"

	"github.com/Chronicle20/atlas-constants/skill"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterEffect = "CharacterEffect"
const CharacterEffectForeign = "CharacterEffectForeign"

type CharacterEffectMode string

// CUser::OnEffect

const (
	CharacterEffectLevelUp                          CharacterEffectMode = "LEVEL_UP"                              // 0
	CharacterEffectSkillUse                         CharacterEffectMode = "SKILL_USE"                             // 1
	CharacterEffectSkillAffected                    CharacterEffectMode = "SKILL_AFFECTED"                        // 2
	CharacterEffectQuest                            CharacterEffectMode = "QUEST"                                 // 3
	CharacterEffectPet                              CharacterEffectMode = "PET"                                   // 4
	CharacterEffectSkillSpecial                     CharacterEffectMode = "SKILL_SPECIAL"                         // 5
	CharacterEffectProtectOnDieItemUse              CharacterEffectMode = "PROTECT_ON_DIE_ITEM_USE"               // 6
	CharacterEffectPlayPortalSoundEffect            CharacterEffectMode = "PLAY_PORTAL_SOUND_EFFECT"              // 7
	CharacterEffectJobChanged                       CharacterEffectMode = "JOB_CHANGED"                           // 8
	CharacterEffectQuestComplete                    CharacterEffectMode = "QUEST_COMPLETE"                        // 9
	CharacterEffectIncDecHPEffect                   CharacterEffectMode = "INC_DEC_HP_EFFECT"                     // 10
	CharacterEffectBuffItemEffect                   CharacterEffectMode = "BUFF_ITEM_EFFECT"                      // 11
	CharacterEffectShowIntroEffect                  CharacterEffectMode = "SHOW_INTRO"                            // 12
	CharacterEffectMonsterBookCardGet               CharacterEffectMode = "MONSTER_BOOK_CARD_GET"                 // 13
	CharacterEffectLotteryUse                       CharacterEffectMode = "LOTTERY_USE"                           // 14
	CharacterEffectItemLevelUp                      CharacterEffectMode = "ITEM_LEVEL_UP"                         // 15
	CharacterEffectItemMaker                        CharacterEffectMode = "ITEM_MAKER"                            // 16
	CharacterEffectItemExperienceConsumed           CharacterEffectMode = "ITEM_EXPERIENCE_CONSUMED"              // 17
	CharacterEffectReservedEffect                   CharacterEffectMode = "RESERVED_EFFECT"                       // 18
	CharacterEffectBuff                             CharacterEffectMode = "BUFF"                                  // 19 not in v83
	CharacterEffectConsumeEffect                    CharacterEffectMode = "CONSUME_EFFECT"                        // 20
	CharacterEffectUpgradeTombItemUse               CharacterEffectMode = "UPGRADE_TOMB_ITEM_USE"                 // 21
	CharacterEffectBattlefieldItemUse               CharacterEffectMode = "BATTLEFIELD_ITEM_USE"                  // 22
	CharacterEffectShowInfo                         CharacterEffectMode = "SHOW_INFO"                             // 23
	CharacterEffectIncubatorUse                     CharacterEffectMode = "INCUBATOR_USE"                         // 24
	CharacterEffectPlaySoundWithMuteBackgroundMusic CharacterEffectMode = "PLAY_SOUND_WITH_MUTE_BACKGROUND_MUSIC" // 25
	CharacterEffectSoulStoneUse                     CharacterEffectMode = "SOUL_STONE_USE"                        // 26

	PetEffectLevelUp   = byte(0)
	PetEffectDisappear = byte(1)
)

func CharacterLevelUpEffectBody() packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getCharacterEffect(l)(options, CharacterEffectLevelUp))
			return w.Bytes()
		}
	}
}

func CharacterLevelUpEffectForeignBody(characterId uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		w.WriteInt(characterId)
		return CharacterLevelUpEffectBody()(l, ctx)
	}
}

func CharacterSkillUseEffectBody(skillId uint32, characterLevel byte, skillLevel byte, darkForceEffect bool, createOrDeleteDragon bool, left bool) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
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

func CharacterSkillUseEffectForeignBody(characterId uint32, skillId uint32, characterLevel byte, skillLevel byte, darkForceEffect bool, createOrDeleteDragon bool, left bool) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		w.WriteInt(characterId)
		return CharacterSkillUseEffectBody(skillId, characterLevel, skillLevel, darkForceEffect, createOrDeleteDragon, left)(l, ctx)
	}
}

func CharacterSkillAffectedEffectBody(skillId uint32, skillLevel byte) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getCharacterEffect(l)(options, CharacterEffectSkillAffected))
			w.WriteInt(skillId)
			w.WriteByte(skillLevel)
			return w.Bytes()
		}
	}
}

func CharacterSkillAffectedEffectForeignBody(characterId uint32, skillId uint32, skillLevel byte) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		w.WriteInt(characterId)
		return CharacterSkillAffectedEffectBody(skillId, skillLevel)(l, ctx)
	}
}

func CharacterQuestEffectBody(message string, rewards []model.QuestReward, nEffect uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
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

func CharacterQuestEffectForeignBody(characterId uint32, message string, rewards []model.QuestReward, nEffect uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		w.WriteInt(characterId)
		return CharacterQuestEffectBody(message, rewards, nEffect)(l, ctx)
	}
}

func CharacterPetEffectBody(petIndex byte, effectType byte) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getCharacterEffect(l)(options, CharacterEffectPet))
			w.WriteByte(effectType)
			w.WriteByte(petIndex)
			return w.Bytes()
		}
	}
}

func CharacterPetEffectForeignBody(characterId uint32, petIndex byte, effectType byte) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		w.WriteInt(characterId)
		return CharacterPetEffectBody(petIndex, effectType)(l, ctx)
	}
}

func CharacterSkillSpecialEffectBody(skillId uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getCharacterEffect(l)(options, CharacterEffectSkillSpecial))
			w.WriteInt(skillId)
			return w.Bytes()
		}
	}
}

func CharacterSkillSpecialEffectForeignBody(characterId uint32, skillId uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		w.WriteInt(characterId)
		return CharacterSkillSpecialEffectBody(skillId)(l, ctx)
	}
}

func CharacterProtectOnDieItemUseEffectBody(safetyCharm bool, usesRemaining byte, days byte, itemId uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
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

func CharacterProtectOnDieItemUseEffectForeignBody(characterId uint32, safetyCharm bool, usesRemaining byte, days byte, itemId uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		w.WriteInt(characterId)
		return CharacterProtectOnDieItemUseEffectBody(safetyCharm, usesRemaining, days, itemId)(l, ctx)
	}
}

func CharacterPlayPortalSoundEffectEffectBody() packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getCharacterEffect(l)(options, CharacterEffectPlayPortalSoundEffect))
			return w.Bytes()
		}
	}
}

func CharacterPlayPortalSoundEffectEffectForeignBody(characterId uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		w.WriteInt(characterId)
		return CharacterPlayPortalSoundEffectEffectBody()(l, ctx)
	}
}

func CharacterJobChangedEffectBody() packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getCharacterEffect(l)(options, CharacterEffectJobChanged))
			return w.Bytes()
		}
	}
}

func CharacterJobChangedEffectForeignBody(characterId uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		w.WriteInt(characterId)
		return CharacterJobChangedEffectBody()(l, ctx)
	}
}

func CharacterQuestCompleteEffectBody() packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getCharacterEffect(l)(options, CharacterEffectQuestComplete))
			return w.Bytes()
		}
	}
}

func CharacterQuestCompleteEffectForeignBody(characterId uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		w.WriteInt(characterId)
		return CharacterQuestCompleteEffectBody()(l, ctx)
	}
}

// TODO this will crash for some reason
func CharacterIncDecHPEffectBody(delta int8) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getCharacterEffect(l)(options, CharacterEffectIncDecHPEffect))
			w.WriteInt8(delta)
			return w.Bytes()
		}
	}
}

// TODO this will crash for some reason
func CharacterIncDecHPEffectForeignBody(characterId uint32, delta int8) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		w.WriteInt(characterId)
		return CharacterIncDecHPEffectBody(delta)(l, ctx)
	}
}

func CharacterBuffItemEffectBody(itemId uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getCharacterEffect(l)(options, CharacterEffectBuffItemEffect))
			w.WriteInt(itemId)
			return w.Bytes()
		}
	}
}

func CharacterBuffItemEffectForeignBody(characterId uint32, itemId uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		w.WriteInt(characterId)
		return CharacterBuffItemEffectBody(itemId)(l, ctx)
	}
}

func CharacterShowIntroEffectBody(message string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			// TODO characters may be facing the wrong way during these interactions. Not possible to change facing direction.
			w.WriteByte(getCharacterEffect(l)(options, CharacterEffectShowIntroEffect))
			w.WriteAsciiString(message)
			return w.Bytes()
		}
	}
}

func CharacterShowIntroEffectForeignBody(characterId uint32, message string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		w.WriteInt(characterId)
		return CharacterShowIntroEffectBody(message)(l, ctx)
	}
}

func CharacterMonsterBookCardGetEffectBody() packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getCharacterEffect(l)(options, CharacterEffectMonsterBookCardGet))
			return w.Bytes()
		}
	}
}

func CharacterMonsterBookCardGetEffectForeignBody(characterId uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		w.WriteInt(characterId)
		return CharacterMonsterBookCardGetEffectBody()(l, ctx)
	}
}

func CharacterLotteryUseEffectBody(itemId uint32, success bool, message string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
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

func CharacterLotteryUseEffectForeignBody(characterId uint32, itemId uint32, success bool, message string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		w.WriteInt(characterId)
		return CharacterLotteryUseEffectBody(itemId, success, message)(l, ctx)
	}
}

func CharacterItemLevelUpEffectBody() packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getCharacterEffect(l)(options, CharacterEffectItemLevelUp))
			return w.Bytes()
		}
	}
}

func CharacterItemLevelUpEffectForeignBody(characterId uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		w.WriteInt(characterId)
		return CharacterItemLevelUpEffectBody()(l, ctx)
	}
}

func CharacterItemMakerEffectBody(state uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getCharacterEffect(l)(options, CharacterEffectItemMaker))
			w.WriteInt(state)
			return w.Bytes()
		}
	}
}

func CharacterItemMakerEffectForeignBody(characterId uint32, state uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		w.WriteInt(characterId)
		return CharacterItemMakerEffectBody(state)(l, ctx)
	}
}

func CharacterShowInfoEffectBody(path string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getCharacterEffect(l)(options, CharacterEffectShowInfo))
			w.WriteAsciiString(path)
			w.WriteInt(1) // not used
			return w.Bytes()
		}
	}
}

func CharacterShowInfoEffectForeignBody(characterId uint32, path string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		w.WriteInt(characterId)
		return CharacterShowInfoEffectBody(path)(l, ctx)
	}
}

func CharacterReservedEffectBody(message string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getCharacterEffect(l)(options, CharacterEffectReservedEffect))
			w.WriteAsciiString(message)
			return w.Bytes()
		}
	}
}

func CharacterReservedEffectForeignBody(characterId uint32, message string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		w.WriteInt(characterId)
		return CharacterReservedEffectBody(message)(l, ctx)
	}
}

func CharacterConsumeEffectBody(itemId uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getCharacterEffect(l)(options, CharacterEffectConsumeEffect))
			w.WriteInt(itemId)
			return w.Bytes()
		}
	}
}

func CharacterConsumeEffectForeignBody(characterId uint32, itemId uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		w.WriteInt(characterId)
		return CharacterConsumeEffectBody(itemId)(l, ctx)
	}
}

func CharacterUpgradeTombItemUseEffectBody(usesRemaining byte) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getCharacterEffect(l)(options, CharacterEffectUpgradeTombItemUse))
			w.WriteByte(usesRemaining)
			return w.Bytes()
		}
	}
}

func CharacterUpgradeTombItemUseEffectForeignBody(characterId uint32, usesRemaining byte) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		w.WriteInt(characterId)
		return CharacterUpgradeTombItemUseEffectBody(usesRemaining)(l, ctx)
	}
}

func CharacterBattlefieldItemUseEffectBody(message string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getCharacterEffect(l)(options, CharacterEffectBattlefieldItemUse))
			w.WriteAsciiString(message)
			return w.Bytes()
		}
	}
}

func CharacterBattlefieldItemUseEffectForeignBody(characterId uint32, message string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		w.WriteInt(characterId)
		return CharacterBattlefieldItemUseEffectBody(message)(l, ctx)
	}
}

func CharacterIncubatorUseEffectBody(itemId uint32, message string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getCharacterEffect(l)(options, CharacterEffectIncubatorUse))
			w.WriteInt(itemId)
			w.WriteAsciiString(message)
			return w.Bytes()
		}
	}
}

func CharacterIncubatorUseEffectForeignBody(characterId uint32, itemId uint32, message string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		w.WriteInt(characterId)
		return CharacterIncubatorUseEffectBody(itemId, message)(l, ctx)
	}
}

func CharacterPlaySoundWithMuteBackgroundMusicEffectBody(songName string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getCharacterEffect(l)(options, CharacterEffectPlaySoundWithMuteBackgroundMusic))
			w.WriteAsciiString(songName)
			return w.Bytes()
		}
	}
}

func CharacterPlaySoundWithMuteBackgroundMusicEffectForeignBody(characterId uint32, songName string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		w.WriteInt(characterId)
		return CharacterPlaySoundWithMuteBackgroundMusicEffectBody(songName)(l, ctx)
	}
}

func CharacterSoulStoneUseEffectBody() packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getCharacterEffect(l)(options, CharacterEffectSoulStoneUse))
			return w.Bytes()
		}
	}
}

func CharacterSoulStoneUseEffectForeignBody(characterId uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		w.WriteInt(characterId)
		return CharacterSoulStoneUseEffectBody()(l, ctx)
	}
}

func getCharacterEffect(l logrus.FieldLogger) func(options map[string]interface{}, key CharacterEffectMode) byte {
	return func(options map[string]interface{}, key CharacterEffectMode) byte {
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
