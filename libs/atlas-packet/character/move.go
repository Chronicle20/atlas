package character

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-packet/model"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterMoveHandle = "CharacterMoveHandle"

type Move struct {
	dr0      uint32
	dr1      uint32
	fieldKey byte
	dr2      uint32
	dr3      uint32
	crc      uint32
	dwKey    uint32
	crc32    uint32
	movement model.Movement
}

func (m Move) Dr0() uint32                    { return m.dr0 }
func (m Move) Dr1() uint32                    { return m.dr1 }
func (m Move) FieldKey() byte                 { return m.fieldKey }
func (m Move) Dr2() uint32                    { return m.dr2 }
func (m Move) Dr3() uint32                    { return m.dr3 }
func (m Move) Crc() uint32                    { return m.crc }
func (m Move) DwKey() uint32                  { return m.dwKey }
func (m Move) Crc32() uint32                  { return m.crc32 }
func (m Move) MovementData() model.Movement   { return m.movement }

func (m Move) Operation() string {
	return CharacterMoveHandle
}

func (m Move) String() string {
	return fmt.Sprintf("dr0 [%d] dr1 [%d] fieldKey [%d] dr2 [%d] dr3 [%d] crc [%d] dwKey [%d] crc32 [%d] elements [%d]",
		m.dr0, m.dr1, m.fieldKey, m.dr2, m.dr3, m.crc, m.dwKey, m.crc32, len(m.movement.Elements))
}

func (m Move) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		if (t.Region() == "GMS" && t.MajorVersion() > 83) || t.Region() == "JMS" {
			w.WriteInt(m.dr0)
			w.WriteInt(m.dr1)
		}
		w.WriteByte(m.fieldKey)
		if (t.Region() == "GMS" && t.MajorVersion() > 83) || t.Region() == "JMS" {
			w.WriteInt(m.dr2)
			w.WriteInt(m.dr3)
		}
		if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
			w.WriteInt(m.crc)
		}
		if (t.Region() == "GMS" && t.MajorVersion() > 83) || t.Region() == "JMS" {
			w.WriteInt(m.dwKey)
			w.WriteInt(m.crc32)
		}
		w.WriteByteArray(m.movement.Encode(l, ctx)(options))
		return w.Bytes()
	}
}

func (m *Move) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		if (t.Region() == "GMS" && t.MajorVersion() > 83) || t.Region() == "JMS" {
			m.dr0 = r.ReadUint32()
			m.dr1 = r.ReadUint32()
		}
		m.fieldKey = r.ReadByte()
		if (t.Region() == "GMS" && t.MajorVersion() > 83) || t.Region() == "JMS" {
			m.dr2 = r.ReadUint32()
			m.dr3 = r.ReadUint32()
		}
		if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
			m.crc = r.ReadUint32()
		}
		if (t.Region() == "GMS" && t.MajorVersion() > 83) || t.Region() == "JMS" {
			m.dwKey = r.ReadUint32()
			m.crc32 = r.ReadUint32()
		}
		m.movement.Decode(l, ctx)(r, options)
	}
}
