package writer

import (
	"atlas-login/character"
	"atlas-login/equipment"
	slot2 "atlas-login/equipment/slot"
	"atlas-login/pet"

	"github.com/Chronicle20/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
)

const CharacterList = "CharacterList"

func CharacterListBody(tenant tenant.Model) func(characters []character.Model, worldId world.Id, status int, pic string, availableCharacterSlots int16, characterSlots int16) BodyProducer {
	return func(characters []character.Model, worldId world.Id, status int, pic string, availableCharacterSlots int16, characterSlots int16) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(byte(status))

			if tenant.Region() == "JMS" {
				w.WriteAsciiString("")
			}

			w.WriteByte(byte(len(characters)))
			for _, x := range characters {
				WriteCharacter(tenant)(w, x, false)
			}
			if tenant.Region() == "GMS" && tenant.MajorVersion() <= 28 {
				// no trailing information
				return w.Bytes()
			}

			w.WriteBool(pic != "")
			if tenant.Region() == "GMS" {
				w.WriteInt(uint32(characterSlots))
				if tenant.MajorVersion() > 87 {
					w.WriteInt(0) // nBuyCharCount
				}
			} else if tenant.Region() == "JMS" {
				w.WriteByte(0)
				w.WriteInt(uint32(characterSlots))
				w.WriteInt(0)
			}

			return w.Bytes()
		}
	}
}

func WriteCharacter(tenant tenant.Model) func(w *response.Writer, character character.Model, viewAll bool) {
	return func(w *response.Writer, character character.Model, viewAll bool) {
		WriteCharacterStatistics(tenant)(w, character)
		WriteCharacterLook(tenant)(w, character, false)
		if !viewAll {
			w.WriteByte(0)
		}
		if character.Gm() {
			w.WriteByte(0)
			return
		}

		if tenant.Region() == "GMS" && tenant.MajorVersion() <= 28 {
			w.WriteInt(1) // auto select first character
		}

		w.WriteByte(1) // world rank enabled (next 4 int are not sent if disabled) Short??
		w.WriteInt(character.Rank())
		w.WriteInt(character.RankMove())
		w.WriteInt(character.JobRank())
		w.WriteInt(character.JobRankMove())
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
	for _, s := range slot.Slots {
		if v, ok := e.Get(s.Type); ok {
			addEquipmentIfPresent(equips, v)
		}
	}
	return equips
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

func getMaskedEquippedItemSlotMap(e equipment.Model) map[slot.Position]uint32 {
	var equips = make(map[slot.Position]uint32)
	for _, s := range slot.Slots {
		if v, ok := e.Get(s.Type); ok {
			addMaskedEquippedItemIfPresent(equips, v)
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

func WriteCharacterStatistics(tenant tenant.Model) func(w *response.Writer, character character.Model) {
	return func(w *response.Writer, character character.Model) {
		w.WriteInt(character.Id())

		name := character.Name()
		if len(name) > 13 {
			name = name[:13]
		}
		padSize := 13 - len(name)
		w.WriteByteArray([]byte(name))
		for i := 0; i < padSize; i++ {
			w.WriteByte(0x0)
		}

		w.WriteByte(character.Gender())
		w.WriteByte(character.SkinColor())
		w.WriteInt(character.Face())
		w.WriteInt(character.Hair())

		if (tenant.Region() == "GMS" && tenant.MajorVersion() > 28) || tenant.Region() == "JMS" {
			writeForEachPet(w, character.Pets(), writePetId, writeEmptyPetId)
		} else {
			if len(character.Pets()) > 0 {
				w.WriteLong(character.Pets()[0].Id()) // pet cash id
			} else {
				w.WriteLong(0)
			}
		}
		w.WriteByte(character.Level())
		w.WriteShort(uint16(character.JobId()))
		w.WriteShort(character.Strength())
		w.WriteShort(character.Dexterity())
		w.WriteShort(character.Intelligence())
		w.WriteShort(character.Luck())
		w.WriteShort(character.Hp())
		w.WriteShort(character.MaxHp())
		w.WriteShort(character.Mp())
		w.WriteShort(character.MaxMp())
		w.WriteShort(character.Ap())

		if character.HasSPTable() {
			WriteRemainingSkillInfo(w, character)
		} else {
			w.WriteShort(character.RemainingSp())
		}

		w.WriteInt(character.Experience())
		w.WriteInt16(character.Fame())
		if (tenant.Region() == "GMS" && tenant.MajorVersion() > 28) || tenant.Region() == "JMS" {
			w.WriteInt(character.GachaponExperience())
		}
		w.WriteInt(uint32(character.MapId()))
		w.WriteByte(character.SpawnPoint())

		if tenant.Region() == "GMS" {
			if tenant.MajorVersion() > 12 {
				w.WriteInt(0)
			} else {
				w.WriteInt64(0)
				w.WriteInt(0)
				w.WriteInt(0)
			}
			if tenant.MajorVersion() >= 87 {
				w.WriteShort(0) // nSubJob
			}
		} else if tenant.Region() == "JMS" {
			w.WriteShort(0)
			w.WriteLong(0)
			w.WriteInt(0)
			w.WriteInt(0)
			w.WriteInt(0)
		}
	}
}

func WriteRemainingSkillInfo(w *response.Writer, character character.Model) {

}
