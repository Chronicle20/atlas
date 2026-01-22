package writer

import (
	"atlas-channel/asset"
	"atlas-channel/buddylist"
	"atlas-channel/character"
	slot2 "atlas-channel/equipment/slot"
	"atlas-channel/quest"
	"errors"
	"math"
	"math/rand"
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas-constants/item"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const SetField = "SetField"

func WarpToMapBody(l logrus.FieldLogger, tenant tenant.Model) func(channelId channel.Id, mapId _map.Id, portalId uint32, hp uint16) BodyProducer {
	return func(channelId channel.Id, mapId _map.Id, portalId uint32, hp uint16) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			if (tenant.Region() == "GMS" && tenant.MajorVersion() > 83) || tenant.Region() == "JMS" {
				w.WriteShort(0) // decode opt, loop with 2 decode 4s
			}
			w.WriteInt(uint32(channelId))
			if tenant.Region() == "JMS" {
				w.WriteByte(0)
				w.WriteInt(0)
			}
			w.WriteByte(0) // sNotifierMessage
			w.WriteByte(0) // bCharacterData
			if (tenant.Region() == "GMS" && tenant.MajorVersion() > 28) || tenant.Region() == "JMS" {
				w.WriteShort(0) // nNotifierCheck
				w.WriteByte(0)  // revive
			}
			w.WriteInt(uint32(mapId))
			w.WriteByte(byte(portalId))
			w.WriteShort(hp)
			if tenant.Region() == "GMS" && tenant.MajorVersion() > 28 {
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

func SetFieldBody(l logrus.FieldLogger, tenant tenant.Model) func(channelId channel.Id, c character.Model, bl buddylist.Model) BodyProducer {
	return func(channelId channel.Id, c character.Model, bl buddylist.Model) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			if (tenant.Region() == "GMS" && tenant.MajorVersion() > 83) || tenant.Region() == "JMS" {
				w.WriteShort(0) // decode opt, loop with 2 decode 4s
			}
			w.WriteInt(uint32(channelId))
			if tenant.Region() == "JMS" {
				w.WriteByte(0)
				w.WriteInt(0)
			}
			w.WriteByte(1) // sNotifierMessage
			w.WriteByte(1) // bCharacterData

			var seedSize = 3
			if (tenant.Region() == "GMS" && tenant.MajorVersion() > 28) || tenant.Region() == "JMS" {
				w.WriteShort(0) // nNotifierCheck, if non zero STRs are encoded
			} else {
				seedSize = 4
			}

			// damage seed
			for i := 0; i < seedSize; i++ {
				w.WriteInt(rand.Uint32())
			}

			WriteCharacterInfo(tenant)(w)(c, bl)
			if (tenant.Region() == "GMS" && tenant.MajorVersion() > 83) || tenant.Region() == "JMS" {
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

func WriteCharacterInfo(tenant tenant.Model) func(w *response.Writer) func(c character.Model, bl buddylist.Model) {
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
			WriteInventoryInfo(tenant)(w, c)
			WriteSkillInfo(tenant)(w, c)
			WriteQuestInfo(tenant)(w, c)
			WriteMiniGameInfo(tenant)(w, c)
			WriteRingInfo(tenant)(w, c)
			WriteTeleportInfo(tenant)(w, c)
			if tenant.Region() == "JMS" {
				w.WriteShort(0)
			}

			if (tenant.Region() == "GMS" && tenant.MajorVersion() > 28) || tenant.Region() == "JMS" {
				WriteMonsterBookInfo(tenant)(w, c)
				if tenant.Region() == "GMS" {
					WriteNewYearInfo(tenant)(w, c)
					WriteAreaInfo(tenant)(w, c)
				} else if tenant.Region() == "JMS" {
					w.WriteShort(0)
				}
				w.WriteShort(0)
			}
		}
	}
}

func WriteAreaInfo(tenant tenant.Model) func(w *response.Writer, c character.Model) {
	return func(w *response.Writer, c character.Model) {
		w.WriteShort(0)
	}
}

func WriteNewYearInfo(tenant tenant.Model) func(w *response.Writer, c character.Model) {
	return func(w *response.Writer, c character.Model) {
		w.WriteShort(0)
	}
}

func WriteMonsterBookInfo(tenant tenant.Model) func(w *response.Writer, c character.Model) {
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

func WriteMiniGameInfo(tenant tenant.Model) func(w *response.Writer, c character.Model) {
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

func WriteInventoryInfo(tenant tenant.Model) func(w *response.Writer, character character.Model) {
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
				WriteEquipableIfPresent(tenant)(w, s)
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
				WriteCashEquipableIfPresent(tenant)(w, s)
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
		_ = model.ForEachSlice(model.FixedProvider(character.Inventory().Equipable().Assets()), WriteAssetInfo(tenant)(false)(w))
		if (tenant.Region() == "GMS" && tenant.MajorVersion() > 28) || tenant.Region() == "JMS" {
			w.WriteInt(0)
		} else {
			w.WriteByte(0)
		}

		// use inventory
		if tenant.Region() == "GMS" && tenant.MajorVersion() < 28 {
			w.WriteByte(byte(character.Inventory().Consumable().Capacity()))
		}
		_ = model.ForEachSlice(model.FixedProvider(character.Inventory().Consumable().Assets()), WriteAssetInfo(tenant)(false)(w))
		w.WriteByte(0)

		// setup inventory
		if tenant.Region() == "GMS" && tenant.MajorVersion() < 28 {
			w.WriteByte(byte(character.Inventory().Setup().Capacity()))
		}
		_ = model.ForEachSlice(model.FixedProvider(character.Inventory().Setup().Assets()), WriteAssetInfo(tenant)(false)(w))
		w.WriteByte(0)

		// etc inventory
		if tenant.Region() == "GMS" && tenant.MajorVersion() < 28 {
			w.WriteByte(byte(character.Inventory().ETC().Capacity()))
		}
		_ = model.ForEachSlice(model.FixedProvider(character.Inventory().ETC().Assets()), WriteAssetInfo(tenant)(false)(w))
		w.WriteByte(0)

		// cash inventory
		if tenant.Region() == "GMS" && tenant.MajorVersion() < 28 {
			w.WriteByte(byte(character.Inventory().Cash().Capacity()))
		}
		_ = model.ForEachSlice(model.FixedProvider(character.Inventory().Cash().Assets()), WriteAssetInfo(tenant)(false)(w))
		w.WriteByte(0)
	}
}

func WriteCashEquipableIfPresent(tenant tenant.Model) func(w *response.Writer, model slot2.Model) {
	return func(w *response.Writer, model slot2.Model) {
		if model.CashEquipable != nil {
			_ = WriteCashEquipableInfo(tenant)(w, false)(*model.CashEquipable)
		}
	}
}

func WriteCashEquipableInfo(tenant tenant.Model) func(w *response.Writer, zeroPosition bool) model.Operator[asset.Model[asset.CashEquipableReferenceData]] {
	return func(w *response.Writer, zeroPosition bool) model.Operator[asset.Model[asset.CashEquipableReferenceData]] {
		return func(e asset.Model[asset.CashEquipableReferenceData]) error {
			slot := e.Slot()
			if !zeroPosition {
				slot = int16(math.Abs(float64(slot)))
				if slot > 100 {
					slot -= 100
				}
				if (tenant.Region() == "GMS" && tenant.MajorVersion() > 28) || tenant.Region() == "JMS" {
					w.WriteShort(uint16(slot))
				} else {
					w.WriteByte(byte(slot))
				}
			}

			if (tenant.Region() == "GMS" && tenant.MajorVersion() > 12) || tenant.Region() == "JMS" {
				w.WriteByte(1)
			}
			w.WriteInt(e.TemplateId())
			w.WriteBool(true)
			if true {
				w.WriteInt64(e.ReferenceData().CashId())
			}
			w.WriteInt64(msTime(e.Expiration()))
			w.WriteByte(byte(e.ReferenceData().GetSlots()))
			w.WriteByte(e.ReferenceData().GetLevel())
			if tenant.Region() == "JMS" {
				w.WriteByte(0)
			}
			w.WriteShort(e.ReferenceData().Strength())
			w.WriteShort(e.ReferenceData().Dexterity())
			w.WriteShort(e.ReferenceData().Intelligence())
			w.WriteShort(e.ReferenceData().Luck())
			w.WriteShort(e.ReferenceData().HP())
			w.WriteShort(e.ReferenceData().MP())
			w.WriteShort(e.ReferenceData().WeaponAttack())
			w.WriteShort(e.ReferenceData().MagicAttack())
			w.WriteShort(e.ReferenceData().WeaponDefense())
			w.WriteShort(e.ReferenceData().MagicDefense())
			w.WriteShort(e.ReferenceData().Accuracy())
			w.WriteShort(e.ReferenceData().Avoidability())
			w.WriteShort(e.ReferenceData().Hands())
			w.WriteShort(e.ReferenceData().Speed())
			w.WriteShort(e.ReferenceData().Jump())

			if (tenant.Region() == "GMS" && tenant.MajorVersion() > 12) || tenant.Region() == "JMS" {
				w.WriteAsciiString("") // TODO retrieve owner name from id
				w.WriteShort(0)        // TODO need to create flags bitmask

				if (tenant.Region() == "GMS" && tenant.MajorVersion() > 28) || tenant.Region() == "JMS" {
					for i := 0; i < 10; i++ {
						w.WriteByte(0x40)
					}
					w.WriteLong(uint64(getTime(-2)))
					w.WriteInt32(-1)
				}
			}
			return nil
		}
	}
}

func WriteCashItemInfo(zeroPosition bool) func(w *response.Writer) model.Operator[asset.Model[asset.CashReferenceData]] {
	return func(w *response.Writer) model.Operator[asset.Model[asset.CashReferenceData]] {
		return func(i asset.Model[asset.CashReferenceData]) error {
			if !zeroPosition {
				w.WriteInt8(int8(i.Slot()))
			}
			w.WriteByte(2)
			w.WriteInt(i.TemplateId())
			w.WriteBool(true)
			w.WriteInt64(i.ReferenceData().CashId())
			w.WriteInt64(msTime(i.Expiration()))
			w.WriteShort(uint16(i.Quantity()))
			w.WriteAsciiString("") // TODO
			w.WriteShort(i.ReferenceData().Flag())
			return nil
		}
	}
}

func WritePetCashItemInfo(zeroPosition bool) func(w *response.Writer) model.Operator[asset.Model[asset.PetReferenceData]] {
	return func(w *response.Writer) model.Operator[asset.Model[asset.PetReferenceData]] {
		return func(i asset.Model[asset.PetReferenceData]) error {
			if !zeroPosition {
				w.WriteInt8(int8(i.Slot()))
			}
			w.WriteByte(3)
			w.WriteInt(i.TemplateId())
			w.WriteBool(true)
			w.WriteLong(uint64(i.ReferenceId()))
			w.WriteInt64(msTime(time.Time{}))
			WritePaddedString(w, i.ReferenceData().Name(), 13)
			w.WriteByte(i.ReferenceData().Level())
			w.WriteShort(i.ReferenceData().Closeness())
			w.WriteByte(i.ReferenceData().Fullness())
			w.WriteInt64(msTime(i.Expiration()))
			w.WriteShort(0)   // attribute
			w.WriteShort(0)   // skill
			w.WriteInt(18000) // remaining life
			w.WriteShort(0)   // attribute
			return nil
		}
	}
}

func WriteAssetInfo(t tenant.Model) func(zeroPosition bool) func(w *response.Writer) model.Operator[asset.Model[any]] {
	return func(zeroPosition bool) func(w *response.Writer) model.Operator[asset.Model[any]] {
		return func(w *response.Writer) model.Operator[asset.Model[any]] {
			return func(a asset.Model[any]) error {
				if ad, ok := a.ReferenceData().(asset.EquipableReferenceData); ok {
					ea := asset.NewBuilder[asset.EquipableReferenceData](a.Id(), uuid.Nil, a.TemplateId(), a.ReferenceId(), a.ReferenceType()).
						SetSlot(a.Slot()).
						SetExpiration(a.Expiration()).
						SetReferenceData(ad).
						MustBuild()
					return WriteEquipableInfo(t)(zeroPosition)(w)(ea)
				}
				if ad, ok := a.ReferenceData().(asset.CashEquipableReferenceData); ok {
					ea := asset.NewBuilder[asset.CashEquipableReferenceData](a.Id(), uuid.Nil, a.TemplateId(), a.ReferenceId(), a.ReferenceType()).
						SetSlot(a.Slot()).
						SetExpiration(a.Expiration()).
						SetReferenceData(ad).
						MustBuild()
					return WriteCashEquipableInfo(t)(w, zeroPosition)(ea)
				}
				if ad, ok := a.ReferenceData().(asset.ConsumableReferenceData); ok {
					ea := asset.NewBuilder[asset.ConsumableReferenceData](a.Id(), uuid.Nil, a.TemplateId(), a.ReferenceId(), a.ReferenceType()).
						SetSlot(a.Slot()).
						SetExpiration(a.Expiration()).
						SetReferenceData(ad).
						MustBuild()
					if !zeroPosition {
						w.WriteInt8(int8(ea.Slot()))
					}
					w.WriteByte(2)
					w.WriteInt(ea.TemplateId())
					w.WriteBool(false)
					w.WriteInt64(msTime(ea.Expiration()))
					w.WriteShort(uint16(ea.Quantity()))
					w.WriteAsciiString("") // TODO
					w.WriteShort(ea.ReferenceData().Flag())
					if item.IsBullet(item.Id(ea.TemplateId())) || item.IsThrowingStar(item.Id(ea.TemplateId())) {
						w.WriteLong(ea.ReferenceData().Rechargeable())
					}
					return nil
				}
				if ad, ok := a.ReferenceData().(asset.SetupReferenceData); ok {
					ea := asset.NewBuilder[asset.SetupReferenceData](a.Id(), uuid.Nil, a.TemplateId(), a.ReferenceId(), a.ReferenceType()).
						SetSlot(a.Slot()).
						SetExpiration(a.Expiration()).
						SetReferenceData(ad).
						MustBuild()
					if !zeroPosition {
						w.WriteInt8(int8(ea.Slot()))
					}
					w.WriteByte(2)
					w.WriteInt(ea.TemplateId())
					w.WriteBool(false)
					w.WriteInt64(msTime(ea.Expiration()))
					w.WriteShort(uint16(ea.Quantity()))
					w.WriteAsciiString("") // TODO
					w.WriteShort(ea.ReferenceData().Flag())
					return nil
				}
				if ad, ok := a.ReferenceData().(asset.EtcReferenceData); ok {
					ea := asset.NewBuilder[asset.EtcReferenceData](a.Id(), uuid.Nil, a.TemplateId(), a.ReferenceId(), a.ReferenceType()).
						SetSlot(a.Slot()).
						SetExpiration(a.Expiration()).
						SetReferenceData(ad).
						MustBuild()
					if !zeroPosition {
						w.WriteInt8(int8(ea.Slot()))
					}
					w.WriteByte(2)
					w.WriteInt(ea.TemplateId())
					w.WriteBool(false)
					w.WriteInt64(msTime(ea.Expiration()))
					w.WriteShort(uint16(ea.Quantity()))
					w.WriteAsciiString("") // TODO
					w.WriteShort(ea.ReferenceData().Flag())
					return nil
				}
				if ad, ok := a.ReferenceData().(asset.CashReferenceData); ok {
					ea := asset.NewBuilder[asset.CashReferenceData](a.Id(), uuid.Nil, a.TemplateId(), a.ReferenceId(), a.ReferenceType()).
						SetSlot(a.Slot()).
						SetExpiration(a.Expiration()).
						SetReferenceData(ad).
						MustBuild()
					return WriteCashItemInfo(zeroPosition)(w)(ea)
				}
				if ad, ok := a.ReferenceData().(asset.PetReferenceData); ok {
					ea := asset.NewBuilder[asset.PetReferenceData](a.Id(), uuid.Nil, a.TemplateId(), a.ReferenceId(), a.ReferenceType()).
						SetSlot(a.Slot()).
						SetExpiration(a.Expiration()).
						SetReferenceData(ad).
						MustBuild()
					return WritePetCashItemInfo(zeroPosition)(w)(ea)
				}
				return errors.New("unknown item type")
			}
		}
	}
}

func WriteEquipableIfPresent(tenant tenant.Model) func(w *response.Writer, model slot2.Model) {
	return func(w *response.Writer, model slot2.Model) {
		if model.Equipable != nil {
			_ = WriteEquipableInfo(tenant)(false)(w)(*model.Equipable)
		}
	}
}

func WriteEquipableInfo(tenant tenant.Model) func(zeroPosition bool) func(w *response.Writer) model.Operator[asset.Model[asset.EquipableReferenceData]] {
	return func(zeroPosition bool) func(w *response.Writer) model.Operator[asset.Model[asset.EquipableReferenceData]] {
		return func(w *response.Writer) model.Operator[asset.Model[asset.EquipableReferenceData]] {
			return func(e asset.Model[asset.EquipableReferenceData]) error {
				slot := e.Slot()
				if !zeroPosition {
					slot = int16(math.Abs(float64(slot)))
					if slot > 100 {
						slot -= 100
					}
					if (tenant.Region() == "GMS" && tenant.MajorVersion() > 28) || tenant.Region() == "JMS" {
						w.WriteShort(uint16(slot))
					} else {
						w.WriteByte(byte(slot))
					}
				}

				if (tenant.Region() == "GMS" && tenant.MajorVersion() > 12) || tenant.Region() == "JMS" {
					w.WriteByte(1)
				}
				w.WriteInt(e.TemplateId())
				w.WriteBool(false)
				w.WriteInt64(msTime(e.Expiration()))
				w.WriteByte(byte(e.ReferenceData().Slots()))
				w.WriteByte(e.ReferenceData().Level())
				if tenant.Region() == "JMS" {
					w.WriteByte(0)
				}
				w.WriteShort(e.ReferenceData().Strength())
				w.WriteShort(e.ReferenceData().Dexterity())
				w.WriteShort(e.ReferenceData().Intelligence())
				w.WriteShort(e.ReferenceData().Luck())
				w.WriteShort(e.ReferenceData().HP())
				w.WriteShort(e.ReferenceData().MP())
				w.WriteShort(e.ReferenceData().WeaponAttack())
				w.WriteShort(e.ReferenceData().MagicAttack())
				w.WriteShort(e.ReferenceData().WeaponDefense())
				w.WriteShort(e.ReferenceData().MagicDefense())
				w.WriteShort(e.ReferenceData().Accuracy())
				w.WriteShort(e.ReferenceData().Avoidability())
				w.WriteShort(e.ReferenceData().Hands())
				w.WriteShort(e.ReferenceData().Speed())
				w.WriteShort(e.ReferenceData().Jump())

				if (tenant.Region() == "GMS" && tenant.MajorVersion() > 12) || tenant.Region() == "JMS" {
					w.WriteAsciiString("") // TODO retrieve owner name from id
					w.WriteShort(0)        // TODO
				}

				if (tenant.Region() == "GMS" && tenant.MajorVersion() > 28) || tenant.Region() == "JMS" {
					w.WriteByte(e.ReferenceData().LevelType())
					w.WriteByte(e.ReferenceData().Level())
					w.WriteInt(e.ReferenceData().Experience())
					w.WriteInt(e.ReferenceData().HammersApplied())

					if tenant.Region() == "JMS" {
						w.WriteByte(0)
						w.WriteShort(0)
						w.WriteShort(0)
						w.WriteShort(0)
						w.WriteShort(0)
						w.WriteShort(0)
						w.WriteInt(0)
					}

					w.WriteLong(0)
					w.WriteLong(uint64(getTime(-2)))
					w.WriteInt32(-1)
				}
				return nil
			}
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
