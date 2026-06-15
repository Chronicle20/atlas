package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const MatchTableHandle = "MatchTable"

// MatchTable - CField::SendChatMsgSlash#MatchTable (opcode varies per version).
// Sent by the /-command parser for the match-table request. Body: a single byte
// (a bool flag).
type MatchTable struct {
	flag byte
}

func NewMatchTable(flag byte) MatchTable {
	return MatchTable{flag: flag}
}

func (m MatchTable) Flag() byte { return m.flag }

func (m MatchTable) Operation() string {
	return MatchTableHandle
}

func (m MatchTable) String() string {
	return fmt.Sprintf("flag [%d]", m.flag)
}

func (m MatchTable) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.flag)
		return w.Bytes()
	}
}

func (m *MatchTable) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.flag = r.ReadByte()
	}
}
