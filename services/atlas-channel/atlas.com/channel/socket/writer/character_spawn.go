package writer

import (
	"atlas-channel/character"
	"atlas-channel/character/buff"
	"atlas-channel/equipment"
	slot2 "atlas-channel/equipment/slot"
	"atlas-channel/guild"
	"atlas-channel/pet"
	"atlas-channel/socket/model"
	"github.com/Chronicle20/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterSpawn = "CharacterSpawn"

func CharacterSpawnBody(l logrus.FieldLogger, t tenant.Model) func(c character.Model, bs []buff.Model, g guild.Model, enteringField bool) BodyProducer {
	return func(c character.Model, bs []buff.Model, g guild.Model, enteringField bool) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(c.Id())
			w.WriteByte(c.Level())
			w.WriteAsciiString(c.Name())
			if g.Id() != 0 {
				w.WriteAsciiString(g.Name())
				w.WriteShort(g.LogoBackground())
				w.WriteByte(g.LogoBackgroundColor())
				w.WriteShort(g.Logo())
				w.WriteByte(g.LogoColor())
			} else {
				w.WriteAsciiString("")
				w.WriteShort(0)
				w.WriteByte(0)
				w.WriteShort(0)
				w.WriteByte(0)
			}

			cts := model.NewCharacterTemporaryStat()
			for _, b := range bs {
				for _, ch := range b.Changes() {
					cts.AddStat(l)(t)(ch.Type(), b.SourceId(), ch.Amount(), b.ExpiresAt())
				}
			}
			cts.EncodeForeign(l, t, options)(w)

			w.WriteShort(c.JobId())

			WriteCharacterLook(t)(w, c, false)

			if (t.Region() == "GMS" && t.MajorVersion() > 87) || t.Region() == "JMS" {
				w.WriteInt(0) // driver id
				w.WriteInt(0) // passenger id
			}
			w.WriteInt(0) // choco count
			w.WriteInt(0) // item effect
			if t.Region() == "GMS" && t.MajorVersion() > 83 {
				w.WriteInt(0) // nCompletedSetItemID
			}
			w.WriteInt(0) // chair

			if enteringField {
				w.WriteInt16(c.X())
				w.WriteInt16(c.Y() - 42)
				w.WriteByte(6) // move action / stance
			} else {
				w.WriteInt16(c.X())
				w.WriteInt16(c.Y())
				w.WriteByte(c.Stance()) // move action / stance
			}

			w.WriteShort(0) // fh
			w.WriteByte(0)  // bShowAdminEffect

			// TODO clean this up.
			writeForEachPet(w, c.Pets(), func(w *response.Writer, p pet.Model) {
				m := model.Pet{
					TemplateId:  p.TemplateId(),
					Name:        p.Name(),
					Id:          p.Id(),
					X:           p.X(),
					Y:           p.Y(),
					Stance:      p.Stance(),
					Foothold:    p.Fh(),
					NameTag:     0,
					ChatBalloon: 0,
				}
				w.WriteBool(true)
				m.Encode(l, t, options)(w)
			}, func(w *response.Writer) {
			})
			w.WriteByte(0) // end of pets

			w.WriteInt(1)  // mount level
			w.WriteInt(0)  // mount exp
			w.WriteInt(0)  // mount tiredness
			w.WriteByte(0) // mini room
			w.WriteByte(0) // ad board

			// TODO GMS - JMS have different ring encoding/decoding
			w.WriteByte(0) // couple ring
			w.WriteByte(0) // friendship ring
			w.WriteByte(0) // marriage ring

			if t.Region() == "GMS" && t.MajorVersion() < 95 {
				w.WriteByte(0) // new year card
			}

			w.WriteByte(0) // berserk

			if t.Region() == "GMS" {
				if t.MajorVersion() <= 87 {
					w.WriteByte(0) // unknown (same as JMS unknown)
				}
				if t.MajorVersion() > 87 {
					w.WriteByte(0) // new year card
					w.WriteInt(0)  // nPhase
				}
			} else if t.Region() == "JMS" {
				w.WriteByte(0) // unknown
			}
			w.WriteByte(0) // team
			return w.Bytes()
		}
	}
}

