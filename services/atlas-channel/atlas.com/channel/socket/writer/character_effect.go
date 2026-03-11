package writer

import (
	"atlas-channel/socket/model"
	"context"

	atlas_packet "github.com/Chronicle20/atlas-packet"
	charpkt "github.com/Chronicle20/atlas-packet/character"
	"github.com/Chronicle20/atlas-constants/skill"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

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
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectLevelUp)
			return charpkt.NewEffectSimple(mode).Encode(l, ctx)(options)
		}
	}
}

func CharacterLevelUpEffectForeignBody(characterId uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectLevelUp)
			return charpkt.NewEffectSimpleForeign(characterId, mode).Encode(l, ctx)(options)
		}
	}
}

func CharacterSkillUseEffectBody(skillId uint32, characterLevel byte, skillLevel byte, darkForceEffect bool, createOrDeleteDragon bool, left bool) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectSkillUse)
			isBerserk := skill.Id(skillId) == skill.DarkKnightBerserkId
			isDragonFury := skill.Id(skillId) == skill.EvanStage8DragonFuryId
			isMonsterMagnet := skill.Is(skill.Id(skillId), skill.HeroMonsterMagnetId, skill.PaladinMonsterMagnetId, skill.DarkKnightMonsterMagnetId)
			return charpkt.NewEffectSkillUse(mode, skillId, characterLevel, skillLevel, isBerserk, darkForceEffect, isDragonFury, createOrDeleteDragon, isMonsterMagnet, left).Encode(l, ctx)(options)
		}
	}
}

func CharacterSkillUseEffectForeignBody(characterId uint32, skillId uint32, characterLevel byte, skillLevel byte, darkForceEffect bool, createOrDeleteDragon bool, left bool) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectSkillUse)
			isBerserk := skill.Id(skillId) == skill.DarkKnightBerserkId
			isDragonFury := skill.Id(skillId) == skill.EvanStage8DragonFuryId
			isMonsterMagnet := skill.Is(skill.Id(skillId), skill.HeroMonsterMagnetId, skill.PaladinMonsterMagnetId, skill.DarkKnightMonsterMagnetId)
			return charpkt.NewEffectSkillUseForeign(characterId, mode, skillId, characterLevel, skillLevel, isBerserk, darkForceEffect, isDragonFury, createOrDeleteDragon, isMonsterMagnet, left).Encode(l, ctx)(options)
		}
	}
}

func CharacterSkillAffectedEffectBody(skillId uint32, skillLevel byte) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectSkillAffected)
			return charpkt.NewEffectSkillAffected(mode, skillId, skillLevel).Encode(l, ctx)(options)
		}
	}
}

func CharacterSkillAffectedEffectForeignBody(characterId uint32, skillId uint32, skillLevel byte) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectSkillAffected)
			return charpkt.NewEffectSkillAffectedForeign(characterId, mode, skillId, skillLevel).Encode(l, ctx)(options)
		}
	}
}

func CharacterQuestEffectBody(message string, rewards []model.QuestReward, nEffect uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectQuest)
			pktRewards := make([]charpkt.QuestReward, len(rewards))
			for i, r := range rewards {
				pktRewards[i] = charpkt.QuestReward{ItemId: r.ItemId(), Amount: r.Amount()}
			}
			return charpkt.NewEffectQuest(mode, message, nEffect, pktRewards).Encode(l, ctx)(options)
		}
	}
}

func CharacterQuestEffectForeignBody(characterId uint32, message string, rewards []model.QuestReward, nEffect uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectQuest)
			pktRewards := make([]charpkt.QuestReward, len(rewards))
			for i, r := range rewards {
				pktRewards[i] = charpkt.QuestReward{ItemId: r.ItemId(), Amount: r.Amount()}
			}
			return charpkt.NewEffectQuestForeign(characterId, mode, message, nEffect, pktRewards).Encode(l, ctx)(options)
		}
	}
}

func CharacterPetEffectBody(petIndex byte, effectType byte) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectPet)
			return charpkt.NewEffectPet(mode, effectType, petIndex).Encode(l, ctx)(options)
		}
	}
}

func CharacterPetEffectForeignBody(characterId uint32, petIndex byte, effectType byte) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectPet)
			return charpkt.NewEffectPetForeign(characterId, mode, effectType, petIndex).Encode(l, ctx)(options)
		}
	}
}

func CharacterSkillSpecialEffectBody(skillId uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectSkillSpecial)
			return charpkt.NewEffectWithId(mode, skillId).Encode(l, ctx)(options)
		}
	}
}

