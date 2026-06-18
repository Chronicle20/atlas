package clientbound

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const WeddingCeremonyEndWriter = "WeddingCeremonyEnd"

// packet-audit:fname CField_Wedding::OnWeddingCeremonyEnd
type WeddingCeremonyEnd struct {
}

func NewWeddingCeremonyEnd() WeddingCeremonyEnd {
	return WeddingCeremonyEnd{}
}

func (m WeddingCeremonyEnd) Operation() string { return WeddingCeremonyEndWriter }
func (m WeddingCeremonyEnd) String() string {
	return "WeddingCeremonyEnd"
}

func (m WeddingCeremonyEnd) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		return w.Bytes()
	}
}

func (m *WeddingCeremonyEnd) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
	}
}
