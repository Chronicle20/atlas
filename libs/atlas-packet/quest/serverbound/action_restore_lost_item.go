package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type ActionRestoreLostItem struct {
	unk1   uint32
	itemId uint32
}

func (m ActionRestoreLostItem) Unk1() uint32   { return m.unk1 }
func (m ActionRestoreLostItem) ItemId() uint32 { return m.itemId }

func (m ActionRestoreLostItem) Operation() string { return "ActionRestoreLostItem" }

func (m ActionRestoreLostItem) String() string {
	return fmt.Sprintf("unk1 [%d] itemId [%d]", m.unk1, m.itemId)
}

func (m ActionRestoreLostItem) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.unk1)
		w.WriteInt(m.itemId)
		return w.Bytes()
	}
}

func (m *ActionRestoreLostItem) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.unk1 = r.ReadUint32()
		m.itemId = r.ReadUint32()
	}
}
