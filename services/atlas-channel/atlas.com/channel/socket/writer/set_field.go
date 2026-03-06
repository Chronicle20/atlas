package writer

import (
	"atlas-channel/buddylist"
	"atlas-channel/character"
	slot2 "atlas-channel/equipment/slot"
	"atlas-channel/quest"
	model2 "atlas-channel/socket/model"
	"context"
	"math/rand"
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/inventory/slot"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const SetField = "SetField"

func WarpToMapBody(channelId channel.Id, mapId _map.Id, portalId uint32, hp uint16) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		t := tenant.MustFromContext(ctx)
		return func(options map[string]interface{}) []byte {
			if (t.Region() == "GMS" && t.MajorVersion() > 83) || t.Region() == "JMS" {
				w.WriteShort(0) // decode opt, loop with 2 decode 4s
			}
			w.WriteInt(uint32(channelId))
			if t.Region() == "JMS" {
				w.WriteByte(0)
				w.WriteInt(0)
			}
			w.WriteByte(0) // sNotifierMessage
			w.WriteByte(0) // bCharacterData
			if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
				w.WriteShort(0) // nNotifierCheck
				w.WriteByte(0)  // revive
			}
			w.WriteInt(uint32(mapId))
			w.WriteByte(byte(portalId))
			w.WriteShort(hp)
			if t.Region() == "GMS" && t.MajorVersion() > 28 {
				w.WriteBool(false) // Chasing?
				if false {
					w.WriteInt(0)
					w.WriteInt(0)
				}
			}
			w.WriteInt64(msTime(time.Now()))
			return w.Bytes()
		}
	}
}

func SetFieldBody(channelId channel.Id, c character.Model, bl buddylist.Model) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		t := tenant.MustFromContext(ctx)
		return func(options map[string]interface{}) []byte {
			if (t.Region() == "GMS" && t.MajorVersion() > 83) || t.Region() == "JMS" {
				w.WriteShort(0) // decode opt, loop with 2 decode 4s
			}
			w.WriteInt(uint32(channelId))
			if t.Region() == "JMS" {
				w.WriteByte(0)
				w.WriteInt(0)
			}
			w.WriteByte(1) // sNotifierMessage
			w.WriteByte(1) // bCharacterData

			var seedSize = 3
			if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
				w.WriteShort(0) // nNotifierCheck, if non zero STRs are encoded
			} else {
				seedSize = 4
			}

			// damage seed
			for i := 0; i < seedSize; i++ {
				w.WriteInt(rand.Uint32())
			}

			WriteCharacterInfo(l, ctx, options)(w)(c, bl)
			if (t.Region() == "GMS" && t.MajorVersion() > 83) || t.Region() == "JMS" {
				w.WriteInt(0) // logout gifts
				w.WriteInt(0)
				w.WriteInt(0)
				w.WriteInt(0)
			}
			w.WriteInt64(msTime(time.Now()))
			return w.Bytes()
		}
	}
}

func WriteCharacterInfo(l logrus.FieldLogger, ctx context.Context, options map[string]interface{}) func(w *response.Writer) func(c character.Model, bl buddylist.Model) {
	tenant := tenant.MustFromContext(ctx)
	return func(w *response.Writer) func(c character.Model, bl buddylist.Model) {
		return func(c character.Model, bl buddylist.Model) {
			if (tenant.Region() == "GMS" && tenant.MajorVersion() > 28) || tenant.Region() == "JMS" {
				w.WriteInt64(-1) // dbcharFlag
				w.WriteByte(0)   // something about SN, I believe this is size of list
			} else {
				w.WriteInt16(-1) // dbcharFlag
			}

			WriteCharacterStatistics(tenant)(w, c)
			w.WriteByte(bl.Capacity())

			if (tenant.Region() == "GMS" && tenant.MajorVersion() > 28) || tenant.Region() == "JMS" {
				if true {
					w.WriteByte(0)
				} else {
					w.WriteByte(1)
					w.WriteAsciiString("") // linked name
				}
			}
			w.WriteInt(c.Meso())

			if tenant.Region() == "JMS" {
				w.WriteInt(c.Id())
				w.WriteInt(0) // dama / gachapon items
				w.WriteInt(0)
			}
			WriteInventoryInfo(l, ctx, options)(w, c)
			WriteSkillInfo(tenant)(w, c)
			WriteQuestInfo(tenant)(w, c)
			WriteMiniGameInfo()(w, c)
			WriteRingInfo(tenant)(w, c)
			WriteTeleportInfo(tenant)(w, c)
			if tenant.Region() == "JMS" {
				w.WriteShort(0)
			}

			if (tenant.Region() == "GMS" && tenant.MajorVersion() > 28) || tenant.Region() == "JMS" {
				WriteMonsterBookInfo()(w, c)
				if tenant.Region() == "GMS" {
					WriteNewYearInfo()(w, c)
					WriteAreaInfo()(w, c)
				} else if tenant.Region() == "JMS" {
					w.WriteShort(0)
				}
				w.WriteShort(0)
			}
		}
	}
}

