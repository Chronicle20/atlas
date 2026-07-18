package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const OxQuizWriter = "OxQuiz"

// packet-audit:fname CField::OnQuiz
type OxQuiz struct {
	enabled  byte
	category byte
	number   uint16
}

func NewOxQuiz(enabled byte, category byte, number uint16) OxQuiz {
	return OxQuiz{enabled: enabled, category: category, number: number}
}

func (m OxQuiz) Enabled() byte  { return m.enabled }
func (m OxQuiz) Category() byte { return m.category }
func (m OxQuiz) Number() uint16 { return m.number }

func (m OxQuiz) Operation() string { return OxQuizWriter }
func (m OxQuiz) String() string {
	return fmt.Sprintf("enabled [%d] category [%d] number [%d]", m.enabled, m.category, m.number)
}

func (m OxQuiz) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.enabled)
		w.WriteByte(m.category)
		w.WriteShort(m.number)
		return w.Bytes()
	}
}

func (m *OxQuiz) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.enabled = r.ReadByte()
		m.category = r.ReadByte()
		m.number = r.ReadUint16()
	}
}
