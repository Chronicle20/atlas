package model

import (
	"atlas-channel/character"

	"github.com/Chronicle20/atlas-constants/inventory/slot"
	packetmodel "github.com/Chronicle20/atlas-packet/model"
)

func NewFromCharacter(c character.Model, mega bool) packetmodel.Avatar {
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

	return packetmodel.NewAvatar(c.Gender(), c.SkinColor(), c.Face(), mega, c.Hair(), equips, maskedEquips, pets)
}
