package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const SheepRanchInfoWriter = "SheepRanchInfo"

// packet-audit:fname CField_Battlefield::OnScoreUpdate
type SheepRanchInfo struct {
	wolfCount          byte
	wolfDisguisedCount byte
}

func NewSheepRanchInfo(wolfCount byte, wolfDisguisedCount byte) SheepRanchInfo {
	return SheepRanchInfo{wolfCount: wolfCount, wolfDisguisedCount: wolfDisguisedCount}
}

func (m SheepRanchInfo) WolfCount() byte          { return m.wolfCount }
func (m SheepRanchInfo) WolfDisguisedCount() byte { return m.wolfDisguisedCount }

func (m SheepRanchInfo) Operation() string { return SheepRanchInfoWriter }
func (m SheepRanchInfo) String() string {
	return fmt.Sprintf("wolfCount [%d] wolfDisguisedCount [%d]", m.wolfCount, m.wolfDisguisedCount)
}

func (m SheepRanchInfo) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.wolfCount)
		w.WriteByte(m.wolfDisguisedCount)
		return w.Bytes()
	}
}

func (m *SheepRanchInfo) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.wolfCount = r.ReadByte()
		m.wolfDisguisedCount = r.ReadByte()
	}
}
