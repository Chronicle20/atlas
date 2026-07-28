package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const PyramidScoreWriter = "PyramidScore"

// packet-audit:fname CField_MassacreResult::OnMassacreResult
type PyramidScore struct {
	rank  byte
	score uint32
}

func NewPyramidScore(rank byte, score uint32) PyramidScore {
	return PyramidScore{rank: rank, score: score}
}

func (m PyramidScore) Rank() byte    { return m.rank }
func (m PyramidScore) Score() uint32 { return m.score }

func (m PyramidScore) Operation() string { return PyramidScoreWriter }
func (m PyramidScore) String() string {
	return fmt.Sprintf("rank [%d] score [%d]", m.rank, m.score)
}

func (m PyramidScore) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.rank)
		w.WriteInt(m.score)
		return w.Bytes()
	}
}

func (m *PyramidScore) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.rank = r.ReadByte()
		m.score = r.ReadUint32()
	}
}
