package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const BlockedServerWriter = "BlockedServer"

// BlockedServer is the clientbound CField::OnTransferChannelReqIgnored packet.
// A single byte reason for why a channel-transfer request was ignored.
// packet-audit:fname CField::OnTransferChannelReqIgnored
type BlockedServer struct {
	reason byte
}

func NewBlockedServer(reason byte) BlockedServer {
	return BlockedServer{reason: reason}
}

func (m BlockedServer) Reason() byte { return m.reason }

func (m BlockedServer) Operation() string { return BlockedServerWriter }
func (m BlockedServer) String() string {
	return fmt.Sprintf("reason [%d]", m.reason)
}

func (m BlockedServer) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.reason)
		return w.Bytes()
	}
}

func (m *BlockedServer) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.reason = r.ReadByte()
	}
}
