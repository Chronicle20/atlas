package writer

import (
	"atlas-channel/npc/shops/commodities"
	"github.com/Chronicle20/atlas-constants/item"
	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"math"
)

const NPCShop = "NPCShop"

func NPCShopBody(_ logrus.FieldLogger, t tenant.Model) func(templateId uint32, commodities []commodities.Model) BodyProducer {
	return func(templateId uint32, commodities []commodities.Model) BodyProducer {
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
				w.WriteShort(uint16(c.SlotMax()))
			}
			return w.Bytes()
		}
	}
}
