package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"time"
)

const (
	CharacterSkillChange = "CharacterSkillChange"
)

func CharacterSkillChangeBody(l logrus.FieldLogger, t tenant.Model) func(exclRequestSent bool, skillId uint32, level byte, masterLevel byte, expiration time.Time, sn bool) BodyProducer {
	return func(exclRequestSent bool, skillId uint32, level byte, masterLevel byte, expiration time.Time, sn bool) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteBool(exclRequestSent)
			w.WriteShort(1) // # of skills being updated
			w.WriteInt(skillId)
			w.WriteInt(uint32(level))
			w.WriteInt(uint32(masterLevel))
			w.WriteInt64(msTime(expiration))
			w.WriteBool(sn)
			return w.Bytes()
		}
	}
}
