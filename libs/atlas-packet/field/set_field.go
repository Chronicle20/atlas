package field

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

// SetField is sent when a character enters a field with full character data.
// The characterInfoBytes are pre-encoded by the service layer since they depend
// on the full character model (stats, inventory, skills, quests, rings, etc.).
type SetField struct {
	channelId          channel.Id
	characterInfoBytes []byte
	damageSeeds        []uint32
	timestamp          int64
}

func NewSetField(channelId channel.Id, characterInfoBytes []byte) SetField {
	seeds := make([]uint32, 4)
	for i := range seeds {
		seeds[i] = rand.Uint32()
	}
	return SetField{
		channelId:          channelId,
		characterInfoBytes: characterInfoBytes,
		damageSeeds:        seeds,
		timestamp:          fieldMsTime(time.Now()),
	}
}

func (m SetField) Operation() string { return SetFieldWriter }
func (m SetField) String() string {
	return fmt.Sprintf("set field channelId [%d]", m.channelId)
}

func (m SetField) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		if (t.Region() == "GMS" && t.MajorVersion() > 83) || t.Region() == "JMS" {
			w.WriteShort(0) // decode opt
		}
		w.WriteInt(uint32(m.channelId))
		if t.Region() == "JMS" {
			w.WriteByte(0)
			w.WriteInt(0)
		}
		w.WriteByte(1) // sNotifierMessage
		w.WriteByte(1) // bCharacterData

		if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
			w.WriteShort(0) // nNotifierCheck
			// 3 damage seeds
			for i := 0; i < 3; i++ {
				w.WriteInt(m.damageSeeds[i])
			}
		} else {
			// 4 damage seeds
			for i := 0; i < 4; i++ {
				w.WriteInt(m.damageSeeds[i])
			}
		}

		w.WriteByteArray(m.characterInfoBytes)

		if (t.Region() == "GMS" && t.MajorVersion() > 83) || t.Region() == "JMS" {
			w.WriteInt(0) // logout gifts
			w.WriteInt(0)
			w.WriteInt(0)
			w.WriteInt(0)
		}
		w.WriteInt64(m.timestamp)
		return w.Bytes()
	}
}

func (m *SetField) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		// No-op: SetField is server-send-only with pre-encoded character info.
	}
}
