package writer

import (
	"context"

	atlas_packet "github.com/Chronicle20/atlas-packet"
	fieldpkt "github.com/Chronicle20/atlas-packet/field"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)


type FieldEffectMode string

// CField::OnFieldEffect

const (
	FieldEffectSummon          FieldEffectMode = "SUMMON"           // 0
	FieldEffectTremble         FieldEffectMode = "TREMBLE"          // 1
	FieldEffectObject          FieldEffectMode = "OBJECT"           // 2
	FieldEffectScreen          FieldEffectMode = "SCREEN"           // 3
	FieldEffectSound           FieldEffectMode = "SOUND"            // 4
	FieldEffectBossHp          FieldEffectMode = "BOSS_HP"          // 5
	FieldEffectBackgroundMusic FieldEffectMode = "BACKGROUND_MUSIC" // 6
	FieldEffectRewardRullet    FieldEffectMode = "REWARD_RULLET"    // 7
)

func FieldEffectSummonBody(effect byte, x int32, y int32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getFieldEffect(l)(options, FieldEffectSummon)
			return fieldpkt.NewFieldEffectSummon(mode, effect, x, y).Encode(l, ctx)(options)
		}
	}
}

func FieldEffectTrembleBody(bHeavyNShortTremble bool, delay uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getFieldEffect(l)(options, FieldEffectTremble)
			return fieldpkt.NewFieldEffectTremble(mode, bHeavyNShortTremble, delay).Encode(l, ctx)(options)
		}
	}
}

func FieldEffectObjectBody(name string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getFieldEffect(l)(options, FieldEffectObject)
			return fieldpkt.NewFieldEffectObject(mode, name).Encode(l, ctx)(options)
		}
	}
}

// FieldEffectScreenBody - path parameter is in relation to Map.wz/Effect.img
func FieldEffectScreenBody(path string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getFieldEffect(l)(options, FieldEffectScreen)
			return fieldpkt.NewFieldEffectScreen(mode, path).Encode(l, ctx)(options)
		}
	}
}

func FieldEffectSoundBody(path string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getFieldEffect(l)(options, FieldEffectSound)
			return fieldpkt.NewFieldEffectSound(mode, path).Encode(l, ctx)(options)
		}
	}
}

func FieldEffectBossHpBody(monsterId uint32, currentHp uint32, maxHp uint32, tagColor byte, tagBackgroundColor byte) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getFieldEffect(l)(options, FieldEffectBossHp)
			return fieldpkt.NewFieldEffectBossHp(mode, monsterId, currentHp, maxHp, tagColor, tagBackgroundColor).Encode(l, ctx)(options)
		}
	}
}

func FieldEffectBackgroundMusicBody(name string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getFieldEffect(l)(options, FieldEffectBackgroundMusic)
			return fieldpkt.NewFieldEffectBackgroundMusic(mode, name).Encode(l, ctx)(options)
		}
	}
}

func FieldEffectRewardRulletBody(nRewardJobIdx uint32, nRewardPartIdx uint32, nRewardLevIdx uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getFieldEffect(l)(options, FieldEffectRewardRullet)
			return fieldpkt.NewFieldEffectRewardRullet(mode, nRewardJobIdx, nRewardPartIdx, nRewardLevIdx).Encode(l, ctx)(options)
		}
	}
}

func getFieldEffect(l logrus.FieldLogger) func(options map[string]interface{}, key FieldEffectMode) byte {
	return func(options map[string]interface{}, key FieldEffectMode) byte {
		return atlas_packet.ResolveCode(l, options, "operations", string(key))
	}
}
