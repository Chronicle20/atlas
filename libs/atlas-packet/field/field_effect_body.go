package field

import (
	"context"

	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	"github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
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
		return clientbound.NewFieldEffectSummon(mode, effect, x, y)
	})
}

func FieldEffectTrembleBody(bHeavyNShortTremble bool, delay uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(FieldEffectTremble), func(mode byte) packet.Encoder {
		return clientbound.NewFieldEffectTremble(mode, bHeavyNShortTremble, delay)
	})
}

func FieldEffectObjectBody(name string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(FieldEffectObject), func(mode byte) packet.Encoder {
		return clientbound.NewFieldEffectObject(mode, name)
	})
}

// FieldEffectScreenBody - path parameter is in relation to Map.wz/Effect.img
func FieldEffectScreenBody(path string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(FieldEffectScreen), func(mode byte) packet.Encoder {
		return clientbound.NewFieldEffectScreen(mode, path)
	})
}

func FieldEffectSoundBody(path string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(FieldEffectSound), func(mode byte) packet.Encoder {
		return clientbound.NewFieldEffectSound(mode, path)
	})
}

func FieldEffectBossHpBody(monsterId uint32, currentHp uint32, maxHp uint32, tagColor byte, tagBackgroundColor byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(FieldEffectBossHp), func(mode byte) packet.Encoder {
		return clientbound.NewFieldEffectBossHp(mode, monsterId, currentHp, maxHp, tagColor, tagBackgroundColor)
	})
}

func FieldEffectBackgroundMusicBody(name string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(FieldEffectBackgroundMusic), func(mode byte) packet.Encoder {
		return clientbound.NewFieldEffectBackgroundMusic(mode, name)
	})
}

func FieldEffectRewardRulletBody(nRewardJobIdx uint32, nRewardPartIdx uint32, nRewardLevIdx uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(FieldEffectRewardRullet), func(mode byte) packet.Encoder {
		return clientbound.NewFieldEffectRewardRullet(mode, nRewardJobIdx, nRewardPartIdx, nRewardLevIdx)
	})
}
