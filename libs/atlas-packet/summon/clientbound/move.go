package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const SummonMoveWriter = "SummonMove"

// SummonMove is the server -> client MOVE_SUMMON packet: int cid, int oid, then
// the client's CMovePath movement blob rebroadcast to the other sessions in the
// map.
//
// The blob is captured verbatim off the wire and crosses Kafka unchanged, but it
// is NOT written back verbatim: Encode re-serializes it through
// model.ReserializeMovePath so the per-element layout matches what the RECEIVING
// client reads. On GMS v87 the client writes the per-element XOffset/YOffset pair
// and never reads it back (CMovePath::Encode @0x6c70fe vs CMovePath::Decode
// @0x6c6e86), so echoing the capture made every observer's element loop desync —
// summons teleported for everyone but their owner. The move-path trailer the
// codec does not model is carried through verbatim.
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
		w.WriteInt(m.oid) // present on all versions (see SummonSpawn)
		// CMovePath blob — begins with start x,y. Re-serialized for the tenant's
		// OUTBOUND element layout rather than echoed verbatim; see the comment on
		// the type.
		w.WriteByteArray(model.ReserializeMovePath(l, ctx)(m.rawMovement, options))
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
