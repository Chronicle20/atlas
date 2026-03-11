package character

import (
	"context"

	atlas_packet "github.com/Chronicle20/atlas-packet"
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

func CharacterLevelUpEffectBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(CharacterEffectLevelUp), func(mode byte) packet.Encoder {
		return NewEffectSimple(mode)
	})
}

func CharacterLevelUpEffectForeignBody(characterId uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(CharacterEffectLevelUp), func(mode byte) packet.Encoder {
		return NewEffectSimpleForeign(characterId, mode)
	})
}

func CharacterSkillUseEffectBody(skillId uint32, characterLevel byte, skillLevel byte, darkForceEffect bool, createOrDeleteDragon bool, left bool) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := atlas_packet.ResolveCode(l, options, "operations", string(CharacterEffectSkillUse))
			isBerserk := skill.Id(skillId) == skill.DarkKnightBerserkId
			isDragonFury := skill.Id(skillId) == skill.EvanStage8DragonFuryId
			isMonsterMagnet := skill.Is(skill.Id(skillId), skill.HeroMonsterMagnetId, skill.PaladinMonsterMagnetId, skill.DarkKnightMonsterMagnetId)
			return NewEffectSkillUse(mode, skillId, characterLevel, skillLevel, isBerserk, darkForceEffect, isDragonFury, createOrDeleteDragon, isMonsterMagnet, left).Encode(l, ctx)(options)
		}
	}
}

func CharacterSkillUseEffectForeignBody(characterId uint32, skillId uint32, characterLevel byte, skillLevel byte, darkForceEffect bool, createOrDeleteDragon bool, left bool) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := atlas_packet.ResolveCode(l, options, "operations", string(CharacterEffectSkillUse))
			isBerserk := skill.Id(skillId) == skill.DarkKnightBerserkId
			isDragonFury := skill.Id(skillId) == skill.EvanStage8DragonFuryId
			isMonsterMagnet := skill.Is(skill.Id(skillId), skill.HeroMonsterMagnetId, skill.PaladinMonsterMagnetId, skill.DarkKnightMonsterMagnetId)
			return NewEffectSkillUseForeign(characterId, mode, skillId, characterLevel, skillLevel, isBerserk, darkForceEffect, isDragonFury, createOrDeleteDragon, isMonsterMagnet, left).Encode(l, ctx)(options)
		}
	}
}

func CharacterSkillAffectedEffectBody(skillId uint32, skillLevel byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(CharacterEffectSkillAffected), func(mode byte) packet.Encoder {
		return NewEffectSkillAffected(mode, skillId, skillLevel)
	})
}

func CharacterSkillAffectedEffectForeignBody(characterId uint32, skillId uint32, skillLevel byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(CharacterEffectSkillAffected), func(mode byte) packet.Encoder {
		return NewEffectSkillAffectedForeign(characterId, mode, skillId, skillLevel)
	})
}

func CharacterQuestEffectBody(message string, rewards []QuestReward, nEffect uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(CharacterEffectQuest), func(mode byte) packet.Encoder {
		return NewEffectQuest(mode, message, nEffect, rewards)
	})
}

func CharacterQuestEffectForeignBody(characterId uint32, message string, rewards []QuestReward, nEffect uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(CharacterEffectQuest), func(mode byte) packet.Encoder {
		return NewEffectQuestForeign(characterId, mode, message, nEffect, rewards)
	})
}

func CharacterPetEffectBody(petIndex byte, effectType byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(CharacterEffectPet), func(mode byte) packet.Encoder {
		return NewEffectPet(mode, effectType, petIndex)
	})
}

func CharacterPetEffectForeignBody(characterId uint32, petIndex byte, effectType byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(CharacterEffectPet), func(mode byte) packet.Encoder {
		return NewEffectPetForeign(characterId, mode, effectType, petIndex)
	})
}

func CharacterSkillSpecialEffectBody(skillId uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(CharacterEffectSkillSpecial), func(mode byte) packet.Encoder {
		return NewEffectWithId(mode, skillId)
	})
}

func CharacterSkillSpecialEffectForeignBody(characterId uint32, skillId uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(CharacterEffectSkillSpecial), func(mode byte) packet.Encoder {
		return NewEffectWithIdForeign(characterId, mode, skillId)
	})
}

