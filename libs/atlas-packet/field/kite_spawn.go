package field

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const KiteSpawnWriter = "KiteSpawn"

type KiteSpawn struct {
	id         uint32
	templateId uint32
	message    string
	name       string
	x          int16
	kiteType   int16
}

func NewKiteSpawn(id uint32, templateId uint32, message string, name string, x int16, kiteType int16) KiteSpawn {
	return KiteSpawn{id: id, templateId: templateId, message: message, name: name, x: x, kiteType: kiteType}
}

func (m KiteSpawn) Operation() string { return KiteSpawnWriter }
func (m KiteSpawn) String() string {
	return fmt.Sprintf("id [%d], templateId [%d]", m.id, m.templateId)
}

func (m KiteSpawn) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.id)
		w.WriteInt(m.templateId)
		w.WriteAsciiString(m.message)
		w.WriteAsciiString(m.name)
		w.WriteInt16(m.x)
		w.WriteInt16(m.kiteType)
		return w.Bytes()
	}
}

func (m *KiteSpawn) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.id = r.ReadUint32()
		m.templateId = r.ReadUint32()
		m.message = r.ReadAsciiString()
		m.name = r.ReadAsciiString()
		m.x = r.ReadInt16()
		m.kiteType = r.ReadInt16()
	}
}
