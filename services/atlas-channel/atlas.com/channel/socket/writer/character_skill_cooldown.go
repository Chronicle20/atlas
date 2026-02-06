package writer

import (
	"time"

	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const (
	CharacterSkillCooldown = "CharacterSkillCooldown"
)

func CharacterSkillCooldownBody(l logrus.FieldLogger, t tenant.Model) func(skillId uint32, cooldownExpiresAt time.Time) BodyProducer {
	return func(skillId uint32, cooldownExpiresAt time.Time) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(skillId)
			if cooldownExpiresAt.IsZero() {
				w.WriteShort(0)
			} else {
				cd := uint32(cooldownExpiresAt.Sub(time.Now()).Seconds())
				w.WriteShort(uint16(cd))
			}
			return w.Bytes()
		}
	}
}
