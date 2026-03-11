package field

import (
	"context"

	atlas_packet "github.com/Chronicle20/atlas-packet"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

type FieldEffectMode string

const (
	FieldEffectSummon          FieldEffectMode = "SUMMON"
	FieldEffectTremble         FieldEffectMode = "TREMBLE"
	FieldEffectObject          FieldEffectMode = "OBJECT"
	FieldEffectScreen          FieldEffectMode = "SCREEN"
	FieldEffectSound           FieldEffectMode = "SOUND"
	FieldEffectBossHp          FieldEffectMode = "BOSS_HP"
	FieldEffectBackgroundMusic FieldEffectMode = "BACKGROUND_MUSIC"
	FieldEffectRewardRullet    FieldEffectMode = "REWARD_RULLET"
)

func FieldEffectSummonBody(effect byte, x int32, y int32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(FieldEffectSummon), func(mode byte) packet.Encoder {
		return NewFieldEffectSummon(mode, effect, x, y)
	})
}

func FieldEffectTrembleBody(bHeavyNShortTremble bool, delay uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(FieldEffectTremble), func(mode byte) packet.Encoder {
		return NewFieldEffectTremble(mode, bHeavyNShortTremble, delay)
	})
}

func FieldEffectObjectBody(name string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(FieldEffectObject), func(mode byte) packet.Encoder {
		return NewFieldEffectObject(mode, name)
	})
}

// FieldEffectScreenBody - path parameter is in relation to Map.wz/Effect.img
func FieldEffectScreenBody(path string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(FieldEffectScreen), func(mode byte) packet.Encoder {
		return NewFieldEffectScreen(mode, path)
	})
}

func FieldEffectSoundBody(path string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(FieldEffectSound), func(mode byte) packet.Encoder {
		return NewFieldEffectSound(mode, path)
	})
}

func FieldEffectBossHpBody(monsterId uint32, currentHp uint32, maxHp uint32, tagColor byte, tagBackgroundColor byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(FieldEffectBossHp), func(mode byte) packet.Encoder {
		return NewFieldEffectBossHp(mode, monsterId, currentHp, maxHp, tagColor, tagBackgroundColor)
	})
}

func FieldEffectBackgroundMusicBody(name string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(FieldEffectBackgroundMusic), func(mode byte) packet.Encoder {
		return NewFieldEffectBackgroundMusic(mode, name)
	})
}

func FieldEffectRewardRulletBody(nRewardJobIdx uint32, nRewardPartIdx uint32, nRewardLevIdx uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(FieldEffectRewardRullet), func(mode byte) packet.Encoder {
		return NewFieldEffectRewardRullet(mode, nRewardJobIdx, nRewardPartIdx, nRewardLevIdx)
	})
}
