package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const ClaimAvailableTimeWriter = "ClaimAvailableTime"

// ClaimAvailableTime - CWvsContext::OnSetClaimSvrAvailableTime. Two bytes:
// open hour -> m_nClaimSvrOpenTime, close hour -> m_nClaimSvrCloseTime.
// open == close == 0 is treated by the client as always-available.
// Byte-identical v72..v95 (opcode 0x2B on v72/v79, 0x2E on v83-v87, 0x2D on v95).
// packet-audit:fname CWvsContext::OnSetClaimSvrAvailableTime
type ClaimAvailableTime struct {
	openHour  byte
	closeHour byte
}

func NewClaimAvailableTime(openHour byte, closeHour byte) ClaimAvailableTime {
	return ClaimAvailableTime{openHour: openHour, closeHour: closeHour}
}

func (m ClaimAvailableTime) OpenHour() byte  { return m.openHour }
func (m ClaimAvailableTime) CloseHour() byte { return m.closeHour }

func (m ClaimAvailableTime) Operation() string { return ClaimAvailableTimeWriter }

func (m ClaimAvailableTime) String() string {
	return fmt.Sprintf("open [%d] close [%d]", m.openHour, m.closeHour)
}

func (m ClaimAvailableTime) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.openHour)
		w.WriteByte(m.closeHour)
		return w.Bytes()
	}
}

func (m *ClaimAvailableTime) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.openHour = r.ReadByte()
		m.closeHour = r.ReadByte()
	}
}
