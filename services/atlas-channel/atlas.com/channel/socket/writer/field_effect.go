package writer

import (
	"context"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const FieldEffect = "FieldEffect"

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
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getFieldEffect(l)(options, FieldEffectSummon))
			w.WriteByte(effect)
			w.WriteInt32(x)
			w.WriteInt32(y)
			return w.Bytes()
		}
	}
}

func FieldEffectTrembleBody(bHeavyNShortTremble bool, delay uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getFieldEffect(l)(options, FieldEffectTremble))
			w.WriteBool(bHeavyNShortTremble)
			w.WriteInt(delay)
			return w.Bytes()
		}
	}
}

func FieldEffectObjectBody(name string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getFieldEffect(l)(options, FieldEffectObject))
			w.WriteAsciiString(name)
			return w.Bytes()
		}
	}
}

// FieldEffectScreenBody - path parameter is in relation to Map.wz/Effect.img
func FieldEffectScreenBody(path string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getFieldEffect(l)(options, FieldEffectScreen))
			w.WriteAsciiString(path)
			return w.Bytes()
		}
	}
}

func FieldEffectSoundBody(path string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getFieldEffect(l)(options, FieldEffectSound))
			w.WriteAsciiString(path)
			return w.Bytes()
		}
	}
}

func FieldEffectBossHpBody(monsterId uint32, currentHp uint32, maxHp uint32, tagColor byte, tagBackgroundColor byte) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getFieldEffect(l)(options, FieldEffectBossHp))
			w.WriteInt(monsterId)
			w.WriteInt(currentHp)
			w.WriteInt(maxHp)
			w.WriteByte(tagColor)
			w.WriteByte(tagBackgroundColor)
			return w.Bytes()
		}
	}
}

func FieldEffectBackgroundMusicBody(name string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getFieldEffect(l)(options, FieldEffectBackgroundMusic))
			w.WriteAsciiString(name)
			return w.Bytes()
		}
	}
}

func FieldEffectRewardRulletBody(nRewardJobIdx uint32, nRewardPartIdx uint32, nRewardLevIdx uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getFieldEffect(l)(options, FieldEffectRewardRullet))
			w.WriteInt(nRewardJobIdx)
			w.WriteInt(nRewardPartIdx)
			w.WriteInt(nRewardLevIdx)
			return w.Bytes()
		}
	}
}

func getFieldEffect(l logrus.FieldLogger) func(options map[string]interface{}, key FieldEffectMode) byte {
	return func(options map[string]interface{}, key FieldEffectMode) byte {
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
