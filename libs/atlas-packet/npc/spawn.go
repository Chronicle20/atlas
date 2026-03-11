package npc

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const NpcSpawnWriter = "SpawnNPC"

type SpawnW struct {
	id       uint32
	template uint32
	x        int16
	cy       int16
	f        int32
	fh       uint16
	rx0      int16
	rx1      int16
}

func NewNpcSpawn(id uint32, template uint32, x int16, cy int16, f int32, fh uint16, rx0 int16, rx1 int16) SpawnW {
	return SpawnW{id: id, template: template, x: x, cy: cy, f: f, fh: fh, rx0: rx0, rx1: rx1}
}

func (m SpawnW) Operation() string { return NpcSpawnWriter }
func (m SpawnW) String() string {
	return fmt.Sprintf("id [%d], template [%d]", m.id, m.template)
}

func (m SpawnW) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.id)
		w.WriteInt(m.template)
		w.WriteInt16(m.x)
		w.WriteInt16(m.cy)
		if m.f == 1 {
			w.WriteByte(0)
		} else {
			w.WriteByte(1)
		}
		w.WriteShort(m.fh)
		w.WriteInt16(m.rx0)
		w.WriteInt16(m.rx1)
		w.WriteByte(1)
		return w.Bytes()
	}
}

func (m *SpawnW) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.id = r.ReadUint32()
		m.template = r.ReadUint32()
		m.x = r.ReadInt16()
		m.cy = r.ReadInt16()
		fByte := r.ReadByte()
		if fByte == 0 {
			m.f = 1
		} else {
			m.f = 0
		}
		m.fh = r.ReadUint16()
		m.rx0 = r.ReadInt16()
		m.rx1 = r.ReadInt16()
		_ = r.ReadByte() // always 1
	}
}
