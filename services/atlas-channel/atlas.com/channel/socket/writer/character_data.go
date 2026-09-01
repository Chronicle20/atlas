package writer

import (
	"atlas-channel/buddylist"
	"atlas-channel/character"
	"atlas-channel/character/equipslot"
	"atlas-channel/character/teleportrock"
	"atlas-channel/quest"
	"atlas-channel/ring"
	model2 "atlas-channel/socket/model"
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/degrade"
)

func BuildCharacterData(l logrus.FieldLogger, ctx context.Context, c character.Model, bl buddylist.Model, mapId _map.Id, trm teleportrock.Model) charpkt.CharacterData {
	cd := charpkt.CharacterData{
		Stats: charpkt.CharacterStats{
			Id:         c.Id(),
			Name:       c.Name(),
			Gender:     c.Gender(),
			SkinColor:  c.SkinColor(),
			Face:       c.Face(),
			Hair:       c.Hair(),
			Level:      c.Level(),
			JobId:      uint16(c.JobId()),
			Str:        c.Strength(),
			Dex:        c.Dexterity(),
			Int:        c.Intelligence(),
			Luk:        c.Luck(),
			Hp:         c.Hp(),
			MaxHp:      c.MaxHp(),
			Mp:         c.Mp(),
			MaxMp:      c.MaxMp(),
			Ap:         c.Ap(),
			Sp:         c.RemainingSp(),
			Exp:        c.Experience(),
			Fame:       c.Fame(),
			GachaExp:   c.GachaponExperience(),
			MapId:      uint32(mapId),
			SpawnPoint: byte(c.SpawnPoint()),
		},
		BuddyCapacity: bl.Capacity(),
		Meso:          c.Meso(),
	}

	// Saved teleport-rock lists (FR-15/16). Codec pads to 5/10 with EmptyMapId.
	cd.TeleportMaps = trm.Regular()
	cd.VipTeleportMaps = trm.Vip()

	// Couple/friendship ring record block (site A). Cache-only: population
	// happens once at character load (PRD §8), not here.
	cd.Rings = ring.NewProcessor(l, ctx).GetRingRecords(c.Id())

	// Pet IDs
	for i, p := range c.Pets() {
		if i < 3 {
			cd.Stats.PetIds[i] = p.CashId()
		}
	}

	// Inventory
	cd.Inventory = buildInventoryData(l, ctx, c)

	// Skills
	for _, s := range c.Skills() {
		// MasterLevel is always populated; whether it reaches the wire is the
		// codec's call (charpkt derives it per SKILL from the id via
		// job.NeedsMasterLevel, which resolves the arm from the tenant's
		// region and major version). Gating it here on the per-JOB IsFourthJob
		// is what closed the client with error 38 on a preset Evan: the client
		// asks is_skill_need_master_level(nSkillID), which for Evan selects
		// only the 9th/10th growths plus three named skills — not the whole
		// 2214-2218 band. See job.NeedsMasterLevel (task-218, task-275).
		entry := charpkt.SkillEntry{
			Id:          uint32(s.Id()),
			Level:       uint32(s.Level()),
			Expiration:  packetmodel.MsTime(s.Expiration()),
			MasterLevel: uint32(s.MasterLevel()),
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

	// Monster book: cover + owned cards (empty when the decorator failed open).
	mb := c.MonsterBook()
	cd.MonsterBook.CoverCardId = mb.CoverCardId()
	for _, mc := range mb.Cards() {
		cd.MonsterBook.Cards = append(cd.MonsterBook.Cards, charpkt.MonsterBookCard{
			CardId: mc.CardId(),
			Level:  mc.Level(),
		})
	}

	return cd
}

func buildInventoryData(l logrus.FieldLogger, ctx context.Context, c character.Model) charpkt.InventoryData {
	exts, err := equipslot.NewProcessor(l, ctx).GetActive(c.Id())
	if err != nil {
		// Fail-open: a missing/unreachable equip-slot-extensions read must
		// never block SET_FIELD, mirroring the teleport-rock fail-open in
		// SetFieldBody (design §4.4).
		degrade.Observe(l, "channel.character_data.equip_slot_ext", c.Id(), err)
		exts = nil
	}
	equipSlotExtExpire := equipSlotExtExpireFor(exts)

	inv := charpkt.InventoryData{
		EquipCapacity:      byte(c.Inventory().Equipable().Capacity()),
		UseCapacity:        byte(c.Inventory().Consumable().Capacity()),
		SetupCapacity:      byte(c.Inventory().Setup().Capacity()),
		EtcCapacity:        byte(c.Inventory().ETC().Capacity()),
		CashCapacity:       byte(c.Inventory().Cash().Capacity()),
		EquipSlotExtExpire: equipSlotExtExpire,
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

// equipSlotExtExpireFor derives the SET_FIELD/InventoryData EquipSlotExtExpire
// FILETIME value from a character's active equip-slot extensions (task-240
// task 23, R3/R4). No active extension keeps ZeroTime -- the correct
// placeholder for the common case, not an error. An active one converts its
// ExpiresAt via packetmodel.MsTime, the same FILETIME formula
// (t.Unix()*10_000_000 + 116444736000000000) every other timestamp field in
// this codec already applies (libs/atlas-packet/model/ms_time.go). Only the
// first entry is used -- pendant2 is currently the only extendable slot, so
// "active extensions" is a one-element list in practice.
func equipSlotExtExpireFor(exts []equipslot.RestModel) int64 {
	if len(exts) == 0 {
		return ZeroTime
	}
	return packetmodel.MsTime(exts[0].ExpiresAt)
}
