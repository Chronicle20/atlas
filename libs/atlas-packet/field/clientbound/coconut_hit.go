package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CoconutHitWriter = "CoconutHit"

type CoconutHit struct {
	id     uint16
	action uint16
	hits   byte
}

func NewCoconutHit(id uint16, action uint16, hits byte) CoconutHit {
	return CoconutHit{id: id, action: action, hits: hits}
}

func (m CoconutHit) Id() uint16     { return m.id }
func (m CoconutHit) Action() uint16 { return m.action }
func (m CoconutHit) Hits() byte     { return m.hits }

func (m CoconutHit) Operation() string { return CoconutHitWriter }
func (m CoconutHit) String() string {
	return fmt.Sprintf("id [%d] action [%d] hits [%d]", m.id, m.action, m.hits)
}

func (m CoconutHit) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteShort(m.id)
		w.WriteShort(m.action)
		w.WriteByte(m.hits)
		return w.Bytes()
	}
}

func (m *CoconutHit) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.id = r.ReadUint16()
		m.action = r.ReadUint16()
		m.hits = r.ReadByte()
	}
}
