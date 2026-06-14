package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const SummonMoveWriter = "SummonMove"

// SummonMove is the server -> client MOVE_SUMMON packet (Cosmic
// PacketCreator.moveSummon): int cid, int oid, startPos (short x, short y),
// then the raw movement blob rebroadcast byte-faithfully from the inbound
// MoveSummonHandler packet.
type SummonMove struct {
	cid         uint32
	oid         uint32
	startX      int16
	startY      int16
	rawMovement []byte
}

func NewSummonMove(cid, oid uint32, startX, startY int16, rawMovement []byte) SummonMove {
	return SummonMove{
		cid:         cid,
		oid:         oid,
		startX:      startX,
		startY:      startY,
		rawMovement: rawMovement,
	}
}

func (m SummonMove) Cid() uint32         { return m.cid }
func (m SummonMove) Oid() uint32         { return m.oid }
func (m SummonMove) StartX() int16       { return m.startX }
func (m SummonMove) StartY() int16       { return m.startY }
func (m SummonMove) RawMovement() []byte { return m.rawMovement }
func (m SummonMove) Operation() string   { return SummonMoveWriter }
func (m SummonMove) String() string {
	return fmt.Sprintf("cid [%d], oid [%d], startX [%d], startY [%d], rawMovement [%d bytes]", m.cid, m.oid, m.startX, m.startY, len(m.rawMovement))
}

func (m SummonMove) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.cid)
		// oid: present on ALL versions. cid is read upstream by
		// CUserPool::OnUserCommonPacket@0x972401 (Decode4 characterId for the whole
		// 0xAF-0xB4 summon band); CSummonedPool::OnPacket@0x938dd7 then does one more
		// Decode4 = the oid (pool lookup key) before OnMove. Wire = cid + oid + body.
		// The old "no oid pre-95" reading missed the upstream cid read and mislabeled
		// the oid as the cid. See summon-wire-truth.md.
		w.WriteInt(m.oid)
		w.WriteInt16(m.startX)
		w.WriteInt16(m.startY)
		w.WriteByteArray(m.rawMovement)
		return w.Bytes()
	}
}

func (m *SummonMove) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.cid = r.ReadUint32()
		m.oid = r.ReadUint32() // present on all versions (see Encode)
		m.startX = r.ReadInt16()
		m.startY = r.ReadInt16()
		m.rawMovement = r.ReadBytes(r.Available())
	}
}
