package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type AgreementResponse struct {
	unk    uint32
	agreed bool
}

func (m AgreementResponse) Unk() uint32  { return m.unk }
func (m AgreementResponse) Agreed() bool { return m.agreed }

func (m AgreementResponse) Operation() string { return "AgreementResponse" }

func (m AgreementResponse) String() string {
	return fmt.Sprintf("unk [%d] agreed [%t]", m.unk, m.agreed)
}

func (m AgreementResponse) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.unk)
		w.WriteBool(m.agreed)
		return w.Bytes()
	}
}

func (m *AgreementResponse) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.unk = r.ReadUint32()
		m.agreed = r.ReadBool()
	}
}
