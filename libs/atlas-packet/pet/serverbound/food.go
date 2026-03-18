package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const PetFoodHandle = "PetFoodHandle"

type Food struct {
	updateTime uint32
	source     int16
	itemId     uint32
}

func (m Food) UpdateTime() uint32 {
	return m.updateTime
}

func (m Food) Source() int16 {
	return m.source
}

func (m Food) ItemId() uint32 {
	return m.itemId
}

func (m Food) Operation() string {
	return PetFoodHandle
}

func (m Food) String() string {
	return fmt.Sprintf("updateTime [%d] source [%d] itemId [%d]", m.updateTime, m.source, m.itemId)
}

func (m Food) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.updateTime)
		w.WriteInt16(m.source)
		w.WriteInt(m.itemId)
		return w.Bytes()
	}
}

func (m *Food) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.updateTime = r.ReadUint32()
		m.source = r.ReadInt16()
		m.itemId = r.ReadUint32()
	}
}
