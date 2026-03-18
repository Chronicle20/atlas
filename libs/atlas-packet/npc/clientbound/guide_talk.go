package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const GuideTalkWriter = "GuideTalk"

type GuideTalkMessage struct {
	message  string
	width    uint32
	duration uint32
}

func NewGuideTalkMessage(message string, width uint32, duration uint32) GuideTalkMessage {
	return GuideTalkMessage{message: message, width: width, duration: duration}
}

func (m GuideTalkMessage) Operation() string { return GuideTalkWriter }
func (m GuideTalkMessage) String() string {
	return fmt.Sprintf("message [%s], width [%d], duration [%d]", m.message, m.width, m.duration)
}

func (m GuideTalkMessage) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteBool(true)
		w.WriteAsciiString(m.message)
		w.WriteInt(m.width)
		w.WriteInt(m.duration)
		return w.Bytes()
	}
}

func (m *GuideTalkMessage) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		_ = r.ReadBool() // always true for message mode
		m.message = r.ReadAsciiString()
		m.width = r.ReadUint32()
		m.duration = r.ReadUint32()
	}
}

type GuideTalkIdx struct {
	hintId   uint32
	duration uint32
}

func NewGuideTalkIdx(hintId uint32, duration uint32) GuideTalkIdx {
	return GuideTalkIdx{hintId: hintId, duration: duration}
}

func (m GuideTalkIdx) Operation() string { return GuideTalkWriter }
func (m GuideTalkIdx) String() string {
	return fmt.Sprintf("hintId [%d], duration [%d]", m.hintId, m.duration)
}

func (m GuideTalkIdx) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteBool(false)
		w.WriteInt(m.hintId)
		w.WriteInt(m.duration)
		return w.Bytes()
	}
}

func (m *GuideTalkIdx) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		_ = r.ReadBool() // always false for idx mode
		m.hintId = r.ReadUint32()
		m.duration = r.ReadUint32()
	}
}
