package clientbound

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const TournamentCharactersWriter = "TournamentCharacters"

// TournamentCharacters mirrors CField_Tournament::OnPacket (op
// TOURNAMENT_CHARACTERS). The handler is a no-op stub with an empty body in the
// audited versions; the packet carries no payload beyond its opcode.
// packet-audit:fname CField_Tournament::OnPacket
type TournamentCharacters struct{}

func NewTournamentCharacters() TournamentCharacters {
	return TournamentCharacters{}
}

func (m TournamentCharacters) Operation() string { return TournamentCharactersWriter }
func (m TournamentCharacters) String() string {
	return "TournamentCharacters"
}

func (m TournamentCharacters) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		return w.Bytes()
	}
}

func (m *TournamentCharacters) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
	}
}
