package serverbound

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const SummonMoveHandle = "SummonMoveHandle"

// Move is the client -> server MOVE_SUMMON packet, decoded from the real client
// SEND site CVecCtrlSummoned::EndUpdateActive (v83 sub_9C84E9, v87 @0xa591da,
// v95 @0x9a0700). The send is:
//
//	COutPacket(op)
//	Encode4 summonId            ; the leading summon identity
//	CMovePath::Flush(...)        ; the opaque movement blob
//
// The CMovePath::Flush blob (CMovePath::Encode, v83 @0x68a563) begins with
// Encode2 startX, Encode2 startY, Encode1 count, then count move commands of
// variable width, a keypad-input run, and a trailing bounding box. It is NOT
// trivially parseable without a full move-path codec, so the server treats the
// entire post-identity remainder as an opaque blob and rebroadcasts it
// byte-faithfully. startX/startY are extracted from the first 4 bytes only to
// seed the persisted position.
//
// Summon identity field (the int after the opcode):
//   - v83 / v87 (GMS < 95): owner charId (cid). The v83/v87 client has no oid
//     concept (the summon pool is cid-keyed); the controller stores the owner
//     cid (v83 ctrl[0x248], v87 ctrl[188]) which propagates the CSummoned cid
//     at [obj+0xAC]). The server resolves the summon by owner channel-side.
//   - v95+ (GMS >= 95): the server-allocated m_dwSummonedID (a real summon id).
//
// Either way the decoded value is exposed via SummonId(); the channel handler
// reconciles cid-vs-id against the sender's owned summons.
//
// packet-audit:fname CSummonedPool::OnMove
type Move struct {
	summonId    uint32
	startX      int16
	startY      int16
	rawMovement []byte
}

func (m Move) SummonId() uint32    { return m.summonId }
func (m Move) StartX() int16       { return m.startX }
func (m Move) StartY() int16       { return m.startY }
func (m Move) RawMovement() []byte { return m.rawMovement }

func (m Move) Operation() string { return SummonMoveHandle }

func (m Move) String() string {
	return fmt.Sprintf("summonId [%d], startX [%d], startY [%d], rawMovement [%d bytes]", m.summonId, m.startX, m.startY, len(m.rawMovement))
}

func (m Move) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	_ = tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.summonId)
		w.WriteByteArray(m.rawMovement)
		return w.Bytes()
	}
}

func (m *Move) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	_ = tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.summonId = r.ReadUint32()
		m.rawMovement = r.ReadBytes(r.Available())
		// startX/startY are the first 4 bytes of the move blob (CMovePath::Encode:
		// Encode2 startX, Encode2 startY ...). Extract them for position seeding;
		// the full blob is still rebroadcast byte-faithfully via rawMovement.
		if len(m.rawMovement) >= 4 {
			m.startX = int16(binary.LittleEndian.Uint16(m.rawMovement[0:2]))
			m.startY = int16(binary.LittleEndian.Uint16(m.rawMovement[2:4]))
		}
	}
}
