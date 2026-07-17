package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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

func (m CharacterList) Status() byte                           { return m.status }
func (m CharacterList) Characters() []model.CharacterListEntry { return m.characters }
func (m CharacterList) HasPic() bool                           { return m.hasPic }
func (m CharacterList) CharacterSlots() uint32                 { return m.characterSlots }
func (m CharacterList) Operation() string                      { return CharacterListWriter }
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
			w.WriteByteArray(c.Encode(l, ctx)(options))
		}

		if t.Region() == "GMS" && t.MajorVersion() <= 28 {
			return w.Bytes()
		}

		// hasPic / m_bLoginOpt byte is absent in legacy GMS (<v83). The v79 client
		// char-list decoder (sub_5CE522 @0x5CE522) reads the slot count (Decode4)
		// directly after the entry loop with no login-option byte /*0x5ce7ac*/.
		// JMS and GMS>=83 read it (list_test.go v83 fixture, hasPic @0x5f9b34).
		if !(t.Region() == "GMS" && t.MajorVersion() < 83) {
			w.WriteBool(m.hasPic)
		}
		// The trailing slot-count int (m_nSlotCount) entered the char-list at GMS v61:
		// v61 char-list decoder (sub_56688D @0x566b02) reads Decode4 after the entry
		// loop; v48 (sub_5013ED @0x501626) ends the loop and returns with NO trailing
		// Decode4. Legacy GMS < 61 omits the slot count entirely.
		if t.Region() == "GMS" {
			if t.MajorVersion() >= 61 {
				w.WriteInt(m.characterSlots)
				if t.MajorVersion() > 87 {
					w.WriteInt(0) // nBuyCharCount
				}
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
			m.characters[i].Decode(l, ctx)(r, options)
		}

		if t.Region() == "GMS" && t.MajorVersion() <= 28 {
			return
		}

		if !(t.Region() == "GMS" && t.MajorVersion() < 83) {
			m.hasPic = r.ReadBool()
		}
		// Mirror of Encode: legacy GMS < 61 omits the trailing slot-count int.
		if t.Region() == "GMS" {
			if t.MajorVersion() >= 61 {
				m.characterSlots = r.ReadUint32()
				if t.MajorVersion() > 87 {
					_ = r.ReadUint32() // nBuyCharCount
				}
			}
		} else if t.Region() == "JMS" {
			_ = r.ReadByte()
			m.characterSlots = r.ReadUint32()
			_ = r.ReadUint32()
		}
	}
}
