package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const RequestFootholdInfoHandle = "RequestFootholdInfo"

// footholdInfoEntrySize is the fixed on-wire size of one RequestFootholdInfo
// entry: nCurState(4) + nCurX(4) + nCurY(4) + reverseVertical(1) +
// reverseHorizontal(1).
const footholdInfoEntrySize = 14

// FootholdInfoEntry is one dynamic-object record the client reports back to the
// server in response to REQUEST_FOOTHOLD_INFO.
//
//   - state: nCurState (Encode4).
//   - x, y : the moving-object position (Encode4 each); 0 when the object has no
//     moving info.
//   - reverseVertical / reverseHorizontal: the trailing flag bytes (Encode1
//     each); 0 when the object has no moving info.
type FootholdInfoEntry struct {
	state             uint32
	x                 uint32
	y                 uint32
	reverseVertical   byte
	reverseHorizontal byte
}

func NewFootholdInfoEntry(state uint32, x uint32, y uint32, reverseVertical byte, reverseHorizontal byte) FootholdInfoEntry {
	return FootholdInfoEntry{
		state:             state,
		x:                 x,
		y:                 y,
		reverseVertical:   reverseVertical,
		reverseHorizontal: reverseHorizontal,
	}
}

func (e FootholdInfoEntry) State() uint32            { return e.state }
func (e FootholdInfoEntry) X() uint32                { return e.x }
func (e FootholdInfoEntry) Y() uint32                { return e.y }
func (e FootholdInfoEntry) ReverseVertical() byte    { return e.reverseVertical }
func (e FootholdInfoEntry) ReverseHorizontal() byte  { return e.reverseHorizontal }

// RequestFootholdInfo models the FOOTHOLD_INFO serverbound packet
// (CField::OnRequestFootHoldInfo). Despite the "On…" name it is the client's
// REPLY to the server's foothold-info request: the client walks its
// m_lDynamicObjs list and builds a COutPacket(270 in v95 / 0xED in jms),
// appending one entry per dynamic object with NO count prefix. Each entry is:
//
//	Encode4(nCurState)
//	Encode4(nCurX), Encode4(nCurY), Encode1(bReverseVertical), Encode1(bReverseHorizontal)
//
// When an object carries no moving info the four position/flag fields are all
// emitted as zero. The server (Atlas) decodes the stream until the buffer is
// exhausted (no length prefix is on the wire).
//
// Version applicability (v95 authoritative): the CField::OnRequestFootHoldInfo
// function exists only in GMS v95 (@0x52ddd0) and jms v185 (@0x576cd2); it is
// VERSION-ABSENT in GMS v83/v84/v87. The wire shape is identical between v95 and
// jms, so the codec is version-invariant.
type RequestFootholdInfo struct {
	entries []FootholdInfoEntry
}

func NewRequestFootholdInfo(entries []FootholdInfoEntry) RequestFootholdInfo {
	return RequestFootholdInfo{entries: entries}
}

func (m RequestFootholdInfo) Entries() []FootholdInfoEntry { return m.entries }

func (m RequestFootholdInfo) Operation() string {
	return RequestFootholdInfoHandle
}

func (m RequestFootholdInfo) String() string {
	return fmt.Sprintf("request foothold info entries [%d]", len(m.entries))
}

func (m RequestFootholdInfo) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		for _, e := range m.entries {
			w.WriteInt(e.state)
			w.WriteInt(e.x)
			w.WriteInt(e.y)
			w.WriteByte(e.reverseVertical)
			w.WriteByte(e.reverseHorizontal)
		}
		return w.Bytes()
	}
}

func (m *RequestFootholdInfo) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.entries = nil
		for r.Available() >= footholdInfoEntrySize {
			m.entries = append(m.entries, FootholdInfoEntry{
				state:             r.ReadUint32(),
				x:                 r.ReadUint32(),
				y:                 r.ReadUint32(),
				reverseVertical:   r.ReadByte(),
				reverseHorizontal: r.ReadByte(),
			})
		}
	}
}
