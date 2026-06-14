package clientbound

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const TournamentMatchTableWriter = "TournamentMatchTable"

// TournamentMatchTable mirrors CField_Tournament::OnTournamentMatchTable. The
// client handler has an empty body in every audited version (no Decode calls);
// the packet carries no payload beyond its opcode.
type TournamentMatchTable struct {
}

func NewTournamentMatchTable() TournamentMatchTable {
	return TournamentMatchTable{}
}

func (m TournamentMatchTable) Operation() string { return TournamentMatchTableWriter }
func (m TournamentMatchTable) String() string {
	return "TournamentMatchTable"
}

func (m TournamentMatchTable) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		return w.Bytes()
	}
}

func (m *TournamentMatchTable) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
	}
}
