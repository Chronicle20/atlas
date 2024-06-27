package writer

import (
	"atlas-login/character"
	"atlas-login/character/inventory"
	"atlas-login/pet"
	"atlas-login/tenant"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterList = "CharacterList"

func CharacterListBody(l logrus.FieldLogger, tenant tenant.Model) func(characters []character.Model, worldId byte, status int, cannotBypassPic bool, pic string, availableCharacterSlots int16, characterSlots int16) BodyProducer {
	return func(characters []character.Model, worldId byte, status int, cannotBypassPic bool, pic string, availableCharacterSlots int16, characterSlots int16) BodyProducer {
		return func(op uint16, options map[string]interface{}) []byte {
			w := response.NewWriter(l)
			w.WriteShort(op)
			w.WriteByte(byte(status))

			if tenant.Region == "JMS" {
				w.WriteAsciiString("")
			}

			w.WriteByte(byte(len(characters)))
			for _, x := range characters {
				WriteCharacter(tenant)(w, x, false)
			}

			w.WriteByte(2) // 0 is create PIC, 1 is enter PIC, 2 is normal
			if tenant.Region == "GMS" {
				w.WriteInt(uint32(characterSlots))
				if tenant.MajorVersion > 87 {
					w.WriteInt(0) // nBuyCharCount
				}
			} else if tenant.Region == "JMS" {
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
		WriteCharacterLook(w, character, false)
		if !viewAll {
			w.WriteByte(0)
		}
		if character.Properties().Gm() || character.Properties().GmJob() {
			w.WriteByte(0)
			return
		}
		w.WriteByte(1) // world rank enabled (next 4 int are not sent if disabled) Short??
		w.WriteInt(uint32(character.Properties().Rank()))
		w.WriteInt(uint32(character.Properties().RankMove()))
		w.WriteInt(uint32(character.Properties().JobRank()))
		w.WriteInt(uint32(character.Properties().JobRankMove()))
	}
}

func WriteCharacterLook(w *response.Writer, character character.Model, mega bool) {
	w.WriteByte(character.Properties().Gender())
	w.WriteByte(character.Properties().SkinColor())
	w.WriteInt(character.Properties().Face())
	w.WriteBool(!mega)
	w.WriteInt(character.Properties().Hair())
	WriteCharacterEquipment(w, character)
}

func WriteCharacterEquipment(w *response.Writer, character character.Model) {

	var equips = getEquippedItemSlotMap(character.Equipment())
	var maskedEquips = make(map[int16]uint32)
	writeEquips(w, equips, maskedEquips)

	var weapon *inventory.EquippedItem
	for _, x := range character.Equipment() {
		if x.InWeaponSlot() {
			weapon = &x
			break
		}
	}
	if weapon != nil {
		w.WriteInt(weapon.ItemId())
	} else {
		w.WriteInt(0)
	}

	writeForEachPet(w, character.Pets(), writePetItemId, writeEmptyPetItemId)
}

func writeEquips(w *response.Writer, equips map[int16]uint32, maskedEquips map[int16]uint32) {
	for k, v := range equips {
		w.WriteKeyValue(byte(k), v)
	}
	w.WriteByte(0xFF)
	for k, v := range maskedEquips {
		w.WriteKeyValue(byte(k), v)
	}
	w.WriteByte(0xFF)
}

func getEquippedItemSlotMap(e []inventory.EquippedItem) map[int16]uint32 {
	var equips = make(map[int16]uint32, 0)
	for _, x := range e {
		if x.NotInWeaponSlot() {
			y := x.InvertSlot()
			equips[y.Slot()] = y.ItemId()
		}
	}
	return equips
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
		w.WriteInt(character.Properties().Id())

		name := character.Properties().Name()
		if len(name) > 13 {
			name = name[:13]
		}
		padSize := 13 - len(name)
		w.WriteByteArray([]byte(name))
		for i := 0; i < padSize; i++ {
			w.WriteByte(0x0)
		}

		w.WriteByte(character.Properties().Gender())
		w.WriteByte(character.Properties().SkinColor())
		w.WriteInt(character.Properties().Face())
		w.WriteInt(character.Properties().Hair())
		writeForEachPet(w, character.Pets(), writePetId, writeEmptyPetId)
		w.WriteByte(character.Properties().Level())
		w.WriteShort(character.Properties().JobId())
		w.WriteShort(character.Properties().Strength())
		w.WriteShort(character.Properties().Dexterity())
		w.WriteShort(character.Properties().Intelligence())
		w.WriteShort(character.Properties().Luck())
		w.WriteShort(character.Properties().Hp())
		w.WriteShort(character.Properties().MaxHp())
		w.WriteShort(character.Properties().Mp())
		w.WriteShort(character.Properties().MaxMp())
		w.WriteShort(character.Properties().Ap())

		if character.Properties().HasSPTable() {
			WriteRemainingSkillInfo(w, character)
		} else {
			w.WriteShort(character.Properties().RemainingSp())
		}

		w.WriteInt(character.Properties().Experience())
		w.WriteShort(uint16(character.Properties().Fame()))
		w.WriteInt(character.Properties().GachaponExperience())
		w.WriteInt(character.Properties().MapId())
		w.WriteByte(character.Properties().SpawnPoint())

		if tenant.Region == "GMS" {
			w.WriteInt(0)
			if tenant.MajorVersion >= 87 {
				w.WriteShort(0) // nSubJob
			}
		} else if tenant.Region == "JMS" {
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
