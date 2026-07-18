package clientbound

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const FieldObstacleAllResetWriter = "FieldObstacleAllReset"

// FieldObstacleAllReset is the clientbound CField::OnFieldObstacleAllReset packet.
// It carries no payload; the client resets all field obstacles to their default
// state on receipt.
// packet-audit:fname CField::OnFieldObstacleAllReset
type FieldObstacleAllReset struct{}

func NewFieldObstacleAllReset() FieldObstacleAllReset {
	return FieldObstacleAllReset{}
}

func (m FieldObstacleAllReset) Operation() string { return FieldObstacleAllResetWriter }
func (m FieldObstacleAllReset) String() string {
	return "FieldObstacleAllReset"
}

func (m FieldObstacleAllReset) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		return w.Bytes()
	}
}

func (m *FieldObstacleAllReset) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
	}
}
