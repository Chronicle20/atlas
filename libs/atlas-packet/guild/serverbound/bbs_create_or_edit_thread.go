package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type BBSCreateOrEditThread struct {
	modify     bool
	threadId   uint32
	notice     bool
	title      string
	message    string
	emoticonId uint32
}

func (m BBSCreateOrEditThread) Modify() bool {
	return m.modify
}

func (m BBSCreateOrEditThread) ThreadId() uint32 {
	return m.threadId
}

func (m BBSCreateOrEditThread) Notice() bool {
	return m.notice
}

func (m BBSCreateOrEditThread) Title() string {
	return m.title
}

func (m BBSCreateOrEditThread) Message() string {
	return m.message
}

func (m BBSCreateOrEditThread) EmoticonId() uint32 {
	return m.emoticonId
}

func (m BBSCreateOrEditThread) Operation() string {
	return "BBSCreateOrEditThread"
}

func (m BBSCreateOrEditThread) String() string {
	return fmt.Sprintf("modify [%t] threadId [%d] notice [%t] title [%s] message [%s] emoticonId [%d]", m.modify, m.threadId, m.notice, m.title, m.message, m.emoticonId)
}

func (m BBSCreateOrEditThread) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteBool(m.modify)
		if m.modify {
			w.WriteInt(m.threadId)
		}
		w.WriteBool(m.notice)
		w.WriteAsciiString(m.title)
		w.WriteAsciiString(m.message)
		w.WriteInt(m.emoticonId)
		return w.Bytes()
	}
}

func (m *BBSCreateOrEditThread) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.modify = r.ReadBool()
		if m.modify {
			m.threadId = r.ReadUint32()
		}
		m.notice = r.ReadBool()
		m.title = r.ReadAsciiString()
		m.message = r.ReadAsciiString()
		m.emoticonId = r.ReadUint32()
	}
}
