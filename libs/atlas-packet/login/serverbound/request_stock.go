package serverbound

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

// decodeStock implements the stock-Nexon v95 LoginHandle.Request wire shape.
// Full implementation lands with the Nexon-passport sibling task; for now
// this is a slot/stub that satisfies the dispatch branch. Real field
// parsing (sPasswd, szPassport, partnerCode, machineId, gameStartMode, etc.)
// lands in the passport-validator sibling task.
func (m *Request) decodeStock(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) {
		// Slot stub — sibling task implements real stock-v95 decode.
		_ = r
	}
}
