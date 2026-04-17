package writer

import (
	"atlas-channel/character/skill"
	"atlas-channel/npc/shops/commodities"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	npcpkt "github.com/Chronicle20/atlas/libs/atlas-packet/npc/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)


func NPCShopBody(templateId uint32, cs []commodities.Model, skills []skill.Model) packet.Encode {
	sc := make([]npcpkt.ShopCommodity, len(cs))
	for i, c := range cs {
		isAmmo := item.IsBullet(item.Id(c.TemplateId())) || item.IsThrowingStar(item.Id(c.TemplateId()))

		addSlotMax := uint16(0)
		if item.IsThrowingStar(item.Id(c.TemplateId())) {
			addSlotMax += uint16(skill.GetLevel(skills, skill2.NightWalkerStage2ClawMasteryId)) * 10
			addSlotMax += uint16(skill.GetLevel(skills, skill2.AssassinClawMasteryId)) * 10
		}
		if item.IsBullet(item.Id(c.TemplateId())) {
			addSlotMax += uint16(skill.GetLevel(skills, skill2.GunslingerGunMasteryId)) * 10
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
