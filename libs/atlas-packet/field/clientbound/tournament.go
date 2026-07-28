package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const TournamentWriter = "Tournament"

// Tournament mirrors CField_Tournament::OnTournament. Read order verified
// against the live IDBs and found identical in every version checked:
//
//	gms_v79 @0x5585af, gms_v83 @0x57b61a, gms_v87 @0x5a9d67,
//	gms_v95 @0x5631a0 (PDB-backed names), jms_v185 @0x5cfdac
//	(gms_v84 byte-identical to v83).
//
// Wire layout: the leading `if (Decode1() || (secType&1)==0)` reads the
// FIRST byte as part of the branch condition itself (C `||` short-circuit:
// the second operand — a purely client-local TSecType check, never a wire
// read — is only evaluated when the first Decode1() is falsy). Whichever
// arm is taken then reads exactly one further Decode1() unconditionally:
// the "if" arm always decodes a rank/place value (0/1/2/other, formatted
// into a champion/finalist/round-N StringPool notice), and the "else" arm
// always decodes a start-state value (0/1/other, formatted into a
// prize-not-set/not-enough-users notice). Both arms terminate after that
// second byte — no further CInPacket reads on any path. The true wire is
// therefore a FLAT, unconditional two bytes (mode + value); there is no
// third byte and no gating needed in the codec itself.
//
// The prior atlas codec wrote/read an unconditional THIRD byte, permanently
// desyncing the client on every OnTournament packet (the excess byte is
// consumed as the start of the next packet header) — a false pass (the
// verify markers asserted the encoder's own three-field output, never the
// true two-byte wire body). Corrected here across all versions (task-181).
//
// packet-audit:fname CField_Tournament::OnTournament
type Tournament struct {
	mode  byte
	value byte
}

func NewTournament(mode byte, value byte) Tournament {
	return Tournament{mode: mode, value: value}
}

func (m Tournament) Mode() byte  { return m.mode }
func (m Tournament) Value() byte { return m.value }

func (m Tournament) Operation() string { return TournamentWriter }
func (m Tournament) String() string {
	return fmt.Sprintf("mode [%d] value [%d]", m.mode, m.value)
}

func (m Tournament) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.value)
		return w.Bytes()
	}
}

func (m *Tournament) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.value = r.ReadByte()
	}
}
