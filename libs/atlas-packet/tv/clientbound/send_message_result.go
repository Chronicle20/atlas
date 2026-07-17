package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const TvSendMessageResultWriter = "TvSendMessageResult"

// TvSendMessageResult reports the outcome of a TV message submission:
// success (00) or an error carrying a config-resolved code (01 <code>).
// packet-audit:fname CMapleTVMan::OnSendMessageResult
type TvSendMessageResult struct {
	hasError bool
	code     byte
}

// NewTvSendMessageResultSuccess builds the success wire: a single 00 byte.
func NewTvSendMessageResultSuccess() TvSendMessageResult {
	return TvSendMessageResult{hasError: false}
}

// NewTvSendMessageResultError builds the error wire: 01 <code>. IDA-verified
// (gms_v83 CMapleTVMan::OnSendMessageResult@0x6373a0): code 1 = GM_MESSAGE
// ("non-GM character tried to send GM message"), code 2 = WRONG_USER
// ("you've entered the wrong user name"), code 3 = QUEUE_TOO_LONG ("waiting
// line is longer than an hour"). The concrete byte a domain service selects
// is resolved from the tenant errorCodes table by TvSendMessageResultErrorBody,
// never passed as a literal by callers outside this codec.
func NewTvSendMessageResultError(code byte) TvSendMessageResult {
	return TvSendMessageResult{hasError: true, code: code}
}

func (m TvSendMessageResult) HasError() bool    { return m.hasError }
func (m TvSendMessageResult) Code() byte        { return m.code }
func (m TvSendMessageResult) Operation() string { return TvSendMessageResultWriter }
func (m TvSendMessageResult) String() string {
	return fmt.Sprintf("tv send message result hasError [%v] code [%d]", m.hasError, m.code)
}

func (m TvSendMessageResult) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteBool(m.hasError)
		if m.hasError {
			w.WriteByte(m.code)
		}
		return w.Bytes()
	}
}

func (m *TvSendMessageResult) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.hasError = r.ReadBool()
		if m.hasError {
			m.code = r.ReadByte()
		}
	}
}
