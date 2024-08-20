package writer

import (
	"atlas-channel/character"
	"atlas-channel/character/equipment"
	"atlas-channel/character/equipment/slot"
	"atlas-channel/pet"
	"atlas-channel/socket/model"
	"atlas-channel/tenant"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterSpawn = "CharacterSpawn"

func CharacterSpawnBody(l logrus.FieldLogger, t tenant.Model) func(c character.Model, enteringField bool) BodyProducer {
	return func(c character.Model, enteringField bool) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(c.Id())
			w.WriteByte(c.Level())
			w.WriteAsciiString(c.Name())
			var guild = false
			if !guild {
				w.WriteAsciiString("")
				w.WriteShort(0)
				w.WriteByte(0)
				w.WriteShort(0)
				w.WriteByte(0)
			}
			cts := model.CharacterTemporaryStat{}
			cts.Encode(l, t, options)(w)

			w.WriteShort(c.JobId())

			WriteCharacterLook(t)(w, c, false)

			if (t.Region == "GMS" && t.MajorVersion > 87) || t.Region == "JMS" {
				w.WriteInt(0) // driver id
				w.WriteInt(0) // passenger id
			}
			w.WriteInt(0) // choco count
			w.WriteInt(0) // item effect
			if t.Region == "GMS" && t.MajorVersion > 83 {
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
			w.WriteByte(0)  // end of pets
			w.WriteInt(1)   // mount level
			w.WriteInt(0)   // mount exp
			w.WriteInt(0)   // mount tiredness
			w.WriteByte(0)  // mini room
			w.WriteByte(0)  // ad board

			// TODO GMS - JMS have different ring encoding/decoding
			w.WriteByte(0) // couple ring
			w.WriteByte(0) // friendship ring
			w.WriteByte(0) // marriage ring

			if t.Region == "GMS" && t.MajorVersion < 95 {
				w.WriteByte(0) // new year card
			}

			w.WriteByte(0) // berserk

			if t.Region == "GMS" {
				if t.MajorVersion <= 87 {
					w.WriteByte(0) // unknown (same as JMS unknown)
				}
				if t.MajorVersion > 87 {
					w.WriteByte(0) // new year card
					w.WriteInt(0)  // nPhase
				}
			} else if t.Region == "JMS" {
				w.WriteByte(0) // unknown
			}
			w.WriteByte(0) // team
			return w.Bytes()
		}
	}
}

func WriteCharacterLook(tenant tenant.Model) func(w *response.Writer, character character.Model, mega bool) {
	return func(w *response.Writer, character character.Model, mega bool) {
		if tenant.Region == "GMS" && tenant.MajorVersion <= 28 {
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

		if (tenant.Region == "GMS" && tenant.MajorVersion > 28) || tenant.Region == "JMS" {
			writeForEachPet(w, character.Pets(), writePetItemId, writeEmptyPetItemId)
		} else {
			if len(character.Pets()) > 0 {
				w.WriteLong(character.Pets()[0].Id()) // pet cash id
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
		if tenant.Region == "GMS" && tenant.MajorVersion <= 28 {
			w.WriteByte(0)
		} else {
			w.WriteByte(0xFF)
		}
		for k, v := range maskedEquips {
			w.WriteKeyValue(byte(k), v)
		}
		if tenant.Region == "GMS" && tenant.MajorVersion <= 28 {
			w.WriteByte(0)
		} else {
			w.WriteByte(0xFF)
		}
	}
}

func getEquippedItemSlotMap(e equipment.Model) map[slot.Position]uint32 {
	var equips = make(map[slot.Position]uint32)
	addEquipmentIfPresent(equips, e.Hat())
	addEquipmentIfPresent(equips, e.Medal())
	addEquipmentIfPresent(equips, e.Forehead())
	addEquipmentIfPresent(equips, e.Ring1())
	addEquipmentIfPresent(equips, e.Ring2())
	addEquipmentIfPresent(equips, e.Eye())
	addEquipmentIfPresent(equips, e.Earring())
	addEquipmentIfPresent(equips, e.Shoulder())
	addEquipmentIfPresent(equips, e.Cape())
	addEquipmentIfPresent(equips, e.Top())
	addEquipmentIfPresent(equips, e.Pendant())
	addEquipmentIfPresent(equips, e.Weapon())
	addEquipmentIfPresent(equips, e.Shield())
	addEquipmentIfPresent(equips, e.Gloves())
	addEquipmentIfPresent(equips, e.Bottom())
	addEquipmentIfPresent(equips, e.Belt())
	addEquipmentIfPresent(equips, e.Ring3())
	addEquipmentIfPresent(equips, e.Ring4())
	addEquipmentIfPresent(equips, e.Shoes())
	return equips
}

func getMaskedEquippedItemSlotMap(e equipment.Model) map[slot.Position]uint32 {
	var equips = make(map[slot.Position]uint32)
	addMaskedEquippedItemIfPresent(equips, e.Hat())
	addMaskedEquippedItemIfPresent(equips, e.Medal())
	addMaskedEquippedItemIfPresent(equips, e.Forehead())
	addMaskedEquippedItemIfPresent(equips, e.Ring1())
	addMaskedEquippedItemIfPresent(equips, e.Ring2())
	addMaskedEquippedItemIfPresent(equips, e.Eye())
	addMaskedEquippedItemIfPresent(equips, e.Earring())
	addMaskedEquippedItemIfPresent(equips, e.Shoulder())
	addMaskedEquippedItemIfPresent(equips, e.Cape())
	addMaskedEquippedItemIfPresent(equips, e.Top())
	addMaskedEquippedItemIfPresent(equips, e.Pendant())
	addMaskedEquippedItemIfPresent(equips, e.Weapon())
	addMaskedEquippedItemIfPresent(equips, e.Shield())
	addMaskedEquippedItemIfPresent(equips, e.Gloves())
	addMaskedEquippedItemIfPresent(equips, e.Bottom())
	addMaskedEquippedItemIfPresent(equips, e.Belt())
	addMaskedEquippedItemIfPresent(equips, e.Ring3())
	addMaskedEquippedItemIfPresent(equips, e.Ring4())
	addMaskedEquippedItemIfPresent(equips, e.Shoes())
	return equips
}

func addMaskedEquippedItemIfPresent(slotMap map[slot.Position]uint32, pi slot.Model) {
	if pi.CashEquipable != nil {
		if pi.Equipable != nil {
			slotMap[pi.Position*-1] = pi.Equipable.ItemId()
		}
	}
}

func addEquipmentIfPresent(slotMap map[slot.Position]uint32, pi slot.Model) {
	if pi.CashEquipable != nil {
		slotMap[pi.Position*-1] = pi.CashEquipable.ItemId()
		return
	}
	if pi.Equipable != nil {
		slotMap[pi.Position*-1] = pi.Equipable.ItemId()
	}
}

func writePetItemId(w *response.Writer, p pet.Model) {
	w.WriteInt(p.ItemId())
}

func writeEmptyPetItemId(w *response.Writer) {
	w.WriteInt(0)
}

func writeForEachPet(w *response.Writer, ps []pet.Model, pe func(w *response.Writer, p pet.Model), pne func(w *response.Writer)) {
	for i := 0; i < 3; i++ {
		if ps != nil && len(ps) > i {
			pe(w, ps[i])
		} else {
			pne(w)
		}
	}
}

func writePetId(w *response.Writer, pet pet.Model) {
	w.WriteLong(pet.Id())
}

func writeEmptyPetId(w *response.Writer) {
	w.WriteLong(0)
}
