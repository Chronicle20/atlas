package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const WeddingProgressWriter = "WeddingProgress"

// packet-audit:fname CField_Wedding::OnWeddingProgress
type WeddingProgress struct {
	step    byte
	groomId uint32
	brideId uint32
}

func NewWeddingProgress(step byte, groomId uint32, brideId uint32) WeddingProgress {
	return WeddingProgress{step: step, groomId: groomId, brideId: brideId}
}

func (m WeddingProgress) Step() byte       { return m.step }
func (m WeddingProgress) GroomId() uint32  { return m.groomId }
func (m WeddingProgress) BrideId() uint32  { return m.brideId }

func (m WeddingProgress) Operation() string { return WeddingProgressWriter }
func (m WeddingProgress) String() string {
	return fmt.Sprintf("step [%d] groomId [%d] brideId [%d]", m.step, m.groomId, m.brideId)
}

func (m WeddingProgress) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	hasStep := t.Region() != "JMS"
	return func(options map[string]interface{}) []byte {
		if hasStep {
			w.WriteByte(m.step)
		}
		w.WriteInt(m.groomId)
		w.WriteInt(m.brideId)
		return w.Bytes()
	}
}

func (m *WeddingProgress) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	hasStep := t.Region() != "JMS"
	return func(r *request.Reader, options map[string]interface{}) {
		if hasStep {
			m.step = r.ReadByte()
		}
		m.groomId = r.ReadUint32()
		m.brideId = r.ReadUint32()
	}
}
