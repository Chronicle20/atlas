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

const CharacterListWriter = "CharacterList"

type CharacterList struct {
	status         byte
	characters     []model.CharacterListEntry
	hasPic         bool
	characterSlots uint32
}

func NewCharacterList(status byte, characters []model.CharacterListEntry, hasPic bool, characterSlots uint32) CharacterList {
	return CharacterList{
		status:         status,
		characters:     characters,
		hasPic:         hasPic,
		characterSlots: characterSlots,
	}
}

func (m CharacterList) Status() byte                              { return m.status }
func (m CharacterList) Characters() []model.CharacterListEntry    { return m.characters }
func (m CharacterList) HasPic() bool                              { return m.hasPic }
func (m CharacterList) CharacterSlots() uint32                    { return m.characterSlots }
func (m CharacterList) Operation() string                         { return CharacterListWriter }
func (m CharacterList) String() string {
	return fmt.Sprintf("status [%d], characters [%d]", m.status, len(m.characters))
}

func (m CharacterList) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.status)

		if t.Region() == "JMS" {
			w.WriteAsciiString("")
		}

		w.WriteByte(byte(len(m.characters)))
		for _, c := range m.characters {
			c.Write(l, ctx, w, options, false)
		}

		if t.Region() == "GMS" && t.MajorVersion() <= 28 {
			return w.Bytes()
		}

		w.WriteBool(m.hasPic)
		if t.Region() == "GMS" {
			w.WriteInt(m.characterSlots)
			if t.MajorVersion() > 87 {
				w.WriteInt(0) // nBuyCharCount
			}
		} else if t.Region() == "JMS" {
			w.WriteByte(0)
			w.WriteInt(m.characterSlots)
			w.WriteInt(0)
		}

		return w.Bytes()
	}
}

func (m *CharacterList) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.status = r.ReadByte()

		if t.Region() == "JMS" {
			_ = r.ReadAsciiString()
		}

		count := r.ReadByte()
		m.characters = make([]model.CharacterListEntry, count)
		for i := byte(0); i < count; i++ {
			m.characters[i].Read(l, ctx, r, options, false)
		}

		if t.Region() == "GMS" && t.MajorVersion() <= 28 {
			return
		}

		m.hasPic = r.ReadBool()
		if t.Region() == "GMS" {
			m.characterSlots = r.ReadUint32()
			if t.MajorVersion() > 87 {
				_ = r.ReadUint32() // nBuyCharCount
			}
		} else if t.Region() == "JMS" {
			_ = r.ReadByte()
			m.characterSlots = r.ReadUint32()
			_ = r.ReadUint32()
		}
	}
}