func CharacterProtectOnDieItemUseEffectBody(safetyCharm bool, usesRemaining byte, days byte, itemId uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(CharacterEffectProtectOnDieItemUse), func(mode byte) packet.Encoder {
		return NewEffectProtectOnDie(mode, safetyCharm, usesRemaining, days, itemId)
	})
}

func CharacterProtectOnDieItemUseEffectForeignBody(characterId uint32, safetyCharm bool, usesRemaining byte, days byte, itemId uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(CharacterEffectProtectOnDieItemUse), func(mode byte) packet.Encoder {
		return NewEffectProtectOnDieForeign(characterId, mode, safetyCharm, usesRemaining, days, itemId)
	})
}

func CharacterPlayPortalSoundEffectEffectBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(CharacterEffectPlayPortalSoundEffect), func(mode byte) packet.Encoder {
		return NewEffectSimple(mode)
	})
}

func CharacterPlayPortalSoundEffectEffectForeignBody(characterId uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(CharacterEffectPlayPortalSoundEffect), func(mode byte) packet.Encoder {
		return NewEffectSimpleForeign(characterId, mode)
	})
}

func CharacterJobChangedEffectBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(CharacterEffectJobChanged), func(mode byte) packet.Encoder {
		return NewEffectSimple(mode)
	})
}

func CharacterJobChangedEffectForeignBody(characterId uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(CharacterEffectJobChanged), func(mode byte) packet.Encoder {
		return NewEffectSimpleForeign(characterId, mode)
	})
}

func CharacterQuestCompleteEffectBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(CharacterEffectQuestComplete), func(mode byte) packet.Encoder {
		return NewEffectSimple(mode)
	})
}

func CharacterQuestCompleteEffectForeignBody(characterId uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(CharacterEffectQuestComplete), func(mode byte) packet.Encoder {
		return NewEffectSimpleForeign(characterId, mode)
	})
}

// TODO this will crash for some reason
func CharacterIncDecHPEffectBody(delta int8) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(CharacterEffectIncDecHPEffect), func(mode byte) packet.Encoder {
		return NewEffectIncDecHP(mode, delta)
	})
}

// TODO this will crash for some reason
func CharacterIncDecHPEffectForeignBody(characterId uint32, delta int8) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(CharacterEffectIncDecHPEffect), func(mode byte) packet.Encoder {
		return NewEffectIncDecHPForeign(characterId, mode, delta)
	})
}

func CharacterBuffItemEffectBody(itemId uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(CharacterEffectBuffItemEffect), func(mode byte) packet.Encoder {
		return NewEffectWithId(mode, itemId)
	})
}

func CharacterBuffItemEffectForeignBody(characterId uint32, itemId uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(CharacterEffectBuffItemEffect), func(mode byte) packet.Encoder {
		return NewEffectWithIdForeign(characterId, mode, itemId)
	})
}

func CharacterShowIntroEffectBody(message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	// TODO characters may be facing the wrong way during these interactions. Not possible to change facing direction.
	return atlas_packet.WithResolvedCode("operations", string(CharacterEffectShowIntroEffect), func(mode byte) packet.Encoder {
		return NewEffectWithMessage(mode, message)
	})
}

func CharacterShowIntroEffectForeignBody(characterId uint32, message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(CharacterEffectShowIntroEffect), func(mode byte) packet.Encoder {
		return NewEffectWithMessageForeign(characterId, mode, message)
	})
}

func CharacterMonsterBookCardGetEffectBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(CharacterEffectMonsterBookCardGet), func(mode byte) packet.Encoder {
		return NewEffectSimple(mode)
	})
}

func CharacterMonsterBookCardGetEffectForeignBody(characterId uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(CharacterEffectMonsterBookCardGet), func(mode byte) packet.Encoder {
		return NewEffectSimpleForeign(characterId, mode)
	})
}

func CharacterLotteryUseEffectBody(itemId uint32, success bool, message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(CharacterEffectLotteryUse), func(mode byte) packet.Encoder {
		return NewEffectLotteryUse(mode, itemId, success, message)
	})
}

func CharacterLotteryUseEffectForeignBody(characterId uint32, itemId uint32, success bool, message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(CharacterEffectLotteryUse), func(mode byte) packet.Encoder {
		return NewEffectLotteryUseForeign(characterId, mode, itemId, success, message)
	})
}

func CharacterItemLevelUpEffectBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(CharacterEffectItemLevelUp), func(mode byte) packet.Encoder {
		return NewEffectSimple(mode)
	})
}

