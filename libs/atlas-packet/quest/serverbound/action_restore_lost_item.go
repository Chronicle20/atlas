package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type ActionRestoreLostItem struct {
	itemIds []uint32
}

func (m ActionRestoreLostItem) ItemIds() []uint32 { return m.itemIds }

func (m ActionRestoreLostItem) Operation() string { return "ActionRestoreLostItem" }

func (m ActionRestoreLostItem) String() string {
	return fmt.Sprintf("itemIds %v", m.itemIds)
}

func (m ActionRestoreLostItem) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(uint32(len(m.itemIds)))
		for _, id := range m.itemIds {
			w.WriteInt(id)
		}
		return w.Bytes()
	}
}

func (m *ActionRestoreLostItem) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		count := r.ReadUint32()
		m.itemIds = make([]uint32, count)
		for i := range m.itemIds {
			m.itemIds[i] = r.ReadUint32()
		}
	}
}
