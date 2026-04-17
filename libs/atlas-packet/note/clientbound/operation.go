package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const NoteOperationWriter = "NoteOperation"

// SendSuccess - just mode byte
type SendSuccess struct {
	mode byte
}

func NewNoteSendSuccess(mode byte) SendSuccess {
	return SendSuccess{mode: mode}
}

func (m SendSuccess) Mode() byte { return m.mode }

func (m SendSuccess) Operation() string { return NoteOperationWriter }
func (m SendSuccess) String() string    { return "note send success" }

func (m SendSuccess) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *SendSuccess) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// SendError - mode, errorCode
type SendError struct {
	mode      byte
	errorCode byte
}

func NewNoteSendError(mode byte, errorCode byte) SendError {
	return SendError{mode: mode, errorCode: errorCode}
}

func (m SendError) Mode() byte      { return m.mode }
func (m SendError) ErrorCode() byte { return m.errorCode }

func (m SendError) Operation() string { return NoteOperationWriter }
func (m SendError) String() string {
	return fmt.Sprintf("note send error [%d]", m.errorCode)
}

func (m SendError) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.errorCode)
		return w.Bytes()
	}
}

func (m *SendError) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.errorCode = r.ReadByte()
	}
}

// Refresh - just mode byte
type Refresh struct {
	mode byte
}

func NewNoteRefresh(mode byte) Refresh {
	return Refresh{mode: mode}
}

func (m Refresh) Mode() byte { return m.mode }

func (m Refresh) Operation() string { return NoteOperationWriter }
func (m Refresh) String() string    { return "note refresh" }

func (m Refresh) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *Refresh) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}
