package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const AriantScoreWriter = "AriantScore"

// AriantScore is v95-only (opcode 358). v95 CField_Witchtower::OnPacket dispatches
// nType==358 to CField_Witchtower::OnScoreUpdate, which reads a single score byte.
// The op is absent from the v83/v84/v87/jms registries.
// packet-audit:fname CField_Witchtower::OnScoreUpdate
type AriantScore struct {
	score byte
}

func NewAriantScore(score byte) AriantScore {
	return AriantScore{score: score}
}

func (m AriantScore) Score() byte { return m.score }

func (m AriantScore) Operation() string { return AriantScoreWriter }
func (m AriantScore) String() string {
	return fmt.Sprintf("score [%d]", m.score)
}

func (m AriantScore) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.score)
		return w.Bytes()
	}
}

func (m *AriantScore) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.score = r.ReadByte()
	}
}
