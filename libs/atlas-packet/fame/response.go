package fame

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const FameResponseWriter = "FameResponse"

type ReceiveResponse struct {
	mode     byte
	fromName string
	amount   int8
}

func NewReceiveFameResponse(mode byte, fromName string, amount int8) ReceiveResponse {
	return ReceiveResponse{mode: mode, fromName: fromName, amount: amount}
}

func (m ReceiveResponse) Operation() string { return FameResponseWriter }
func (m ReceiveResponse) String() string {
	return fmt.Sprintf("receive from [%s], amount [%d]", m.fromName, m.amount)
}

func (m ReceiveResponse) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteAsciiString(m.fromName)
		fameMode := (m.amount + 1) / 2
		w.WriteInt8(fameMode)
		return w.Bytes()
	}
}

func (m *ReceiveResponse) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.fromName = r.ReadAsciiString()
		fameMode := r.ReadInt8()
		m.amount = fameMode*2 - 1
	}
}

type GiveResponse struct {
	mode   byte
	toName string
	amount int8
	total  int16
}

func NewGiveFameResponse(mode byte, toName string, amount int8, total int16) GiveResponse {
	return GiveResponse{mode: mode, toName: toName, amount: amount, total: total}
}

func (m GiveResponse) Operation() string { return FameResponseWriter }
func (m GiveResponse) String() string {
	return fmt.Sprintf("give to [%s], amount [%d], total [%d]", m.toName, m.amount, m.total)
}

func (m GiveResponse) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteAsciiString(m.toName)
		fameMode := (m.amount + 1) / 2
		w.WriteInt8(fameMode)
		w.WriteInt16(m.total)
		w.WriteShort(0)
		return w.Bytes()
	}
}

func (m *GiveResponse) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.toName = r.ReadAsciiString()
		fameMode := r.ReadInt8()
		m.amount = fameMode*2 - 1
		m.total = r.ReadInt16()
		_ = r.ReadUint16() // padding
	}
}

type ErrorResponse struct {
	mode byte
}

func NewFameErrorResponse(mode byte) ErrorResponse {
	return ErrorResponse{mode: mode}
}

func (m ErrorResponse) Operation() string { return FameResponseWriter }
func (m ErrorResponse) String() string    { return fmt.Sprintf("error mode [%d]", m.mode) }

func (m ErrorResponse) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *ErrorResponse) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}
