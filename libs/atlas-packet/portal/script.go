package portal

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const PortalScriptHandle = "PortalScriptHandle"

// Script - CField::SendPortalScript
type Script struct {
	fieldKey   byte
	portalName string
	x          int16
	y          int16
}

func (m Script) FieldKey() byte {
	return m.fieldKey
}

func (m Script) PortalName() string {
	return m.portalName
}

func (m Script) X() int16 {
	return m.x
}

func (m Script) Y() int16 {
	return m.y
}

func (m Script) Operation() string {
	return PortalScriptHandle
}

func (m Script) String() string {
	return fmt.Sprintf("fieldKey [%d], portalName [%s], x [%d], y [%d]", m.fieldKey, m.portalName, m.x, m.y)
}

func (m Script) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.fieldKey)
		w.WriteAsciiString(m.portalName)
		w.WriteInt16(m.x)
		w.WriteInt16(m.y)
		return w.Bytes()
	}
}

func (m *Script) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.fieldKey = r.ReadByte()
		m.portalName = r.ReadAsciiString()
		m.x = r.ReadInt16()
		m.y = r.ReadInt16()
	}
}
