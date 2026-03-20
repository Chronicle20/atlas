package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type ItemUseChalkboard struct {
	message         string
	updateTime      uint32
	updateTimeFirst bool
}

func NewItemUseChalkboard(updateTimeFirst bool) *ItemUseChalkboard {
	return &ItemUseChalkboard{updateTimeFirst: updateTimeFirst}
}

func (m ItemUseChalkboard) Message() string    { return m.message }
func (m ItemUseChalkboard) UpdateTime() uint32 { return m.updateTime }

func (m ItemUseChalkboard) Operation() string { return "ItemUseChalkboard" }

func (m ItemUseChalkboard) String() string {
	return fmt.Sprintf("message [%s] updateTime [%d]", m.message, m.updateTime)
}

func (m ItemUseChalkboard) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.message)
		if !m.updateTimeFirst {
			w.WriteInt(m.updateTime)
		}
		return w.Bytes()
	}
}

func (m *ItemUseChalkboard) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.message = r.ReadAsciiString()
		if !m.updateTimeFirst {
			m.updateTime = r.ReadUint32()
		}
	}
}
