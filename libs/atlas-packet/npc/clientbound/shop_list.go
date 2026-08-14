package clientbound

import (
	"context"
	"fmt"
	"math"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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

// packet-audit:fname CShopDlg::SetShopDlg
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
			// v92 CShopDlg::SetShopDlg@0x6df6c0 reads FOUR Decode4 calls between the
			// discount-rate byte and the ammo/quantity branch (v97/v98/v99/v100 in the
			// decompile), one more than v87's THREE (nTokenItemID/nItemPeriod/
			// nLevelLimited @0x79e558-0x79e565) — v92 already carries tokenTemplateId,
			// v87 does not. v87 and v92 are adjacent columns in the coverage matrix
			// (no IDB exists for the intervening 88-91 range), so >=92 is the precise,
			// evidence-supported boundary; the >=95 boundary this gate previously used
			// was never verified against a v88-94 IDB. task-221 task-16.
			if t.Region() == "GMS" && t.MajorVersion() >= 92 {
				w.WriteInt(c.TokenTemplateId)
			}
			// GMS v79 CShopDlg::SetShopDlg@0x6d3459 reads only itemId, mesoPrice,
			// then the quantity/unitPrice branch and maxPerSlot short — the
			// tokenPrice/period/levelLimit ints were added after v79. delta §3.2
			if !(t.Region() == "GMS" && t.MajorVersion() < 83) {
				w.WriteInt(c.TokenPrice)
				w.WriteInt(c.Period)
				w.WriteInt(c.LevelLimit)
			}
			// v48 (pre-v61) CShopDlg::SetShopDlg sub_5B430A@0x5b430a
			// (GMS_v48_1_DEVM.exe port 13337) reads per item Decode4 itemId,
			// Decode4 mesoPrice, then — only for rechargeable/ammo
			// (itemId/10000==207) — DecodeBuffer(8) unitPrice, and ALWAYS Decode2
			// quantity; there is NO trailing maxPerSlot short (added between v48
			// and v61: v61 SetShopDlg@0x6437e3 / v79 @0x6d3459 both read it). So
			// v48 emits [unitPrice(8) if ammo] + quantity(2), no slotMax.
			// task-113 v48 Stage E.
			if t.Region() == "GMS" && t.MajorVersion() < 61 {
				if c.IsAmmo {
					w.WriteLong(math.Float64bits(c.UnitPrice))
				}
				w.WriteShort(c.Quantity)
				continue
			}
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
			// See the matching Encode-side comment: v92 verified to already carry
			// tokenTemplateId (v87 verified not to); >=92 is the evidence-supported
			// boundary, not the previous unverified >=95.
			if t.Region() == "GMS" && t.MajorVersion() >= 92 {
				m.commodities[i].TokenTemplateId = r.ReadUint32()
			}
			// GMS v79 omits tokenPrice/period/levelLimit (SetShopDlg@0x6d3459).
			if !(t.Region() == "GMS" && t.MajorVersion() < 83) {
				m.commodities[i].TokenPrice = r.ReadUint32()
				m.commodities[i].Period = r.ReadUint32()
				m.commodities[i].LevelLimit = r.ReadUint32()
			}
			if t.Region() == "GMS" && t.MajorVersion() < 61 {
				// v48: [unitPrice(8) if ammo] + quantity(2), no slotMax.
				if m.commodities[i].IsAmmo {
					m.commodities[i].UnitPrice = math.Float64frombits(r.ReadUint64())
				}
				m.commodities[i].Quantity = r.ReadUint16()
				continue
			}
			if !m.commodities[i].IsAmmo {
				m.commodities[i].Quantity = r.ReadUint16()
			} else {
				m.commodities[i].UnitPrice = math.Float64frombits(r.ReadUint64())
			}
			m.commodities[i].SlotMax = r.ReadUint16()
		}
	}
}
