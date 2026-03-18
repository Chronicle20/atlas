package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type ItemUsePetConsumable struct {
	updateTime      uint32
	updateTimeFirst bool
}

func NewItemUsePetConsumable(updateTimeFirst bool) *ItemUsePetConsumable {
	return &ItemUsePetConsumable{updateTimeFirst: updateTimeFirst}
}

func (m ItemUsePetConsumable) UpdateTime() uint32 { return m.updateTime }

func (m ItemUsePetConsumable) Operation() string { return "ItemUsePetConsumable" }

func (m ItemUsePetConsumable) String() string {
	return fmt.Sprintf("updateTime [%d]", m.updateTime)
}

func (m ItemUsePetConsumable) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		if !m.updateTimeFirst {
			w.WriteInt(m.updateTime)
		}
		return w.Bytes()
	}
}

func (m *ItemUsePetConsumable) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		if !m.updateTimeFirst {
			m.updateTime = r.ReadUint32()
		}
	}
}
