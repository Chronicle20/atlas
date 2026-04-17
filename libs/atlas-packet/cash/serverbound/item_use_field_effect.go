package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type ItemUseFieldEffect struct {
	message         string
	updateTime      uint32
	updateTimeFirst bool
}

func NewItemUseFieldEffect(updateTimeFirst bool) *ItemUseFieldEffect {
	return &ItemUseFieldEffect{updateTimeFirst: updateTimeFirst}
}

func (m ItemUseFieldEffect) Message() string    { return m.message }
func (m ItemUseFieldEffect) UpdateTime() uint32 { return m.updateTime }

func (m ItemUseFieldEffect) Operation() string { return "ItemUseFieldEffect" }

func (m ItemUseFieldEffect) String() string {
	return fmt.Sprintf("message [%s] updateTime [%d]", m.message, m.updateTime)
}

func (m ItemUseFieldEffect) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.message)
		if !m.updateTimeFirst {
			w.WriteInt(m.updateTime)
		}
		return w.Bytes()
	}
}

func (m *ItemUseFieldEffect) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.message = r.ReadAsciiString()
		if !m.updateTimeFirst {
			m.updateTime = r.ReadUint32()
		}
	}
}