func CharacterSkillSpecialEffectForeignBody(characterId uint32, skillId uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectSkillSpecial)
			return charpkt.NewEffectWithIdForeign(characterId, mode, skillId).Encode(l, ctx)(options)
		}
	}
}

func CharacterProtectOnDieItemUseEffectBody(safetyCharm bool, usesRemaining byte, days byte, itemId uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectProtectOnDieItemUse)
			return charpkt.NewEffectProtectOnDie(mode, safetyCharm, usesRemaining, days, itemId).Encode(l, ctx)(options)
		}
	}
}

func CharacterProtectOnDieItemUseEffectForeignBody(characterId uint32, safetyCharm bool, usesRemaining byte, days byte, itemId uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectProtectOnDieItemUse)
			return charpkt.NewEffectProtectOnDieForeign(characterId, mode, safetyCharm, usesRemaining, days, itemId).Encode(l, ctx)(options)
		}
	}
}

func CharacterPlayPortalSoundEffectEffectBody() packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectPlayPortalSoundEffect)
			return charpkt.NewEffectSimple(mode).Encode(l, ctx)(options)
		}
	}
}

func CharacterPlayPortalSoundEffectEffectForeignBody(characterId uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectPlayPortalSoundEffect)
			return charpkt.NewEffectSimpleForeign(characterId, mode).Encode(l, ctx)(options)
		}
	}
}

func CharacterJobChangedEffectBody() packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectJobChanged)
			return charpkt.NewEffectSimple(mode).Encode(l, ctx)(options)
		}
	}
}

func CharacterJobChangedEffectForeignBody(characterId uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectJobChanged)
			return charpkt.NewEffectSimpleForeign(characterId, mode).Encode(l, ctx)(options)
		}
	}
}

func CharacterQuestCompleteEffectBody() packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectQuestComplete)
			return charpkt.NewEffectSimple(mode).Encode(l, ctx)(options)
		}
	}
}

func CharacterQuestCompleteEffectForeignBody(characterId uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectQuestComplete)
			return charpkt.NewEffectSimpleForeign(characterId, mode).Encode(l, ctx)(options)
		}
	}
}

// TODO this will crash for some reason
func CharacterIncDecHPEffectBody(delta int8) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectIncDecHPEffect)
			return charpkt.NewEffectIncDecHP(mode, delta).Encode(l, ctx)(options)
		}
	}
}

// TODO this will crash for some reason
func CharacterIncDecHPEffectForeignBody(characterId uint32, delta int8) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectIncDecHPEffect)
			return charpkt.NewEffectIncDecHPForeign(characterId, mode, delta).Encode(l, ctx)(options)
		}
	}
}

func CharacterBuffItemEffectBody(itemId uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectBuffItemEffect)
			return charpkt.NewEffectWithId(mode, itemId).Encode(l, ctx)(options)
		}
	}
}

func CharacterBuffItemEffectForeignBody(characterId uint32, itemId uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectBuffItemEffect)
			return charpkt.NewEffectWithIdForeign(characterId, mode, itemId).Encode(l, ctx)(options)
		}
	}
}

func CharacterShowIntroEffectBody(message string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			// TODO characters may be facing the wrong way during these interactions. Not possible to change facing direction.
			mode := getCharacterEffect(l)(options, CharacterEffectShowIntroEffect)
			return charpkt.NewEffectWithMessage(mode, message).Encode(l, ctx)(options)
		}
	}
}

func CharacterShowIntroEffectForeignBody(characterId uint32, message string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectShowIntroEffect)
			return charpkt.NewEffectWithMessageForeign(characterId, mode, message).Encode(l, ctx)(options)
		}
	}
}

func CharacterMonsterBookCardGetEffectBody() packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectMonsterBookCardGet)
			return charpkt.NewEffectSimple(mode).Encode(l, ctx)(options)
		}
	}
}

func CharacterMonsterBookCardGetEffectForeignBody(characterId uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectMonsterBookCardGet)
			return charpkt.NewEffectSimpleForeign(characterId, mode).Encode(l, ctx)(options)
		}
	}
}

func CharacterLotteryUseEffectBody(itemId uint32, success bool, message string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectLotteryUse)
			return charpkt.NewEffectLotteryUse(mode, itemId, success, message).Encode(l, ctx)(options)
		}
	}
}

func CharacterLotteryUseEffectForeignBody(characterId uint32, itemId uint32, success bool, message string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectLotteryUse)
			return charpkt.NewEffectLotteryUseForeign(characterId, mode, itemId, success, message).Encode(l, ctx)(options)
		}
	}
}

func CharacterItemLevelUpEffectBody() packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectItemLevelUp)
			return charpkt.NewEffectSimple(mode).Encode(l, ctx)(options)
		}
	}
}

