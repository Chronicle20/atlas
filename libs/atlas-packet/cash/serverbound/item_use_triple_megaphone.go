package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// ItemUseTripleMegaphone is the USE_CASH_ITEM sub-body for the Triple
// Megaphone (5076xxx). Cosmic-derived (UseCashItemHandler case 4); per-version
// IDA verification in task-123 phases 19-20. Decode reads exactly the count
// carried on the wire; validating that count falls within 1..3 is the
// caller's responsibility (a later task), not this codec's. Encoded inline by
// SendConsumeCashItemUseRequest's CSpeakerWorldDlgEx case (task-19/task-123
// phase 20, gms_v95 IDA-verified): count(byte) + count×line(str) + whisper(bool).
// packet-audit:fname CWvsContext::SendConsumeCashItemUseRequest
type ItemUseTripleMegaphone struct {
	lines           []string
	whisper         bool
	updateTime      uint32
	updateTimeFirst bool
}

func NewItemUseTripleMegaphone(updateTimeFirst bool) *ItemUseTripleMegaphone {
	return &ItemUseTripleMegaphone{updateTimeFirst: updateTimeFirst}
}

func (m ItemUseTripleMegaphone) Lines() []string    { return m.lines }
func (m ItemUseTripleMegaphone) Whisper() bool      { return m.whisper }
func (m ItemUseTripleMegaphone) UpdateTime() uint32 { return m.updateTime }

func (m ItemUseTripleMegaphone) Operation() string { return "ItemUseTripleMegaphone" }

func (m ItemUseTripleMegaphone) String() string {
	return fmt.Sprintf("lines %v whisper [%t] updateTime [%d]", m.lines, m.whisper, m.updateTime)
}

func (m ItemUseTripleMegaphone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(byte(len(m.lines)))
		for _, ln := range m.lines {
			w.WriteAsciiString(ln)
		}
		w.WriteBool(m.whisper)
		if !m.updateTimeFirst {
			w.WriteInt(m.updateTime)
		}
		return w.Bytes()
	}
}

func (m *ItemUseTripleMegaphone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		count := r.ReadByte()
		m.lines = make([]string, 0, count)
		for i := byte(0); i < count; i++ {
			m.lines = append(m.lines, r.ReadAsciiString())
		}
		m.whisper = r.ReadBool()
		if !m.updateTimeFirst {
			m.updateTime = r.ReadUint32()
		}
	}
}
