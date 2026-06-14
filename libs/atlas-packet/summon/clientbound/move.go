package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const SummonMoveWriter = "SummonMove"

// SummonMove is the server -> client MOVE_SUMMON packet: int cid, int oid, then
// the raw CMovePath movement blob rebroadcast byte-faithfully.
//
// The blob ALREADY begins with the start position (CMovePath::Encode writes
// Encode2 startX, Encode2 startY first). The client reads it via
// CMovePath::Decode (v83 @0x68a33c, reached from CSummonedPool::OnMove@0x7a6861).
// The start position must therefore NOT be written separately — doing so makes
// the observer's CMovePath::Decode read 4 bytes off, mis-parse the command count,
// run past the buffer, and throw (ZException / client "error 38"). The owner
// renders its own summon's movement locally and never receives this packet
// (it broadcasts to OTHER sessions only), so the duplication only ever crashed
// other players in the map.
type SummonMove struct {
	cid         uint32
	oid         uint32
	rawMovement []byte
}

func NewSummonMove(cid, oid uint32, rawMovement []byte) SummonMove {
	return SummonMove{cid: cid, oid: oid, rawMovement: rawMovement}
}

func (m SummonMove) Cid() uint32         { return m.cid }
func (m SummonMove) Oid() uint32         { return m.oid }
func (m SummonMove) RawMovement() []byte { return m.rawMovement }
func (m SummonMove) Operation() string   { return SummonMoveWriter }
func (m SummonMove) String() string {
	return fmt.Sprintf("cid [%d], oid [%d], rawMovement [%d bytes]", m.cid, m.oid, len(m.rawMovement))
}

func (m SummonMove) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.cid)
		w.WriteInt(m.oid)               // present on all versions (see SummonSpawn)
		w.WriteByteArray(m.rawMovement) // CMovePath blob — begins with start x,y
		return w.Bytes()
	}
}

func (m *SummonMove) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.cid = r.ReadUint32()
		m.oid = r.ReadUint32()
		m.rawMovement = r.ReadBytes(r.Available())
	}
}
