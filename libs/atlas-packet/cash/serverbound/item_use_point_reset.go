package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// ItemUsePointReset is the USE_CASH_ITEM sub-body for AP Reset (5050000) and
// SP Reset (5050001-5050004): two int32s read as To then From. For AP resets
// they are client stat flags; for SP resets they are skill ids. Layouts
// without an updateTime-first prefix carry a trailing updateTime. Read order
// is IDA-verified per version (see byte fixtures in the test file).
type ItemUsePointReset struct {
	to              uint32
	from            uint32
	updateTime      uint32
	updateTimeFirst bool
}

func NewItemUsePointReset(updateTimeFirst bool) *ItemUsePointReset {
	return &ItemUsePointReset{updateTimeFirst: updateTimeFirst}
}

func (m ItemUsePointReset) To() uint32         { return m.to }
func (m ItemUsePointReset) From() uint32       { return m.from }
func (m ItemUsePointReset) UpdateTime() uint32 { return m.updateTime }

func (m ItemUsePointReset) Operation() string { return "ItemUsePointReset" }

func (m ItemUsePointReset) String() string {
	return fmt.Sprintf("to [%d] from [%d] updateTime [%d]", m.to, m.from, m.updateTime)
}

func (m ItemUsePointReset) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.to)
		w.WriteInt(m.from)
		if !m.updateTimeFirst {
			w.WriteInt(m.updateTime)
		}
		return w.Bytes()
	}
}

func (m *ItemUsePointReset) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.to = r.ReadUint32()
		m.from = r.ReadUint32()
		if !m.updateTimeFirst {
			m.updateTime = r.ReadUint32()
		}
	}
}
