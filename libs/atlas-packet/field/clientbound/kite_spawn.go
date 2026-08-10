package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const KiteSpawnWriter = "SpawnKite"

// packet-audit:fname CMessageBoxPool::OnMessageBoxEnterField
//
// The sixth field is the spawn Y coordinate, NOT a kite-type discriminator.
// CMessageBoxPool::OnMessageBoxEnterField (gms_v95 @0x6369c0, gms_v83
// @0x65acdf) decodes it into MESSAGEBOX+32, then computes
// renderX = (+28) - 3 and renderY = (+32) - 100 and feeds BOTH to a single
// IWzVector2D::RelMove. The -3/-100 are sprite-anchor offsets, not flags.
// The banner's appearance is selected by templateId alone, which is the sole
// argument to CItemInfo::GetItemProp further down the same function. There is
// no kite-type field on the wire.
type KiteSpawn struct {
	id         uint32
	templateId uint32
	message    string
	name       string
	x          int16
	y          int16
}

func NewKiteSpawn(id uint32, templateId uint32, message string, name string, x int16, y int16) KiteSpawn {
	return KiteSpawn{id: id, templateId: templateId, message: message, name: name, x: x, y: y}
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
		w.WriteInt16(m.y)
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
		m.y = r.ReadInt16()
	}
}
