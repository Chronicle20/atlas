package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const TournamentWriter = "Tournament"

// Tournament mirrors CField_Tournament::OnTournament. The client dispatches on a
// leading mode byte (Decode1) into a switch whose branches each read one further
// Decode1; the flat wire shape is three sequential bytes (mode + two operands).
// packet-audit:fname CField_Tournament::OnTournament
type Tournament struct {
	mode byte
	arg0 byte
	arg1 byte
}

func NewTournament(mode byte, arg0 byte, arg1 byte) Tournament {
	return Tournament{mode: mode, arg0: arg0, arg1: arg1}
}

func (m Tournament) Mode() byte { return m.mode }
func (m Tournament) Arg0() byte { return m.arg0 }
func (m Tournament) Arg1() byte { return m.arg1 }

func (m Tournament) Operation() string { return TournamentWriter }
func (m Tournament) String() string {
	return fmt.Sprintf("mode [%d] arg0 [%d] arg1 [%d]", m.mode, m.arg0, m.arg1)
}

func (m Tournament) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.arg0)
		w.WriteByte(m.arg1)
		return w.Bytes()
	}
}

func (m *Tournament) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.arg0 = r.ReadByte()
		m.arg1 = r.ReadByte()
	}
}
