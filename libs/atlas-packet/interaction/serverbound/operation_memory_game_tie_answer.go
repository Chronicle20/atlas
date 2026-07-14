package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// packet-audit:fname CMemoryGameDlg::OnTieRequest
//
// VERSION QUIRK (gms_v79): the tie pair's bodies are inverted vs v83 — on v79
// ASK_TIE (mode 50) carries the response bool and TIE_ANSWER (mode 51) carries
// none (the reverse of v83; IDA-verified, see interaction-legacy-audit.md).
// This codec (reading the bool under TIE_ANSWER) is therefore byte-wrong for
// v79. It is currently harmless: Atlas does not implement mini-games (the
// ASK_TIE/TIE_ANSWER handlers only log), and the reader is bounds-safe, so the
// v79 over-read yields a discarded `false`. If mini-games are ever implemented
// on v79, version-gate this pair (swap which mode reads the bool).
type OperationMemoryGameTieAnswer struct {
	response bool
}

func (m OperationMemoryGameTieAnswer) Response() bool { return m.response }

func (m OperationMemoryGameTieAnswer) Operation() string { return "OperationMemoryGameTieAnswer" }

func (m OperationMemoryGameTieAnswer) String() string {
	return fmt.Sprintf("response [%v]", m.response)
}

func (m OperationMemoryGameTieAnswer) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteBool(m.response)
		return w.Bytes()
	}
}

func (m *OperationMemoryGameTieAnswer) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.response = r.ReadBool()
	}
}
