package writer

import (
	"atlas-channel/character/skill"
	"atlas-channel/npc/shops/commodities"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	npcpkt "github.com/Chronicle20/atlas/libs/atlas-packet/npc/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

// NPCShopBody encodes the NPC shop packet. addSlotMax's Throwing-Star/Bullet
// masteries are resolved through set (task-187): the caller resolves the
// tenant's version once (constants.For(...).Skill) and passes it in, since
// this pure packet-encode function has no ctx of its own to resolve from.
func NPCShopBody(templateId uint32, cs []commodities.Model, skills []skill.Model, set skill2.Set) packet.Encode {
	sc := make([]npcpkt.ShopCommodity, len(cs))
	for i, c := range cs {
		isAmmo := item.IsBullet(item.Id(c.TemplateId())) || item.IsThrowingStar(item.Id(c.TemplateId()))

		addSlotMax := uint16(0)
		if item.IsThrowingStar(item.Id(c.TemplateId())) {
			addSlotMax += uint16(skill.GetLevelIdentity(skills, set, skill2.NightWalkerStage2ClawMastery)) * 10
			addSlotMax += uint16(skill.GetLevelIdentity(skills, set, skill2.AssassinClawMastery)) * 10
		}
		if item.IsBullet(item.Id(c.TemplateId())) {
			addSlotMax += uint16(skill.GetLevelIdentity(skills, set, skill2.GunslingerGunMastery)) * 10
		}

		sc[i] = npcpkt.ShopCommodity{
			TemplateId:      c.TemplateId(),
			MesoPrice:       c.MesoPrice(),
			DiscountRate:    c.DiscountRate(),
			TokenTemplateId: c.TokenTemplateId(),
			TokenPrice:      c.TokenPrice(),
			Period:          c.Period(),
			LevelLimit:      c.LevelLimit(),
			IsAmmo:          isAmmo,
			Quantity:        c.Quantity(),
			UnitPrice:       c.UnitPrice(),
			SlotMax:         uint16(c.SlotMax()) + addSlotMax,
		}
	}
	return npcpkt.NewNPCShop(templateId, sc).Encode
}