func WriteCharacterLook(tenant tenant.Model) func(w *response.Writer, character character.Model, mega bool) {
	return func(w *response.Writer, character character.Model, mega bool) {
		if tenant.Region() == "GMS" && tenant.MajorVersion() <= 28 {
			// older versions don't write gender / skin color / face / mega / hair a second time
		} else {
			w.WriteByte(character.Gender())
			w.WriteByte(character.SkinColor())
			w.WriteInt(character.Face())
			w.WriteBool(!mega)
			w.WriteInt(character.Hair())
		}
		WriteCharacterEquipment(tenant)(w, character)
	}
}
func WriteCharacterEquipment(tenant tenant.Model) func(w *response.Writer, character character.Model) {
	return func(w *response.Writer, character character.Model) {
		var equips = getEquippedItemSlotMap(character.Equipment())
		var maskedEquips = getMaskedEquippedItemSlotMap(character.Equipment())
		writeEquips(tenant)(w, equips, maskedEquips)

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

		if (tenant.Region() == "GMS" && tenant.MajorVersion() > 28) || tenant.Region() == "JMS" {
			writeForEachPet(w, character.Pets(), writePetItemId, writeEmptyPetItemId)
		} else {
			if len(character.Pets()) > 0 {
				w.WriteLong(uint64(character.Pets()[0].Id())) // pet cash id
			} else {
				w.WriteLong(0)
			}
		}
	}
}

func writeEquips(tenant tenant.Model) func(w *response.Writer, equips map[slot.Position]uint32, maskedEquips map[slot.Position]uint32) {
	return func(w *response.Writer, equips map[slot.Position]uint32, maskedEquips map[slot.Position]uint32) {
		for k, v := range equips {
			w.WriteKeyValue(byte(k), v)
		}
		if tenant.Region() == "GMS" && tenant.MajorVersion() <= 28 {
			w.WriteByte(0)
		} else {
			w.WriteByte(0xFF)
		}
		for k, v := range maskedEquips {
			w.WriteKeyValue(byte(k), v)
		}
		if tenant.Region() == "GMS" && tenant.MajorVersion() <= 28 {
			w.WriteByte(0)
		} else {
			w.WriteByte(0xFF)
		}
	}
}

func getEquippedItemSlotMap(e equipment.Model) map[slot.Position]uint32 {
	var equips = make(map[slot.Position]uint32)
	for _, t := range slot.Slots {
		if s, ok := e.Get(t.Type); ok {
			addEquipmentIfPresent(equips, s)
		}
	}
	return equips
}

func getMaskedEquippedItemSlotMap(e equipment.Model) map[slot.Position]uint32 {
	var equips = make(map[slot.Position]uint32)
	for _, t := range slot.Slots {
		if s, ok := e.Get(t.Type); ok {
			addMaskedEquippedItemIfPresent(equips, s)
		}
	}
	return equips
}

func addMaskedEquippedItemIfPresent(slotMap map[slot.Position]uint32, pi slot2.Model) {
	if pi.CashEquipable != nil {
		if pi.Equipable != nil {
			slotMap[pi.Position*-1] = pi.Equipable.TemplateId()
		}
	}
}

func addEquipmentIfPresent(slotMap map[slot.Position]uint32, pi slot2.Model) {
	if pi.CashEquipable != nil {
		slotMap[pi.Position*-1] = pi.CashEquipable.TemplateId()
		return
	}
	if pi.Equipable != nil {
		slotMap[pi.Position*-1] = pi.Equipable.TemplateId()
	}
}

func writePetItemId(w *response.Writer, p pet.Model) {
	w.WriteInt(p.TemplateId())
}

func writeEmptyPetItemId(w *response.Writer) {
	w.WriteInt(0)
}

func writeForEachPet(w *response.Writer, ps []pet.Model, pe func(w *response.Writer, p pet.Model), pne func(w *response.Writer)) {
	for i := int8(0); i < 3; i++ {
		if ps == nil {
			pne(w)
			continue
		}

		var p *pet.Model
		for _, rp := range ps {
			if rp.Slot() == i {
				p = &rp
			}
		}
		if p != nil {
			pe(w, *p)
		} else {
			pne(w)
		}
	}
}

func writePetId(w *response.Writer, pet pet.Model) {
	w.WriteLong(uint64(pet.Id()))
}

func writeEmptyPetId(w *response.Writer) {
	w.WriteLong(0)
}
