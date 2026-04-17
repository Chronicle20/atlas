package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterDropMesoHandle = "CharacterDropMesoHandle"

// DropMeso - CUser::SendDropMeso
type DropMeso struct {
	updateTime uint32
	amount     uint32
}

func (m DropMeso) UpdateTime() uint32 {
	return m.updateTime
}

func (m DropMeso) Amount() uint32 {
	return m.amount
}

func (m DropMeso) Operation() string {
	return CharacterDropMesoHandle
}

func (m DropMeso) String() string {
	return fmt.Sprintf("updateTime [%d], amount [%d]", m.updateTime, m.amount)
}

func (m DropMeso) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.updateTime)
		w.WriteInt(m.amount)
		return w.Bytes()
	}
}

func (m *DropMeso) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.updateTime = r.ReadUint32()
		m.amount = r.ReadUint32()
	}
}