func WriteAreaInfo() func(w *response.Writer, c character.Model) {
	return func(w *response.Writer, c character.Model) {
		w.WriteShort(0)
	}
}

func WriteNewYearInfo() func(w *response.Writer, c character.Model) {
	return func(w *response.Writer, c character.Model) {
		w.WriteShort(0)
	}
}

func WriteMonsterBookInfo() func(w *response.Writer, c character.Model) {
	return func(w *response.Writer, c character.Model) {
		w.WriteInt(0) // cover id
		w.WriteByte(0)
		w.WriteShort(0) // card size
	}
}

func WriteTeleportInfo(tenant tenant.Model) func(w *response.Writer, c character.Model) {
	return func(w *response.Writer, c character.Model) {
		for i := 0; i < 5; i++ {
			w.WriteInt(uint32(_map.EmptyMapId))
		}

		if (tenant.Region() == "GMS" && tenant.MajorVersion() > 28) || tenant.Region() == "JMS" {
			for j := 0; j < 10; j++ {
				w.WriteInt(uint32(_map.EmptyMapId))
			}
		}
	}
}

func WriteRingInfo(tenant tenant.Model) func(w *response.Writer, c character.Model) {
	return func(w *response.Writer, c character.Model) {
		w.WriteShort(0) // crush rings

		if (tenant.Region() == "GMS" && tenant.MajorVersion() > 28) || tenant.Region() == "JMS" {
			w.WriteShort(0) // friendship rings
			w.WriteShort(0) // partner
		}
	}
}

func WriteMiniGameInfo() func(w *response.Writer, c character.Model) {
	return func(w *response.Writer, c character.Model) {
		w.WriteShort(0)
	}
}

func WriteQuestInfo(tenant tenant.Model) func(w *response.Writer, c character.Model) {
	return func(w *response.Writer, c character.Model) {
		startedQuests := quest.Started(c.Quests())
		w.WriteShort(uint16(len(startedQuests)))
		for _, q := range startedQuests {
			w.WriteShort(uint16(q.QuestId()))
			w.WriteAsciiString(q.ProgressString())
		}

		if tenant.Region() == "JMS" {
			w.WriteShort(0)
		}

		if (tenant.Region() == "GMS" && tenant.MajorVersion() > 12) || tenant.Region() == "JMS" {
			completedQuests := quest.Completed(c.Quests())
			w.WriteShort(uint16(len(completedQuests)))
			for _, q := range completedQuests {
				w.WriteShort(uint16(q.QuestId()))
				w.WriteInt64(msTime(q.CompletedAt()))
			}
		}
	}
}

func WriteSkillInfo(tenant tenant.Model) func(w *response.Writer, c character.Model) {
	return func(w *response.Writer, c character.Model) {
		var onCooldown []int

		w.WriteShort(uint16(len(c.Skills())))
		for i, s := range c.Skills() {
			w.WriteInt(uint32(s.Id()))
			w.WriteInt(uint32(s.Level()))
			w.WriteInt64(msTime(s.Expiration()))
			if s.IsFourthJob() {
				w.WriteInt(uint32(s.MasterLevel()))
			}
			if s.OnCooldown() {
				onCooldown = append(onCooldown, i)
			}
		}

		if (tenant.Region() == "GMS" && tenant.MajorVersion() > 28) || tenant.Region() == "JMS" {
			w.WriteShort(uint16(len(onCooldown)))
			for _, i := range onCooldown {
				s := c.Skills()[i]
				w.WriteInt(uint32(s.Id()))
				cd := uint32(s.CooldownExpiresAt().Sub(time.Now()).Seconds())
				w.WriteShort(uint16(cd))
			}
		}
	}
}

const (
	DefaultTime int64 = 150842304000000000
	ZeroTime    int64 = 94354848000000000
	Permanent   int64 = 150841440000000000
)

// TODO usages of this may be invalid
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

