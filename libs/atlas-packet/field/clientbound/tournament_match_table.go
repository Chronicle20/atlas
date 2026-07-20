package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const TournamentMatchTableWriter = "TournamentMatchTable"

// TournamentMatchTableBufferSize is the raw byte size of the match-table
// blob (CInPacket::DecodeBuffer(..., 0x300)).
const TournamentMatchTableBufferSize = 0x300

// TournamentMatchTable mirrors CField_Tournament::OnTournamentMatchTable.
// The handler itself only allocates + constructs a CMatchTableDlg (v87/v95
// name; v79/v83/jms keep it anonymous as sub_750E40/sub_7DE42C/sub_864212)
// and calls CDialog::DoModal on it — ALL wire reads happen inside that
// dialog's constructor, which every audited version performs identically:
//
//	gms_v79 @0x55871f (ctor helper sub_750E40 @0x750e40),
//	gms_v83 @0x57b78a (ctor helper sub_7DE42C @0x7de42c),
//	gms_v87 @0x5a9ed7 (CMatchTableDlg::CMatchTableDlg @0x83517f),
//	gms_v95 @0x5630d0 (CMatchTableDlg::CMatchTableDlg @0x780210, PDB-backed
//	names), jms_v185 @0x5cff1c (ctor helper sub_864212 @0x864212)
//	(gms_v84 @0x58b29b byte-identical to v83).
//
// Wire layout (identical in every version):
//
//	DecodeBuffer(0x300)  -> m_aaMatch  (raw 768-byte blob; PDB types it
//	                         `unsigned int[32][6]` — 32 rows x 6 uint32
//	                         columns — but CInPacket::DecodeBuffer is a
//	                         single bulk memcpy, not per-field Decode4
//	                         calls, so the wire itself carries one opaque
//	                         768-byte buffer, not 192 individually-typed
//	                         reads)
//	Decode1              -> m_nState  (one trailing byte)
//
// No count prefix, no conditional gating — both fields are always present
// and fixed-size. The prior atlas codec's Encode was an EMPTY stub (wrote
// zero bytes), a false pass: the client always expects this 769-byte body.
// Corrected here across all versions (task-181).
//
// packet-audit:fname CField_Tournament::OnTournamentMatchTable
type TournamentMatchTable struct {
	match [TournamentMatchTableBufferSize]byte
	state byte
}

func NewTournamentMatchTable(match [TournamentMatchTableBufferSize]byte, state byte) TournamentMatchTable {
	return TournamentMatchTable{match: match, state: state}
}

func (m TournamentMatchTable) Match() [TournamentMatchTableBufferSize]byte { return m.match }
func (m TournamentMatchTable) State() byte                                 { return m.state }

func (m TournamentMatchTable) Operation() string { return TournamentMatchTableWriter }
func (m TournamentMatchTable) String() string {
	return fmt.Sprintf("match [%d bytes] state [%d]", len(m.match), m.state)
}

func (m TournamentMatchTable) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByteArray(m.match[:])
		w.WriteByte(m.state)
		return w.Bytes()
	}
}

func (m *TournamentMatchTable) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		copy(m.match[:], r.ReadBytes(TournamentMatchTableBufferSize))
		m.state = r.ReadByte()
	}
}
