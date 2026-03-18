package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type ChangeLeader struct {
	mode              byte
	targetCharacterId uint32
	disconnected      bool
}

func NewChangeLeader(mode byte, targetCharacterId uint32, disconnected bool) ChangeLeader {
	return ChangeLeader{mode: mode, targetCharacterId: targetCharacterId, disconnected: disconnected}
}

func (m ChangeLeader) Mode() byte                { return m.mode }
func (m ChangeLeader) TargetCharacterId() uint32  { return m.targetCharacterId }
func (m ChangeLeader) Disconnected() bool         { return m.disconnected }

func (m ChangeLeader) Operation() string {
	return PartyOperationWriter
}

func (m ChangeLeader) String() string {
	return fmt.Sprintf("mode [%d], targetCharacterId [%d], disconnected [%t]", m.mode, m.targetCharacterId, m.disconnected)
}

func (m ChangeLeader) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.targetCharacterId)
		w.WriteBool(m.disconnected)
		return w.Bytes()
	}
}

func (m *ChangeLeader) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.targetCharacterId = r.ReadUint32()
		m.disconnected = r.ReadBool()
	}
}
