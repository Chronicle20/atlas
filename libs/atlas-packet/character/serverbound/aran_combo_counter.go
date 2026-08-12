package serverbound

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

const AranComboCounterHandle = "AranComboCounterHandle"

// AranComboCounterRequest - CUserLocal::RequestIncCombo. Sent from
// CMob::OnHit when the local Aran lands a damaging hit with a polearm while
// owning Combo Ability. The body is empty on every in-scope version (v83,
// v84, v87, v92, v95, jms185 -- design.md §2.1), so there is nothing here to
// trust: every gate is re-derived server-side.
type AranComboCounterRequest struct{}

func (m AranComboCounterRequest) Operation() string {
	return AranComboCounterHandle
}

func (m AranComboCounterRequest) String() string {
	return ""
}

func (m AranComboCounterRequest) Encode(_ logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	return func(options map[string]interface{}) []byte {
		return []byte{}
	}
}

func (m *AranComboCounterRequest) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
	}
}
