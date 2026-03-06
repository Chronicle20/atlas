package writer

import (
	"atlas-login/character"
	"atlas-login/pet"
	"atlas-login/socket/model"
	"context"

	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterList = "CharacterList"

func CharacterListBody(characters []character.Model, worldId world.Id, status int, pic string, availableCharacterSlots int16, characterSlots int16) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		t := tenant.MustFromContext(ctx)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(byte(status))

			if t.Region() == "JMS" {
				w.WriteAsciiString("")
			}

			w.WriteByte(byte(len(characters)))
			for _, x := range characters {
				WriteCharacter(l, ctx)(w, options)(x, false)
			}
			if t.Region() == "GMS" && t.MajorVersion() <= 28 {
				// no trailing information
				return w.Bytes()
			}

			w.WriteBool(pic != "")
			if t.Region() == "GMS" {
				w.WriteInt(uint32(characterSlots))
				if t.MajorVersion() > 87 {
					w.WriteInt(0) // nBuyCharCount
				}
			} else if t.Region() == "JMS" {
				w.WriteByte(0)
				w.WriteInt(uint32(characterSlots))
				w.WriteInt(0)
			}

			return w.Bytes()
		}
	}
}

func WriteCharacter(l logrus.FieldLogger, ctx context.Context) func(w *response.Writer, options map[string]interface{}) func(character character.Model, viewAll bool) {
	t := tenant.MustFromContext(ctx)
	return func(w *response.Writer, options map[string]interface{}) func(character character.Model, viewAll bool) {
		return func(character character.Model, viewAll bool) {
			WriteCharacterStatistics(t)(w, character)
			ava := model.NewFromCharacter(character, false)
			w.WriteByteArray(ava.Encode(l, ctx)(options))
			if !viewAll {
				w.WriteByte(0)
			}
			if character.Gm() {
				w.WriteByte(0)
				return
			}

			if t.Region() == "GMS" && t.MajorVersion() <= 28 {
				w.WriteInt(1) // auto select first character
			}

			w.WriteByte(1) // world rank enabled (next 4 int are not sent if disabled) Short??
			w.WriteInt(character.Rank())
			w.WriteInt(character.RankMove())
			w.WriteInt(character.JobRank())
			w.WriteInt(character.JobRankMove())
		}
	}
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
	w.WriteLong(pet.CashId())
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
				w.WriteLong(character.Pets()[0].CashId()) // pet cash id
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
