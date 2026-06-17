package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const WitchTowerScoreUpdateWriter = "WitchTowerScoreUpdate"

// WitchTowerScoreUpdate is version-split. v83/v84/v87 (GMS<95) and jms read a
// single score byte via CField_Witchtower::OnScoreUpdate. GMS v95 routes this op
// (opcode 360) to CField::OnChaosZakumTimer, which reads the score byte followed
// by a uint32 seconds value. The trailing int is emitted only for GMS>=95.
// packet-audit:fname CField::OnChaosZakumTimer
type WitchTowerScoreUpdate struct {
	score   byte
	seconds uint32
}

func NewWitchTowerScoreUpdate(score byte, seconds uint32) WitchTowerScoreUpdate {
	return WitchTowerScoreUpdate{score: score, seconds: seconds}
}

func (m WitchTowerScoreUpdate) Score() byte     { return m.score }
func (m WitchTowerScoreUpdate) Seconds() uint32 { return m.seconds }

func (m WitchTowerScoreUpdate) Operation() string { return WitchTowerScoreUpdateWriter }
func (m WitchTowerScoreUpdate) String() string {
	return fmt.Sprintf("score [%d] seconds [%d]", m.score, m.seconds)
}

func (m WitchTowerScoreUpdate) hasSeconds(t tenant.Model) bool {
	return t.Region() == "GMS" && t.MajorAtLeast(95)
}

func (m WitchTowerScoreUpdate) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	hasSeconds := m.hasSeconds(t)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.score)
		if hasSeconds {
			w.WriteInt(m.seconds)
		}
		return w.Bytes()
	}
}

func (m *WitchTowerScoreUpdate) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	hasSeconds := m.hasSeconds(t)
	return func(r *request.Reader, options map[string]interface{}) {
		m.score = r.ReadByte()
		if hasSeconds {
			m.seconds = r.ReadUint32()
		}
	}
}
