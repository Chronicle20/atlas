package writer

import (
	"atlas-channel/buddylist"
	"atlas-channel/character"
	"atlas-channel/quest"
	model2 "atlas-channel/socket/model"
	"time"

	"github.com/Chronicle20/atlas-constants/inventory/slot"
	charpkt "github.com/Chronicle20/atlas-packet/character"
	packetmodel "github.com/Chronicle20/atlas-packet/model"
)

func BuildCharacterData(c character.Model, bl buddylist.Model) charpkt.CharacterData {
	cd := charpkt.CharacterData{
		Stats: charpkt.CharacterStats{
			Id:        c.Id(),
			Name:      c.Name(),
			Gender:    c.Gender(),
			SkinColor: c.SkinColor(),
			Face:      c.Face(),
			Hair:      c.Hair(),
			Level:     c.Level(),
			JobId:     uint16(c.JobId()),
			Str:       c.Strength(),
			Dex:       c.Dexterity(),
			Int:       c.Intelligence(),
			Luk:       c.Luck(),
			Hp:        c.Hp(),
			MaxHp:     c.MaxHp(),
			Mp:        c.Mp(),
			MaxMp:     c.MaxMp(),
			Ap:        c.Ap(),
			Sp:        c.RemainingSp(),
			Exp:       c.Experience(),
			Fame:      c.Fame(),
			GachaExp:  c.GachaponExperience(),
			MapId:     uint32(c.MapId()),
			SpawnPoint: c.SpawnPoint(),
		},
		BuddyCapacity: bl.Capacity(),
		Meso:          c.Meso(),
	}

	// Pet IDs
	for i, p := range c.Pets() {
		if i < 3 {
			cd.Stats.PetIds[i] = p.CashId()
		}
	}

	// Inventory
	cd.Inventory = buildInventoryData(c)

	// Skills
	for _, s := range c.Skills() {
		entry := charpkt.SkillEntry{
			Id:         uint32(s.Id()),
			Level:      uint32(s.Level()),
			Expiration: packetmodel.MsTime(s.Expiration()),
			FourthJob:  s.IsFourthJob(),
		}
		if s.IsFourthJob() {
			entry.MasterLevel = uint32(s.MasterLevel())
		}
		cd.Skills = append(cd.Skills, entry)

		if s.OnCooldown() {
			remaining := uint16(s.CooldownExpiresAt().Sub(time.Now()).Seconds())
			cd.Cooldowns = append(cd.Cooldowns, charpkt.CooldownEntry{
				SkillId:   uint32(s.Id()),
				Remaining: remaining,
			})
		}
	}

	// Quests
	for _, q := range quest.Started(c.Quests()) {
		cd.StartedQuests = append(cd.StartedQuests, charpkt.QuestProgress{
			QuestId:  uint16(q.QuestId()),
			Progress: q.ProgressString(),
		})
	}
	for _, q := range quest.Completed(c.Quests()) {
		cd.CompletedQuests = append(cd.CompletedQuests, charpkt.QuestCompleted{
			QuestId:     uint16(q.QuestId()),
			CompletedAt: packetmodel.MsTime(q.CompletedAt()),
		})
	}

	return cd
}

func buildInventoryData(c character.Model) charpkt.InventoryData {
	inv := charpkt.InventoryData{
		EquipCapacity: byte(c.Inventory().Equipable().Capacity()),
		UseCapacity:   byte(c.Inventory().Consumable().Capacity()),
		SetupCapacity: byte(c.Inventory().Setup().Capacity()),
		EtcCapacity:   byte(c.Inventory().ETC().Capacity()),
		CashCapacity:  byte(c.Inventory().Cash().Capacity()),
		Timestamp:     ZeroTime,
	}

	// Regular equipment and cash equipment from equipment slots
	for _, t := range slot.Slots {
		if s, ok := c.Equipment().Get(t.Type); ok {
			if s.Equipable != nil {
				inv.RegularEquip = append(inv.RegularEquip, model2.NewAsset(false, *s.Equipable))
			}
			if s.CashEquipable != nil {
				inv.CashEquip = append(inv.CashEquip, model2.NewAsset(false, *s.CashEquipable))
			}
		}
	}

	// Equipable inventory (slot > 0)
	for _, a := range c.Inventory().Equipable().Assets() {
		inv.EquipInv = append(inv.EquipInv, model2.NewAsset(false, a))
	}

	// Use inventory
	for _, a := range c.Inventory().Consumable().Assets() {
		inv.UseInv = append(inv.UseInv, model2.NewAsset(false, a))
	}

	// Setup inventory
	for _, a := range c.Inventory().Setup().Assets() {
		inv.SetupInv = append(inv.SetupInv, model2.NewAsset(false, a))
	}

	// Etc inventory
	for _, a := range c.Inventory().ETC().Assets() {
		inv.EtcInv = append(inv.EtcInv, model2.NewAsset(false, a))
	}

	// Cash inventory
	for _, a := range c.Inventory().Cash().Assets() {
		inv.CashInv = append(inv.CashInv, model2.NewAsset(false, a))
	}

	return inv
}
