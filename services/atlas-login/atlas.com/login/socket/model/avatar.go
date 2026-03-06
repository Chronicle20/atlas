package model

import (
	"atlas-login/character"
	"context"

	"github.com/Chronicle20/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
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

func NewFromCharacter(c character.Model, mega bool) Avatar {
	var equips = make(map[slot.Position]uint32)
	for _, t := range slot.Slots {
		if s, ok := c.Equipment().Get(t.Type); ok {
			if s.CashEquipable != nil {
				if s.Equipable != nil {
					equips[s.Position*-1] = s.Equipable.TemplateId()
				}
			}
		}
	}
	var maskedEquips = make(map[slot.Position]uint32)
	for _, t := range slot.Slots {
		if s, ok := c.Equipment().Get(t.Type); ok {
			if s.CashEquipable != nil {
				maskedEquips[s.Position*-1] = s.CashEquipable.TemplateId()
				continue
			}
			if s.Equipable != nil {
				maskedEquips[s.Position*-1] = s.Equipable.TemplateId()
			}
		}
	}
	var pets = make(map[int8]uint32)
	for _, p := range c.Pets() {
		pets[p.Slot()] = p.TemplateId()
	}

	return Avatar{
		gender:          c.Gender(),
		skinColor:       c.SkinColor(),
		face:            c.Face(),
		mega:            mega,
		hair:            c.Hair(),
		equipment:       equips,
		maskedEquipment: maskedEquips,
		pets:            pets,
	}
}

func (m *Avatar) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
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

		//var weapon *inventory.EquippedItem
		//for _, x := range character.Equipment() {
		//	if x.InWeaponSlot() {
		//		weapon = &x
		//		break
		//	}
		//}
		//if weapon != nil {
		//	w.WriteInt(weapon.ItemId())
		//} else {
		w.WriteInt(0)
		//}

		// TODO confirm whether or not we should be writing TemplateId or CashId here
		if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
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
		} else {
			if len(m.pets) > 0 {
				w.WriteLong(uint64(m.pets[0])) // pet cash id
			} else {
				w.WriteLong(0)
			}
		}
		return w.Bytes()
	}
}
