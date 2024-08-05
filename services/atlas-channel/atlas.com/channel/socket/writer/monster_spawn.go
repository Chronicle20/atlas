package writer

import (
	"atlas-channel/monster"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const SpawnMonster = "SpawnMonster"

func SpawnMonsterBody(l logrus.FieldLogger) func(m monster.Model, newSpawn bool) BodyProducer {
	return func(m monster.Model, newSpawn bool) BodyProducer {
		return SpawnMonsterWithEffectBody(l)(m, newSpawn, 0)
	}
}

func SpawnMonsterWithEffectBody(l logrus.FieldLogger) func(m monster.Model, newSpawn bool, effect byte) BodyProducer {
	return func(m monster.Model, newSpawn bool, effect byte) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(m.UniqueId())
			if m.Controlled() {
				w.WriteByte(1)
			} else {
				w.WriteByte(5)
			}
			w.WriteInt(m.MonsterId())

			//TODO fix this. CMob::SetTemporaryStat
			w.WriteInt(0)
			w.WriteInt(0)
			w.WriteInt(0)
			w.WriteInt(0)
			w.WriteInt16(m.X())
			w.WriteInt16(m.Y())
			w.WriteByte(m.Stance())
			w.WriteInt16(0)
			w.WriteInt16(m.Fh())

			//TODO handle parent mobs
			encodeParentlessMobSpawnEffect(w, newSpawn, effect)

			w.WriteInt8(m.Team())
			w.WriteInt(0) // getItemEffect
			return w.Bytes()
		}
	}
}

func encodeParentlessMobSpawnEffect(w *response.Writer, newSpawn bool, effect byte) {
	if effect > 0 {
		w.WriteByte(effect)
		w.WriteByte(0)
		w.WriteShort(0)
		if effect == 15 {
			w.WriteByte(0)
		}
	}
	if newSpawn {
		w.WriteInt8(-2)
	} else {
		w.WriteInt8(-1)
	}
}
