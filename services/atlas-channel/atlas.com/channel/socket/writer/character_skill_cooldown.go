package writer

import (
	"context"
	"time"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const (
	CharacterSkillCooldown = "CharacterSkillCooldown"
)

func CharacterSkillCooldownBody(skillId uint32, cooldownExpiresAt time.Time) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
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
