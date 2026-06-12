package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const SummonMoveHandle = "SummonMoveHandle"

// Move is the client -> server MOVE_SUMMON packet, decoded per Cosmic
// MoveSummonHandler.handlePacket (MoveSummonHandler.java:36-59): int oid,
// startPos (short x, short y), then the raw movement blob (remaining bytes).
type Move struct {
	oid         uint32
	startX      int16
	startY      int16
	rawMovement []byte
}

func (m Move) Oid() uint32         { return m.oid }
func (m Move) StartX() int16       { return m.startX }
func (m Move) StartY() int16       { return m.startY }
func (m Move) RawMovement() []byte { return m.rawMovement }

func (m Move) Operation() string { return SummonMoveHandle }

func (m Move) String() string {
	return fmt.Sprintf("oid [%d], startX [%d], startY [%d], rawMovement [%d bytes]", m.oid, m.startX, m.startY, len(m.rawMovement))
}

func (m Move) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	_ = tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.oid)
		w.WriteInt16(m.startX)
		w.WriteInt16(m.startY)
		w.WriteByteArray(m.rawMovement)
		return w.Bytes()
	}
}

func (m *Move) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	_ = tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.oid = r.ReadUint32()
		m.startX = r.ReadInt16()
		m.startY = r.ReadInt16()
		m.rawMovement = r.ReadBytes(r.Available())
	}
}
