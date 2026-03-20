package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CreateCharacterHandle = "CreateCharacterHandle"

// CreateCharacter - CLogin::SendNewCharPacket
type CreateCharacter struct {
	name             string
	jobIndex         uint32
	subJobIndex      uint16
	face             uint32
	hair             uint32
	hairColor        uint32
	skinColor        uint32
	topTemplateId    uint32
	bottomTemplateId uint32
	shoesTemplateId  uint32
	weaponTemplateId uint32
	gender           byte
	strength         byte
	dexterity        byte
	intelligence     byte
	luck             byte
}

func (m CreateCharacter) Name() string {
	return m.name
}

func (m CreateCharacter) JobIndex() uint32 {
	return m.jobIndex
}

func (m CreateCharacter) SubJobIndex() uint16 {
	return m.subJobIndex
}

func (m CreateCharacter) Face() uint32 {
	return m.face
}

func (m CreateCharacter) Hair() uint32 {
	return m.hair
}

func (m CreateCharacter) HairColor() uint32 {
	return m.hairColor
}

func (m CreateCharacter) SkinColor() uint32 {
	return m.skinColor
}

func (m CreateCharacter) TopTemplateId() uint32 {
	return m.topTemplateId
}

func (m CreateCharacter) BottomTemplateId() uint32 {
	return m.bottomTemplateId
}

func (m CreateCharacter) ShoesTemplateId() uint32 {
	return m.shoesTemplateId
}

func (m CreateCharacter) WeaponTemplateId() uint32 {
	return m.weaponTemplateId
}

func (m CreateCharacter) Gender() byte {
	return m.gender
}

func (m CreateCharacter) Strength() byte {
	return m.strength
}

func (m CreateCharacter) Dexterity() byte {
	return m.dexterity
}

func (m CreateCharacter) Intelligence() byte {
	return m.intelligence
}

func (m CreateCharacter) Luck() byte {
	return m.luck
}

func (m CreateCharacter) Operation() string {
	return CreateCharacterHandle
}

func (m CreateCharacter) String() string {
	return fmt.Sprintf("name [%s] jobIndex [%d] subJobIndex [%d] face [%d] hair [%d] hairColor [%d] skinColor [%d] topTemplateId [%d] bottomTemplateId [%d] shoesTemplateId [%d] weaponTemplateId [%d] gender [%d] strength [%d] dexterity [%d] intelligence [%d] luck [%d]",
		m.name, m.jobIndex, m.subJobIndex, m.face, m.hair, m.hairColor, m.skinColor, m.topTemplateId, m.bottomTemplateId, m.shoesTemplateId, m.weaponTemplateId, m.gender, m.strength, m.dexterity, m.intelligence, m.luck)
}

func (m CreateCharacter) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.Name())
		if (t.Region() == "GMS" && t.MajorVersion() >= 73) || t.Region() == "JMS" {
			w.WriteInt(m.JobIndex())
		}
		if (t.Region() == "GMS" && t.MajorVersion() > 83) || t.Region() == "JMS" {
			w.WriteShort(m.SubJobIndex())
		}
		w.WriteInt(m.Face())
		w.WriteInt(m.Hair())
		if t.Region() != "JMS" {
			w.WriteInt(m.HairColor())
			w.WriteInt(m.SkinColor())
		}
		w.WriteInt(m.TopTemplateId())
		w.WriteInt(m.BottomTemplateId())
		w.WriteInt(m.ShoesTemplateId())
		w.WriteInt(m.WeaponTemplateId())
		if (t.Region() == "GMS" && t.MajorVersion() > 28) && t.Region() != "JMS" {
			w.WriteByte(m.Gender())
		}
		if t.Region() == "GMS" && t.MajorVersion() <= 28 {
			w.WriteByte(m.Strength())
			w.WriteByte(m.Dexterity())
			w.WriteByte(m.Intelligence())
			w.WriteByte(m.Luck())
		}
		return w.Bytes()
	}
}

func (m *CreateCharacter) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.name = r.ReadAsciiString()

		if (t.Region() == "GMS" && t.MajorVersion() >= 73) || t.Region() == "JMS" {
			m.jobIndex = r.ReadUint32()
		} else {
			m.jobIndex = 1
		}

		if t.Region() == "GMS" && t.MajorVersion() <= 83 {
			m.subJobIndex = 0
		} else {
			m.subJobIndex = r.ReadUint16()
		}

		m.face = r.ReadUint32()
		m.hair = r.ReadUint32()

		if t.Region() != "JMS" {
			m.hairColor = r.ReadUint32()
			m.skinColor = r.ReadUint32()
		}

		m.topTemplateId = r.ReadUint32()
		m.bottomTemplateId = r.ReadUint32()
		m.shoesTemplateId = r.ReadUint32()
		m.weaponTemplateId = r.ReadUint32()

		if (t.Region() == "GMS" && t.MajorVersion() <= 28) || t.Region() == "JMS" {
			// TODO see if this is just an assumption of if they default to account gender.
			m.gender = 0
		} else {
			m.gender = r.ReadByte()
		}

		if t.Region() == "GMS" && t.MajorVersion() <= 28 {
			m.strength = r.ReadByte()
			m.dexterity = r.ReadByte()
			m.intelligence = r.ReadByte()
			m.luck = r.ReadByte()
		} else {
			m.strength = 13
			m.dexterity = 4
			m.intelligence = 4
			m.luck = 4
		}
	}
}
