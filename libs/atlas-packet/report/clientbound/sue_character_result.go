package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const SueCharacterResultWriter = "SueCharacterResult"

// SueCharacterResult - CWvsContext::OnSueCharacterResult. Single result byte
// rendered as a chat-log line (CHATLOG_ADD, not a modal): 0 success,
// 1 unable to locate, 2 daily limit, 3 reported-notice to the accused,
// any other value = generic failure. Byte-identical v61..v95 (only the
// opcode moves: 0x34 on v61/v72/v79, 0x37 on v83+). Version-absent on v48.
// packet-audit:fname CWvsContext::OnSueCharacterResult
type SueCharacterResult struct {
	result byte
}

func NewSueCharacterResult(result byte) SueCharacterResult {
	return SueCharacterResult{result: result}
}

func (m SueCharacterResult) Result() byte { return m.result }

func (m SueCharacterResult) Operation() string { return SueCharacterResultWriter }

func (m SueCharacterResult) String() string {
	return fmt.Sprintf("result [%d]", m.result)
}

func (m SueCharacterResult) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.result)
		return w.Bytes()
	}
}

func (m *SueCharacterResult) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.result = r.ReadByte()
	}
}
