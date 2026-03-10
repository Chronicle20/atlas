package monster

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const MonsterHealthWriter = "MonsterHealth"

type Health struct {
	uniqueId   uint32
	hpPercent  byte
}

func NewMonsterHealth(uniqueId uint32, hpPercent byte) Health {
	return Health{uniqueId: uniqueId, hpPercent: hpPercent}
}

func (m Health) UniqueId() uint32  { return m.uniqueId }
func (m Health) HpPercent() byte   { return m.hpPercent }
func (m Health) Operation() string { return MonsterHealthWriter }
func (m Health) String() string {
	return fmt.Sprintf("uniqueId [%d], hpPercent [%d]", m.uniqueId, m.hpPercent)
}

func (m Health) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.uniqueId)
		w.WriteByte(m.hpPercent)
		return w.Bytes()
	}
}

func (m *Health) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.uniqueId = r.ReadUint32()
		m.hpPercent = r.ReadByte()
	}
}
