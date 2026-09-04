package clientbound

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const ClearBackEffectWriter = "ClearBackEffect"

// ClearBackEffect is the clientbound CMapLoadable::OnClearBackEffect packet. It
// carries no payload; the handler thunks straight to CMapLoadable::ReloadBack,
// which rebuilds the whole back-layer set. The effect is therefore field-wide —
// every page is cleared at once, and there is no per-page clear on the wire.
// packet-audit:fname CMapLoadable::OnClearBackEffect
type ClearBackEffect struct{}

func NewClearBackEffect() ClearBackEffect {
	return ClearBackEffect{}
}

func (m ClearBackEffect) Operation() string { return ClearBackEffectWriter }
func (m ClearBackEffect) String() string {
	return "ClearBackEffect"
}

func (m ClearBackEffect) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		return w.Bytes()
	}
}

func (m *ClearBackEffect) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
	}
}
