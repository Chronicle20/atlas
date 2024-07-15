package writer

import (
	"atlas-channel/character"
	"atlas-channel/pet"
	"atlas-channel/tenant"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
	"math/rand"
	"time"
)

const SetField = "SetField"

func SetFieldBody(l logrus.FieldLogger, tenant tenant.Model) func(channelId byte, c character.Model) BodyProducer {
	return func(channelId byte, c character.Model) BodyProducer {
		return func(op uint16, options map[string]interface{}) []byte {
			w := response.NewWriter(l)
			w.WriteShort(op)
			if (tenant.Region == "GMS" && tenant.MajorVersion > 83) || tenant.Region == "JMS" {
				w.WriteShort(0) // decode opt, loop with 2 decode 4s
			}
			w.WriteInt(uint32(channelId))
			if tenant.Region == "JMS" {
				w.WriteByte(0)
				w.WriteInt(0)
			}
			w.WriteByte(1)  // sNotifierMessage
			w.WriteByte(1)  // bCharacterData
			w.WriteShort(0) // nNotifierCheck, if non zero STRs are encoded

			// damage seed
			for i := 0; i < 3; i++ {
				w.WriteInt(rand.Uint32())
			}
			WriteCharacterInfo(tenant)(w)(c)
			if (tenant.Region == "GMS" && tenant.MajorVersion > 83) || tenant.Region == "JMS" {
				w.WriteInt(0) // logout gifts
				w.WriteInt(0)
				w.WriteInt(0)
				w.WriteInt(0)
			}
			w.WriteInt64(getTime(timeNow()))
			return w.Bytes()
		}
	}
}

func WriteCharacterInfo(tenant tenant.Model) func(w *response.Writer) func(m character.Model) {
	return func(w *response.Writer) func(m character.Model) {
		return func(m character.Model) {
			w.WriteInt64(-1) // dbcharFlag
			w.WriteByte(0)   // something about SN, I believe this is size of list
			WriteCharacterStatistics(tenant)(w, m)
			w.WriteByte(0) // buddy list capacity

			if true {
				w.WriteByte(0)
			} else {
				w.WriteByte(1)
				w.WriteAsciiString("") // linked name
			}
			w.WriteInt(m.Meso())

			if tenant.Region == "JMS" {
				w.WriteInt(m.Id())
				w.WriteInt(0) // dama / gachapon items
				w.WriteInt(0)
			}
			WriteInventoryInfo(tenant)(w, m)
			WriteSkillInfo(tenant)(w, m)
			WriteQuestInfo(tenant)(w, m)
			WriteMiniGameInfo(tenant)(w, m)
			WriteRingInfo(tenant)(w, m)
			WriteTeleportInfo(tenant)(w, m)
			if tenant.Region == "JMS" {
				w.WriteShort(0)
			}
			WriteMonsterBookInfo(tenant)(w, m)
			if tenant.Region == "GMS" {
				WriteNewYearInfo(tenant)(w, m)
				WriteAreaInfo(tenant)(w, m)
			} else if tenant.Region == "JMS" {
				w.WriteShort(0)
			}
			w.WriteShort(0)
		}
	}
}

func WriteAreaInfo(tenant tenant.Model) func(w *response.Writer, character character.Model) {
	return func(w *response.Writer, character character.Model) {
		w.WriteShort(0)
	}
}

func WriteNewYearInfo(tenant tenant.Model) func(w *response.Writer, character character.Model) {
	return func(w *response.Writer, character character.Model) {
		w.WriteShort(0)
	}
}

func WriteMonsterBookInfo(tenant tenant.Model) func(w *response.Writer, character character.Model) {
	return func(w *response.Writer, character character.Model) {
		w.WriteInt(0) // cover id
		w.WriteByte(0)
		w.WriteShort(0) // card size
	}
}

func WriteTeleportInfo(tenant tenant.Model) func(w *response.Writer, character character.Model) {
	return func(w *response.Writer, character character.Model) {
		for i := 0; i < 5; i++ {
			w.WriteInt(999999999)
		}
		for j := 0; j < 10; j++ {
			w.WriteInt(999999999)
		}
	}
}

func WriteRingInfo(tenant tenant.Model) func(w *response.Writer, character character.Model) {
	return func(w *response.Writer, character character.Model) {
		w.WriteShort(0) // crush rings
		w.WriteShort(0) // friendship rings
		w.WriteShort(0) // partner
	}
}

func WriteMiniGameInfo(tenant tenant.Model) func(w *response.Writer, character character.Model) {
	return func(w *response.Writer, character character.Model) {
		w.WriteShort(0)
	}
}

func WriteQuestInfo(tenant tenant.Model) func(w *response.Writer, character character.Model) {
	return func(w *response.Writer, character character.Model) {
		w.WriteShort(0) // started size
		if tenant.Region == "JMS" {
			w.WriteShort(0)
		}
		w.WriteShort(0) // completed size
	}
}

func WriteSkillInfo(tenant tenant.Model) func(w *response.Writer, character character.Model) {
	return func(w *response.Writer, character character.Model) {
		w.WriteShort(0) // skill size
		w.WriteShort(0) // cooldowns
	}
}

const (
	DefaultTime int64 = 150842304000000000
	ZeroTime    int64 = 94354848000000000
	Permanent   int64 = 150841440000000000
)

func getTime(utcTimestamp int64) int64 {
	if utcTimestamp < 0 && utcTimestamp >= -3 {
		if utcTimestamp == -1 {
			return DefaultTime //high number ll
		} else if utcTimestamp == -2 {
			return ZeroTime
		} else {
			return Permanent
		}
	}

	ftUtOffset := 116444736010800000 + (10000 * timeNow())
	return utcTimestamp*10000 + ftUtOffset
}

func timeNow() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func WriteInventoryInfo(tenant tenant.Model) func(w *response.Writer, character character.Model) {
	return func(w *response.Writer, character character.Model) {
		w.WriteByte(byte(character.Inventory().Equipable().Capacity()))
		w.WriteByte(byte(character.Inventory().Use().Capacity()))
		w.WriteByte(byte(character.Inventory().Setup().Capacity()))
		w.WriteByte(byte(character.Inventory().Etc().Capacity()))
		w.WriteByte(byte(character.Inventory().Cash().Capacity()))

		w.WriteLong(uint64(getTime(-2)))

		// regular equipment
		w.WriteShort(0)
		// cash equipment
		w.WriteShort(0)
		// equipable inventory
		w.WriteInt(0)
		// use inventory
		w.WriteByte(0)
		// setup inventory
		w.WriteByte(0)
		// etc inventory
		w.WriteByte(0)
		// cash inventory
		w.WriteByte(0)
	}
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
		writeForEachPet(w, character.Pets(), writePetId, writeEmptyPetId)
		w.WriteByte(character.Level())
		w.WriteShort(character.JobId())
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
		w.WriteInt(character.GachaponExperience())
		w.WriteInt(character.MapId())
		w.WriteByte(character.SpawnPoint())

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

func WriteRemainingSkillInfo(w *response.Writer, character character.Model) {

}
