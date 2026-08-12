package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const ShowComboWriter = "ShowCombo"

// ShowCombo - CUserLocal::OnIncComboResponse. Body is one 4-byte
// little-endian combo count, decoded into m_nCombo and handed to DrawCombo
// (design.md §2.2). Identical on every in-scope version.
//
// Never send a count of 0: DrawCombo early-returns on a non-positive count
// WITHOUT releasing its digit layers, so a 0 leaves stale digits on screen
// rather than clearing them. The client clears its own HUD on its idle timer
// (design.md §2.5, §5.3).
type ShowCombo struct {
	count uint32
}

func NewShowCombo(count uint32) ShowCombo {
	return ShowCombo{count: count}
}

func (m ShowCombo) Count() uint32     { return m.count }
func (m ShowCombo) Operation() string { return ShowComboWriter }
func (m ShowCombo) String() string    { return fmt.Sprintf("count [%d]", m.count) }

func (m ShowCombo) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.count)
		return w.Bytes()
	}
}

func (m *ShowCombo) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.count = r.ReadUint32()
	}
}
