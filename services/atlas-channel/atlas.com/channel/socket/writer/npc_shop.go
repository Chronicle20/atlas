package writer

import (
	"atlas-channel/character/skill"
	"atlas-channel/npc/shops/commodities"
	"github.com/Chronicle20/atlas-constants/item"
	skill2 "github.com/Chronicle20/atlas-constants/skill"
	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"math"
)

const NPCShop = "NPCShop"

func NPCShopBody(_ logrus.FieldLogger, t tenant.Model) func(templateId uint32, commodities []commodities.Model, skills []skill.Model) BodyProducer {
	return func(templateId uint32, commodities []commodities.Model, skills []skill.Model) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(templateId)
			w.WriteShort(uint16(len(commodities)))
			for _, c := range commodities {
				w.WriteInt(c.TemplateId())
				w.WriteInt(c.MesoPrice())
				if t.Region() == "GMS" && t.MajorVersion() >= 87 {
					w.WriteByte(c.DiscountRate())
				}
				if t.Region() == "GMS" && t.MajorVersion() >= 95 {
					w.WriteInt(c.TokenItemId())
				}
				w.WriteInt(c.TokenPrice())
				// Changes confirmation prompt to indicate a time-restricted item.
				w.WriteInt(c.Period())
				w.WriteInt(c.LevelLimit())
				if !item.IsBullet(item.Id(c.TemplateId())) && !item.IsThrowingStar(item.Id(c.TemplateId())) {
					w.WriteShort(c.Quantity())
				} else {
					w.WriteLong(math.Float64bits(c.UnitPrice()))
				}

				addSlotMax := uint16(0)
				if item.IsThrowingStar(item.Id(c.TemplateId())) {
					addSlotMax += uint16(skill.GetLevel(skills, skill2.NightWalkerStage2ClawMasteryId)) * 10
					addSlotMax += uint16(skill.GetLevel(skills, skill2.AssassinClawMasteryId)) * 10
				}
				if item.IsBullet(item.Id(c.TemplateId())) {
					addSlotMax += uint16(skill.GetLevel(skills, skill2.GunslingerGunMasteryId)) * 10
				}
				w.WriteShort(uint16(c.SlotMax()) + addSlotMax)
			}
			return w.Bytes()
		}
	}
}
