package writer

import (
	"atlas-channel/socket/model"

	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const MonsterStatSet = "MonsterStatSet"
const MonsterStatReset = "MonsterStatReset"

func MonsterStatSetBody(l logrus.FieldLogger, t tenant.Model) func(uniqueId uint32, stat *model.MonsterTemporaryStat) BodyProducer {
	return func(uniqueId uint32, stat *model.MonsterTemporaryStat) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(uniqueId)
			stat.Encode(l, t, options)(w)
			w.WriteByte(byte(0))
			w.WriteInt(0)
			return w.Bytes()
		}
	}
}

func MonsterStatResetBody(l logrus.FieldLogger, t tenant.Model) func(uniqueId uint32, stat *model.MonsterTemporaryStat) BodyProducer {
	return func(uniqueId uint32, stat *model.MonsterTemporaryStat) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(uniqueId)
			stat.Encode(l, t, options)(w)
			w.WriteByte(byte(0))
			w.WriteInt(0)
			return w.Bytes()
		}
	}
}
