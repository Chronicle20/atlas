package clientbound

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const AriantArenaShowResultWriter = "AriantArenaShowResult"

// packet-audit:fname CField_AriantArena::OnShowResult
type AriantArenaShowResult struct {
}

func NewAriantArenaShowResult() AriantArenaShowResult {
	return AriantArenaShowResult{}
}

func (m AriantArenaShowResult) Operation() string { return AriantArenaShowResultWriter }
func (m AriantArenaShowResult) String() string {
	return "AriantArenaShowResult"
}

func (m AriantArenaShowResult) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		return w.Bytes()
	}
}

func (m *AriantArenaShowResult) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
	}
}
