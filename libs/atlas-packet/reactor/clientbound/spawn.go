package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const ReactorSpawnWriter = "ReactorSpawn"

// hasReactorName reports whether the reactor-spawn body carries the trailing
// reactor name string.
//
// Unlike most v48 gaps this boundary is v79/v83, not v48/v61. IDA
// CReactorPool::OnReactorEnterField reads Decode4(id), Decode4(templateId),
// Decode1(state), Decode2(x), Decode2(y), Decode1(flags) on EVERY version; only
// v83 @0x735127, v84 @0x75271c, v87 @0x77af9c and v95 @0x6cf490 follow it with a
// DecodeStr. v48 @0x5a54b4 (read in full: the six decodes end at 0x5a5544 and
// the remainder of the function is layer/canvas setup with no further CInPacket
// call), v72 @0x69207c and v79 @0x6b77bb do not.
//
// Atlas wrote the name unconditionally, so all four legacy versions received a
// trailing length-prefixed string they never read.
func hasReactorName(t tenant.Model) bool {
	return (t.IsRegion("GMS") && t.MajorAtLeast(83)) || t.Region() == "JMS"
}

// packet-audit:fname CReactorPool::OnReactorEnterField
type Spawn struct {
	id             uint32
	classification uint32
	state          int8
	x              int16
	y              int16
	direction      byte
	name           string
}

func NewReactorSpawn(id uint32, classification uint32, state int8, x int16, y int16, direction byte, name string) Spawn {
	return Spawn{id: id, classification: classification, state: state, x: x, y: y, direction: direction, name: name}
}

func (m Spawn) Id() uint32             { return m.id }
func (m Spawn) Classification() uint32 { return m.classification }
func (m Spawn) State() int8            { return m.state }
func (m Spawn) X() int16               { return m.x }
func (m Spawn) Y() int16               { return m.y }
func (m Spawn) Direction() byte        { return m.direction }
func (m Spawn) Name() string           { return m.name }
func (m Spawn) Operation() string      { return ReactorSpawnWriter }
func (m Spawn) String() string {
	return fmt.Sprintf("id [%d], classification [%d], state [%d]", m.id, m.classification, m.state)
}

func (m Spawn) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.id)
		w.WriteInt(m.classification)
		w.WriteInt8(m.state)
		w.WriteInt16(m.x)
		w.WriteInt16(m.y)
		w.WriteByte(m.direction)
		if hasReactorName(t) {
			w.WriteAsciiString(m.name)
		}
		return w.Bytes()
	}
}

func (m *Spawn) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.id = r.ReadUint32()
		m.classification = r.ReadUint32()
		m.state = r.ReadInt8()
		m.x = r.ReadInt16()
		m.y = r.ReadInt16()
		m.direction = r.ReadByte()
		if hasReactorName(t) {
			m.name = r.ReadAsciiString()
		}
	}
}
