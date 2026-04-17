package clientbound

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type SetField struct {
	channelId     channel.Id
	characterData charpkt.CharacterData
	damageSeeds   []uint32
	timestamp     int64
}

func NewSetField(channelId channel.Id, characterData charpkt.CharacterData) SetField {
	seeds := make([]uint32, 4)
	for i := range seeds {
		seeds[i] = rand.Uint32()
	}
	return SetField{
		channelId:     channelId,
		characterData: characterData,
		damageSeeds:   seeds,
		timestamp:     fieldMsTime(time.Now()),
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

		w.WriteByteArray(m.characterData.Encode(l, ctx)(options))

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

func (m SetField) CharacterData() charpkt.CharacterData { return m.characterData }

func (m *SetField) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		t := tenant.MustFromContext(ctx)

		if (t.Region() == "GMS" && t.MajorVersion() > 83) || t.Region() == "JMS" {
			_ = r.ReadUint16() // decode opt
		}
		m.channelId = channel.Id(r.ReadUint32())
		if t.Region() == "JMS" {
			_ = r.ReadByte()
			_ = r.ReadUint32()
		}
		_ = r.ReadByte() // sNotifierMessage
		_ = r.ReadByte() // bCharacterData

		if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
			_ = r.ReadUint16() // nNotifierCheck
			m.damageSeeds = make([]uint32, 4)
			for i := 0; i < 3; i++ {
				m.damageSeeds[i] = r.ReadUint32()
			}
		} else {
			m.damageSeeds = make([]uint32, 4)
			for i := 0; i < 4; i++ {
				m.damageSeeds[i] = r.ReadUint32()
			}
		}

		m.characterData.Decode(l, ctx)(r, options)

		if (t.Region() == "GMS" && t.MajorVersion() > 83) || t.Region() == "JMS" {
			_ = r.ReadUint32() // logout gifts
			_ = r.ReadUint32()
			_ = r.ReadUint32()
			_ = r.ReadUint32()
		}
		m.timestamp = r.ReadInt64()
	}
}
