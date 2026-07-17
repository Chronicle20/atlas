package model

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type Avatar struct {
	gender          byte
	skinColor       byte
	face            uint32
	mega            bool
	hair            uint32
	equipment       map[slot.Position]uint32
	maskedEquipment map[slot.Position]uint32
	pets            map[int8]uint32
}

func NewAvatar(gender byte, skinColor byte, face uint32, mega bool, hair uint32, equipment map[slot.Position]uint32, maskedEquipment map[slot.Position]uint32, pets map[int8]uint32) Avatar {
	return Avatar{
		gender:          gender,
		skinColor:       skinColor,
		face:            face,
		mega:            mega,
		hair:            hair,
		equipment:       equipment,
		maskedEquipment: maskedEquipment,
		pets:            pets,
	}
}

func (m Avatar) Gender() byte                              { return m.gender }
func (m Avatar) SkinColor() byte                           { return m.skinColor }
func (m Avatar) Face() uint32                              { return m.face }
func (m Avatar) Mega() bool                                { return m.mega }
func (m Avatar) Hair() uint32                              { return m.hair }
func (m Avatar) Equipment() map[slot.Position]uint32       { return m.equipment }
func (m Avatar) MaskedEquipment() map[slot.Position]uint32 { return m.maskedEquipment }
func (m Avatar) Pets() map[int8]uint32                     { return m.pets }

func (m Avatar) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		if t.Region() == "GMS" && t.MajorVersion() <= 28 {
			// older versions don't write gender / skin color / face / mega / hair a second time
		} else {
			w.WriteByte(m.gender)
			w.WriteByte(m.skinColor)
			w.WriteInt(m.face)
			w.WriteBool(!m.mega)
			w.WriteInt(m.hair)
		}
		for k, v := range m.equipment {
			w.WriteKeyValue(byte(k), v)
		}
		if t.Region() == "GMS" && t.MajorVersion() <= 28 {
			w.WriteByte(0)
		} else {
			w.WriteByte(0xFF)
		}
		for k, v := range m.maskedEquipment {
			w.WriteKeyValue(byte(k), v)
		}
		if t.Region() == "GMS" && t.MajorVersion() <= 28 {
			w.WriteByte(0)
		} else {
			w.WriteByte(0xFF)
		}

		w.WriteInt(0)

		// Pet template ids in AvatarLook: 3 ints (DecodeBuffer(12)) is the v61+ shape
		// (v61 AvatarLook::Decode @0x4b77b1). Legacy GMS v29..v60 (v48, sub_49E1E0
		// @0x49e2b9) reads a SINGLE 4-byte pet int; v28 and below read one 8-byte long.
		if (t.Region() == "GMS" && t.MajorVersion() >= 61) || t.Region() == "JMS" {
			for i := int8(0); i < 3; i++ {
				if m.pets == nil {
					w.WriteInt(0)
					continue
				}
				if _, ok := m.pets[i]; ok {
					w.WriteInt(m.pets[i])
				} else {
					w.WriteInt(0)
				}
			}
		} else if t.Region() == "GMS" && t.MajorVersion() > 28 {
			// v48 legacy single pet: one 4-byte int (IDA sub_49E1E0 @0x49e2b9).
			if len(m.pets) > 0 {
				w.WriteInt(m.pets[0])
			} else {
				w.WriteInt(0)
			}
		} else {
			if len(m.pets) > 0 {
				w.WriteLong(uint64(m.pets[0]))
			} else {
				w.WriteLong(0)
			}
		}
		return w.Bytes()
	}
}

func (m *Avatar) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		if t.Region() == "GMS" && t.MajorVersion() <= 28 {
			// older versions don't write these fields
		} else {
			m.gender = r.ReadByte()
			m.skinColor = r.ReadByte()
			m.face = r.ReadUint32()
			notMega := r.ReadBool()
			m.mega = !notMega
			m.hair = r.ReadUint32()
		}

		terminator := byte(0xFF)
		if t.Region() == "GMS" && t.MajorVersion() <= 28 {
			terminator = 0
		}

		m.equipment = make(map[slot.Position]uint32)
		for {
			key := r.ReadByte()
			if key == terminator {
				break
			}
			m.equipment[slot.Position(key)] = r.ReadUint32()
		}

		m.maskedEquipment = make(map[slot.Position]uint32)
		for {
			key := r.ReadByte()
			if key == terminator {
				break
			}
			m.maskedEquipment[slot.Position(key)] = r.ReadUint32()
		}

		_ = r.ReadUint32() // cash weapon

		m.pets = make(map[int8]uint32)
		if (t.Region() == "GMS" && t.MajorVersion() >= 61) || t.Region() == "JMS" {
			for i := int8(0); i < 3; i++ {
				petId := r.ReadUint32()
				if petId != 0 {
					m.pets[i] = petId
				}
			}
		} else if t.Region() == "GMS" && t.MajorVersion() > 28 {
			// v48 legacy single pet: one 4-byte int.
			petId := r.ReadUint32()
			if petId != 0 {
				m.pets[0] = petId
			}
		} else {
			petId := r.ReadUint64()
			if petId != 0 {
				m.pets[0] = uint32(petId)
			}
		}
	}
}
