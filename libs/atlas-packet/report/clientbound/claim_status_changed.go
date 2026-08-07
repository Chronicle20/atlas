package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const ClaimSvrStatusChangedWriter = "ClaimSvrStatusChanged"

// ClaimSvrStatusChanged - CWvsContext::OnClaimSvrStatusChanged. One byte
// connected flag -> m_bClaimSvrConnected. Without connected = 1 the client
// refuses to open the claim dialog. Byte-identical v72..v95.
// packet-audit:fname CWvsContext::OnClaimSvrStatusChanged
type ClaimSvrStatusChanged struct {
	connected bool
}

func NewClaimSvrStatusChanged(connected bool) ClaimSvrStatusChanged {
	return ClaimSvrStatusChanged{connected: connected}
}

func (m ClaimSvrStatusChanged) Connected() bool { return m.connected }

func (m ClaimSvrStatusChanged) Operation() string { return ClaimSvrStatusChangedWriter }

func (m ClaimSvrStatusChanged) String() string {
	return fmt.Sprintf("connected [%t]", m.connected)
}

func (m ClaimSvrStatusChanged) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteBool(m.connected)
		return w.Bytes()
	}
}

func (m *ClaimSvrStatusChanged) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.connected = r.ReadBool()
	}
}
