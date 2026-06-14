package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const TournamentUewWriter = "TournamentUew"

// TournamentUew mirrors CField_Tournament::OnTournamentUEW. The client reads a
// single byte (Decode1).
type TournamentUew struct {
	effect byte
}

func NewTournamentUew(effect byte) TournamentUew {
	return TournamentUew{effect: effect}
}

func (m TournamentUew) Effect() byte { return m.effect }

func (m TournamentUew) Operation() string { return TournamentUewWriter }
func (m TournamentUew) String() string {
	return fmt.Sprintf("effect [%d]", m.effect)
}

func (m TournamentUew) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.effect)
		return w.Bytes()
	}
}

func (m *TournamentUew) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.effect = r.ReadByte()
	}
}
