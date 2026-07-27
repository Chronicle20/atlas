package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const PartyOperationWriter = "PartyOperation"

// packet-audit:fname CWvsContext::OnPartyResult#Created
type Created struct {
	mode            byte
	partyId         uint32
	doorTownMapId   _map.Id
	doorTargetMapId _map.Id
	doorX           int16
	doorY           int16
}

func NewCreated(mode byte, partyId uint32) Created {
	return Created{
		mode:            mode,
		partyId:         partyId,
		doorTownMapId:   _map.EmptyMapId,
		doorTargetMapId: _map.EmptyMapId,
		doorX:           0,
		doorY:           0,
	}
}

// WithDoor returns a copy of the Created packet with the door fields populated.
// When door data is present the encoder writes the real town/target map ids and
// minimap x/y instead of the empty-map sentinel zeros (FR-3.3).
func (m Created) WithDoor(townMapId _map.Id, targetMapId _map.Id, x int16, y int16) Created {
	m.doorTownMapId = townMapId
	m.doorTargetMapId = targetMapId
	m.doorX = x
	m.doorY = y
	return m
}

func (m Created) Mode() byte               { return m.mode }
func (m Created) PartyId() uint32          { return m.partyId }
func (m Created) DoorTownMapId() _map.Id   { return m.doorTownMapId }
func (m Created) DoorTargetMapId() _map.Id { return m.doorTargetMapId }
func (m Created) DoorX() int16             { return m.doorX }
func (m Created) DoorY() int16             { return m.doorY }

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
		w.WriteInt(uint32(m.doorTownMapId))
		w.WriteInt(uint32(m.doorTargetMapId))
		w.WriteShort(uint16(m.doorX))
		w.WriteShort(uint16(m.doorY))
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