func CharacterItemLevelUpEffectForeignBody(characterId uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectItemLevelUp)
			return charpkt.NewEffectSimpleForeign(characterId, mode).Encode(l, ctx)(options)
		}
	}
}

func CharacterItemMakerEffectBody(state uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectItemMaker)
			return charpkt.NewEffectItemMaker(mode, state).Encode(l, ctx)(options)
		}
	}
}

func CharacterItemMakerEffectForeignBody(characterId uint32, state uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectItemMaker)
			return charpkt.NewEffectItemMakerForeign(characterId, mode, state).Encode(l, ctx)(options)
		}
	}
}

func CharacterShowInfoEffectBody(path string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectShowInfo)
			return charpkt.NewEffectShowInfo(mode, path).Encode(l, ctx)(options)
		}
	}
}

func CharacterShowInfoEffectForeignBody(characterId uint32, path string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectShowInfo)
			return charpkt.NewEffectShowInfoForeign(characterId, mode, path).Encode(l, ctx)(options)
		}
	}
}

func CharacterReservedEffectBody(message string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectReservedEffect)
			return charpkt.NewEffectWithMessage(mode, message).Encode(l, ctx)(options)
		}
	}
}

func CharacterReservedEffectForeignBody(characterId uint32, message string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectReservedEffect)
			return charpkt.NewEffectWithMessageForeign(characterId, mode, message).Encode(l, ctx)(options)
		}
	}
}

func CharacterConsumeEffectBody(itemId uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectConsumeEffect)
			return charpkt.NewEffectWithId(mode, itemId).Encode(l, ctx)(options)
		}
	}
}

func CharacterConsumeEffectForeignBody(characterId uint32, itemId uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectConsumeEffect)
			return charpkt.NewEffectWithIdForeign(characterId, mode, itemId).Encode(l, ctx)(options)
		}
	}
}

func CharacterUpgradeTombItemUseEffectBody(usesRemaining byte) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectUpgradeTombItemUse)
			return charpkt.NewEffectUpgradeTomb(mode, usesRemaining).Encode(l, ctx)(options)
		}
	}
}

func CharacterUpgradeTombItemUseEffectForeignBody(characterId uint32, usesRemaining byte) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectUpgradeTombItemUse)
			return charpkt.NewEffectUpgradeTombForeign(characterId, mode, usesRemaining).Encode(l, ctx)(options)
		}
	}
}

func CharacterBattlefieldItemUseEffectBody(message string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectBattlefieldItemUse)
			return charpkt.NewEffectWithMessage(mode, message).Encode(l, ctx)(options)
		}
	}
}

func CharacterBattlefieldItemUseEffectForeignBody(characterId uint32, message string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectBattlefieldItemUse)
			return charpkt.NewEffectWithMessageForeign(characterId, mode, message).Encode(l, ctx)(options)
		}
	}
}

func CharacterIncubatorUseEffectBody(itemId uint32, message string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectIncubatorUse)
			return charpkt.NewEffectIncubatorUse(mode, itemId, message).Encode(l, ctx)(options)
		}
	}
}

func CharacterIncubatorUseEffectForeignBody(characterId uint32, itemId uint32, message string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectIncubatorUse)
			return charpkt.NewEffectIncubatorUseForeign(characterId, mode, itemId, message).Encode(l, ctx)(options)
		}
	}
}

func CharacterPlaySoundWithMuteBackgroundMusicEffectBody(songName string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectPlaySoundWithMuteBackgroundMusic)
			return charpkt.NewEffectWithMessage(mode, songName).Encode(l, ctx)(options)
		}
	}
}

func CharacterPlaySoundWithMuteBackgroundMusicEffectForeignBody(characterId uint32, songName string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectPlaySoundWithMuteBackgroundMusic)
			return charpkt.NewEffectWithMessageForeign(characterId, mode, songName).Encode(l, ctx)(options)
		}
	}
}

func CharacterSoulStoneUseEffectBody() packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectSoulStoneUse)
			return charpkt.NewEffectSimple(mode).Encode(l, ctx)(options)
		}
	}
}

func CharacterSoulStoneUseEffectForeignBody(characterId uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCharacterEffect(l)(options, CharacterEffectSoulStoneUse)
			return charpkt.NewEffectSimpleForeign(characterId, mode).Encode(l, ctx)(options)
		}
	}
}

func getCharacterEffect(l logrus.FieldLogger) func(options map[string]interface{}, key CharacterEffectMode) byte {
	return func(options map[string]interface{}, key CharacterEffectMode) byte {
		return atlas_packet.ResolveCode(l, options, "operations", string(key))
	}
}
