package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const AriantArenaUserScoreWriter = "AriantArenaUserScore"

// AriantArenaScoreEntry is one entry of CField_AriantArena::UserScore: a
// character name paired with their current arena score.
type AriantArenaScoreEntry struct {
	Name  string
	Score uint32
}

// AriantArenaUserScore mirrors CField_AriantArena::OnUserScore. Read order
// verified against the live IDBs and found identical in every version
// checked:
//
//	gms_v79 @0x528799, gms_v83 @0x53e5e1, gms_v87 @0x567b7d,
//	gms_v95 @0x5492b0 (PDB-backed names), jms_v185 @0x57dc4a
//	(gms_v84 @0x54abaa byte-identical to v83).
//
// Wire layout:
//
//	Decode1 count
//	count x { DecodeStr name; Decode4 score }
//
// The prior atlas codec modelled a single entry (count byte followed by ONE
// name+score pair) instead of a count-length loop; corrected here.
//
// packet-audit:fname CField_AriantArena::OnUserScore
type AriantArenaUserScore struct {
	entries []AriantArenaScoreEntry
}

func NewAriantArenaUserScore(entries []AriantArenaScoreEntry) AriantArenaUserScore {
	return AriantArenaUserScore{entries: entries}
}

// Entries returns a copy so the immutable model can't be mutated through the
// returned slice (entries are value structs, so a shallow copy suffices).
func (m AriantArenaUserScore) Entries() []AriantArenaScoreEntry {
	out := make([]AriantArenaScoreEntry, len(m.entries))
	copy(out, m.entries)
	return out
}

func (m AriantArenaUserScore) Operation() string { return AriantArenaUserScoreWriter }
func (m AriantArenaUserScore) String() string {
	return fmt.Sprintf("entries [%d]", len(m.entries))
}

func (m AriantArenaUserScore) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(byte(len(m.entries)))
		for _, e := range m.entries {
			w.WriteAsciiString(e.Name)
			w.WriteInt(e.Score)
		}
		return w.Bytes()
	}
}

func (m *AriantArenaUserScore) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		count := r.ReadByte()
		m.entries = make([]AriantArenaScoreEntry, count)
		for i := range m.entries {
			m.entries[i].Name = r.ReadAsciiString()
			m.entries[i].Score = r.ReadUint32()
		}
	}
}
