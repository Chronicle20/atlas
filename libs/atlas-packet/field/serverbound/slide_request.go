package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const SlideRequestHandle = "SlideRequest"

// SlideRequest - CField::SendChatMsgSlash#SlideRequest (v95 0x09E, jms 0x089).
// Sent by the /-command parser for the slide request (v95+jms only; absent in
// v83/v84/v87). Body: a single byte.
// packet-audit:fname CField::SendChatMsgSlash#SlideRequest
type SlideRequest struct {
	value byte
}

func NewSlideRequest(value byte) SlideRequest {
	return SlideRequest{value: value}
}

func (m SlideRequest) Value() byte { return m.value }

func (m SlideRequest) Operation() string {
	return SlideRequestHandle
}

func (m SlideRequest) String() string {
	return fmt.Sprintf("value [%d]", m.value)
}

func (m SlideRequest) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.value)
		return w.Bytes()
	}
}

func (m *SlideRequest) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.value = r.ReadByte()
	}
}
