package writer

import (
	"context"
	"time"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const (
	CharacterSkillChange = "CharacterSkillChange"
)

func CharacterSkillChangeBody(exclRequestSent bool, skillId uint32, level byte, masterLevel byte, expiration time.Time, sn bool) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
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
