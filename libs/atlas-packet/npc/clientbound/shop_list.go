package clientbound

import (
	"context"
	"fmt"
	"math"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const NPCShopWriter = "NPCShop"

type ShopCommodity struct {
	TemplateId      uint32
	MesoPrice       uint32
	DiscountRate    byte
	TokenTemplateId uint32
	TokenPrice      uint32
	Period          uint32
	LevelLimit      uint32
	IsAmmo          bool
	Quantity        uint16
	UnitPrice       float64
	SlotMax         uint16
}

type ShopList struct {
	templateId  uint32
	commodities []ShopCommodity
}

func NewNPCShop(templateId uint32, commodities []ShopCommodity) ShopList {
	return ShopList{templateId: templateId, commodities: commodities}
}

func (m ShopList) Operation() string { return NPCShopWriter }
func (m ShopList) String() string {
	return fmt.Sprintf("npc shop templateId [%d] commodities [%d]", m.templateId, len(m.commodities))
}

func (m ShopList) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.templateId)
		w.WriteShort(uint16(len(m.commodities)))
		for _, c := range m.commodities {
			w.WriteInt(c.TemplateId)
			w.WriteInt(c.MesoPrice)
			if t.Region() == "GMS" && t.MajorVersion() >= 87 {
				w.WriteByte(c.DiscountRate)
			}
			if t.Region() == "GMS" && t.MajorVersion() >= 95 {
				w.WriteInt(c.TokenTemplateId)
			}
			w.WriteInt(c.TokenPrice)
			w.WriteInt(c.Period)
			w.WriteInt(c.LevelLimit)
			if !c.IsAmmo {
				w.WriteShort(c.Quantity)
			} else {
				w.WriteLong(math.Float64bits(c.UnitPrice))
			}
			w.WriteShort(c.SlotMax)
		}
		return w.Bytes()
	}
}

func (m *ShopList) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		t := tenant.MustFromContext(ctx)
		m.templateId = r.ReadUint32()
		count := r.ReadUint16()
		m.commodities = make([]ShopCommodity, count)
		for i := range m.commodities {
			m.commodities[i].TemplateId = r.ReadUint32()
			m.commodities[i].MesoPrice = r.ReadUint32()
			if t.Region() == "GMS" && t.MajorVersion() >= 87 {
				m.commodities[i].DiscountRate = r.ReadByte()
			}
			if t.Region() == "GMS" && t.MajorVersion() >= 95 {
				m.commodities[i].TokenTemplateId = r.ReadUint32()
			}
			m.commodities[i].TokenPrice = r.ReadUint32()
			m.commodities[i].Period = r.ReadUint32()
			m.commodities[i].LevelLimit = r.ReadUint32()
			if !m.commodities[i].IsAmmo {
				m.commodities[i].Quantity = r.ReadUint16()
			} else {
				m.commodities[i].UnitPrice = math.Float64frombits(r.ReadUint64())
			}
			m.commodities[i].SlotMax = r.ReadUint16()
		}
	}
}