func CharacterItemLevelUpEffectForeignBody(characterId uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(CharacterEffectItemLevelUp), func(mode byte) packet.Encoder {
		return NewEffectSimpleForeign(characterId, mode)
	})
}

func CharacterItemMakerEffectBody(state uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(CharacterEffectItemMaker), func(mode byte) packet.Encoder {
		return NewEffectItemMaker(mode, state)
	})
}

func CharacterItemMakerEffectForeignBody(characterId uint32, state uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(CharacterEffectItemMaker), func(mode byte) packet.Encoder {
		return NewEffectItemMakerForeign(characterId, mode, state)
	})
}

func CharacterShowInfoEffectBody(path string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(CharacterEffectShowInfo), func(mode byte) packet.Encoder {
		return NewEffectShowInfo(mode, path)
	})
}

func CharacterShowInfoEffectForeignBody(characterId uint32, path string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(CharacterEffectShowInfo), func(mode byte) packet.Encoder {
		return NewEffectShowInfoForeign(characterId, mode, path)
	})
}

func CharacterReservedEffectBody(message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(CharacterEffectReservedEffect), func(mode byte) packet.Encoder {
		return NewEffectWithMessage(mode, message)
	})
}

func CharacterReservedEffectForeignBody(characterId uint32, message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(CharacterEffectReservedEffect), func(mode byte) packet.Encoder {
		return NewEffectWithMessageForeign(characterId, mode, message)
	})
}

func CharacterConsumeEffectBody(itemId uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(CharacterEffectConsumeEffect), func(mode byte) packet.Encoder {
		return NewEffectWithId(mode, itemId)
	})
}

func CharacterConsumeEffectForeignBody(characterId uint32, itemId uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(CharacterEffectConsumeEffect), func(mode byte) packet.Encoder {
		return NewEffectWithIdForeign(characterId, mode, itemId)
	})
}

func CharacterUpgradeTombItemUseEffectBody(usesRemaining byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(CharacterEffectUpgradeTombItemUse), func(mode byte) packet.Encoder {
		return NewEffectUpgradeTomb(mode, usesRemaining)
	})
}

func CharacterUpgradeTombItemUseEffectForeignBody(characterId uint32, usesRemaining byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(CharacterEffectUpgradeTombItemUse), func(mode byte) packet.Encoder {
		return NewEffectUpgradeTombForeign(characterId, mode, usesRemaining)
	})
}

func CharacterBattlefieldItemUseEffectBody(message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(CharacterEffectBattlefieldItemUse), func(mode byte) packet.Encoder {
		return NewEffectWithMessage(mode, message)
	})
}

func CharacterBattlefieldItemUseEffectForeignBody(characterId uint32, message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(CharacterEffectBattlefieldItemUse), func(mode byte) packet.Encoder {
		return NewEffectWithMessageForeign(characterId, mode, message)
	})
}

func CharacterIncubatorUseEffectBody(itemId uint32, message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(CharacterEffectIncubatorUse), func(mode byte) packet.Encoder {
		return NewEffectIncubatorUse(mode, itemId, message)
	})
}

func CharacterIncubatorUseEffectForeignBody(characterId uint32, itemId uint32, message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(CharacterEffectIncubatorUse), func(mode byte) packet.Encoder {
		return NewEffectIncubatorUseForeign(characterId, mode, itemId, message)
	})
}

func CharacterPlaySoundWithMuteBackgroundMusicEffectBody(songName string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(CharacterEffectPlaySoundWithMuteBackgroundMusic), func(mode byte) packet.Encoder {
		return NewEffectWithMessage(mode, songName)
	})
}

func CharacterPlaySoundWithMuteBackgroundMusicEffectForeignBody(characterId uint32, songName string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(CharacterEffectPlaySoundWithMuteBackgroundMusic), func(mode byte) packet.Encoder {
		return NewEffectWithMessageForeign(characterId, mode, songName)
	})
}

func CharacterSoulStoneUseEffectBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(CharacterEffectSoulStoneUse), func(mode byte) packet.Encoder {
		return NewEffectSimple(mode)
	})
}

func CharacterSoulStoneUseEffectForeignBody(characterId uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(CharacterEffectSoulStoneUse), func(mode byte) packet.Encoder {
		return NewEffectSimpleForeign(characterId, mode)
	})
}
