package party

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type ChangeLeaderW struct {
	mode              byte
	targetCharacterId uint32
	disconnected      bool
}

func NewChangeLeaderW(mode byte, targetCharacterId uint32, disconnected bool) ChangeLeaderW {
	return ChangeLeaderW{mode: mode, targetCharacterId: targetCharacterId, disconnected: disconnected}
}

func (m ChangeLeaderW) Mode() byte                { return m.mode }
func (m ChangeLeaderW) TargetCharacterId() uint32  { return m.targetCharacterId }
func (m ChangeLeaderW) Disconnected() bool         { return m.disconnected }

func (m ChangeLeaderW) Operation() string {
	return PartyOperationWriter
}

func (m ChangeLeaderW) String() string {
	return fmt.Sprintf("mode [%d], targetCharacterId [%d], disconnected [%t]", m.mode, m.targetCharacterId, m.disconnected)
}

func (m ChangeLeaderW) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.targetCharacterId)
		w.WriteBool(m.disconnected)
		return w.Bytes()
	}
}

func (m *ChangeLeaderW) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.targetCharacterId = r.ReadUint32()
		m.disconnected = r.ReadBool()
	}
}
