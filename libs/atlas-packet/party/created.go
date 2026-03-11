package party

import (
	"context"
	"fmt"

	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const PartyOperationWriter = "PartyOperation"

type Created struct {
	mode    byte
	partyId uint32
}

func NewCreated(mode byte, partyId uint32) Created {
	return Created{mode: mode, partyId: partyId}
}

func (m Created) Mode() byte      { return m.mode }
func (m Created) PartyId() uint32 { return m.partyId }

func (m Created) Operation() string {
	return PartyOperationWriter
}

func (m Created) String() string {
	return fmt.Sprintf("mode [%d], partyId [%d]", m.mode, m.partyId)
}

func (m Created) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.partyId)
		w.WriteInt(uint32(_map.EmptyMapId))
		w.WriteInt(uint32(_map.EmptyMapId))
		w.WriteShort(0)
		w.WriteShort(0)
		return w.Bytes()
	}
}

func (m *Created) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.partyId = r.ReadUint32()
		_ = r.ReadUint32() // EmptyMapId
		_ = r.ReadUint32() // EmptyMapId
		_ = r.ReadUint16() // door x
		_ = r.ReadUint16() // door y
	}
}