func WriteInventoryInfo(l logrus.FieldLogger, ctx context.Context, options map[string]interface{}) func(w *response.Writer, character character.Model) {
	tenant := tenant.MustFromContext(ctx)
	return func(w *response.Writer, character character.Model) {

		if (tenant.Region() == "GMS" && tenant.MajorVersion() > 12) || tenant.Region() == "JMS" {
			w.WriteByte(byte(character.Inventory().Equipable().Capacity()))
			w.WriteByte(byte(character.Inventory().Consumable().Capacity()))
			w.WriteByte(byte(character.Inventory().Setup().Capacity()))
			w.WriteByte(byte(character.Inventory().ETC().Capacity()))
			w.WriteByte(byte(character.Inventory().Cash().Capacity()))
		}

		if (tenant.Region() == "GMS" && tenant.MajorVersion() > 28) || tenant.Region() == "JMS" {
			w.WriteLong(uint64(getTime(-2)))
		}

		// regular equipment
		for _, t := range slot.Slots {
			if s, ok := character.Equipment().Get(t.Type); ok {
				WriteEquipableIfPresent(l, ctx, options)(w, s)
			}
		}

		if (tenant.Region() == "GMS" && tenant.MajorVersion() > 28) || tenant.Region() == "JMS" {
			w.WriteShort(0)
		} else {
			w.WriteByte(0)
		}

		// cash equipment
		for _, t := range slot.Slots {
			if s, ok := character.Equipment().Get(t.Type); ok {
				WriteCashEquipableIfPresent(l, ctx, options)(w, s)
			}
		}

		if (tenant.Region() == "GMS" && tenant.MajorVersion() > 28) || tenant.Region() == "JMS" {
			w.WriteShort(0)
		} else {
			w.WriteByte(0)
		}

		// equipable inventory
		if tenant.Region() == "GMS" && tenant.MajorVersion() < 28 {
			w.WriteByte(byte(character.Inventory().Equipable().Capacity()))
		}
		_ = model.ForEachSlice(model.FixedProvider(character.Inventory().Equipable().Assets()), model2.NewAssetWriter(l, ctx, options, w)(false))
		if (tenant.Region() == "GMS" && tenant.MajorVersion() > 28) || tenant.Region() == "JMS" {
			w.WriteInt(0)
		} else {
			w.WriteByte(0)
		}

		// use inventory
		if tenant.Region() == "GMS" && tenant.MajorVersion() < 28 {
			w.WriteByte(byte(character.Inventory().Consumable().Capacity()))
		}
		_ = model.ForEachSlice(model.FixedProvider(character.Inventory().Consumable().Assets()), model2.NewAssetWriter(l, ctx, options, w)(false))
		w.WriteByte(0)

		// setup inventory
		if tenant.Region() == "GMS" && tenant.MajorVersion() < 28 {
			w.WriteByte(byte(character.Inventory().Setup().Capacity()))
		}
		_ = model.ForEachSlice(model.FixedProvider(character.Inventory().Setup().Assets()), model2.NewAssetWriter(l, ctx, options, w)(false))
		w.WriteByte(0)

		// etc inventory
		if tenant.Region() == "GMS" && tenant.MajorVersion() < 28 {
			w.WriteByte(byte(character.Inventory().ETC().Capacity()))
		}
		_ = model.ForEachSlice(model.FixedProvider(character.Inventory().ETC().Assets()), model2.NewAssetWriter(l, ctx, options, w)(false))
		w.WriteByte(0)

		// cash inventory
		if tenant.Region() == "GMS" && tenant.MajorVersion() < 28 {
			w.WriteByte(byte(character.Inventory().Cash().Capacity()))
		}
		_ = model.ForEachSlice(model.FixedProvider(character.Inventory().Cash().Assets()), model2.NewAssetWriter(l, ctx, options, w)(false))
		w.WriteByte(0)
	}
}

func WriteCashEquipableIfPresent(l logrus.FieldLogger, ctx context.Context, options map[string]interface{}) func(w *response.Writer, model slot2.Model) {
	return func(w *response.Writer, model slot2.Model) {
		if model.CashEquipable != nil {
			_ = model2.NewAssetWriter(l, ctx, options, w)(false)(*model.CashEquipable)
		}
	}
}

func WriteEquipableIfPresent(l logrus.FieldLogger, ctx context.Context, options map[string]interface{}) func(w *response.Writer, model slot2.Model) {
	return func(w *response.Writer, model slot2.Model) {
		if model.Equipable != nil {
			_ = model2.NewAssetWriter(l, ctx, options, w)(false)(*model.Equipable)
		}
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

		if (tenant.Region() == "GMS" && tenant.MajorVersion() > 28) || tenant.Region() == "JMS" {
			writeForEachPet(w, character.Pets(), writePetId, writeEmptyPetId)
		} else {
			if len(character.Pets()) > 0 {
				w.WriteLong(uint64(character.Pets()[0].Id())) // pet cash id
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
			WriteRemainingSkillInfo()
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

func WriteRemainingSkillInfo() {

}
