package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const UseDoorHandle = "UseDoor"

// UseDoor - CField::TryEnterTownPortal
// Sent when the player enters a town portal. Body: portalFieldId uint32, flag byte.
// packet-audit:fname CField::TryEnterTownPortal#UseDoor
type UseDoor struct {
	portalFieldId uint32
	flag          byte
}

func NewUseDoor(portalFieldId uint32, flag byte) UseDoor {
	return UseDoor{portalFieldId: portalFieldId, flag: flag}
}

func (m UseDoor) PortalFieldId() uint32 { return m.portalFieldId }
func (m UseDoor) Flag() byte            { return m.flag }

func (m UseDoor) Operation() string {
	return UseDoorHandle
}

func (m UseDoor) String() string {
	return fmt.Sprintf("portalFieldId [%d], flag [%d]", m.portalFieldId, m.flag)
}

func (m UseDoor) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.portalFieldId)
		w.WriteByte(m.flag)
		return w.Bytes()
	}
}

func (m *UseDoor) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.portalFieldId = r.ReadUint32()
		m.flag = r.ReadByte()
	}
}
