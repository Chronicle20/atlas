package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Counter-arm family (task-183 Wave 1.2). RE-proven shape: mode + uint16
// absolute-counter update — NO separate inventory/slot-type byte (contra
// InventoryCapacitySuccess's mode+type+uint16 shape). See
// docs/tasks/task-183-cashshop-result-family/arm-catalog.md rows
// INC_TRUNK_COUNT_SUCCESS / INC_CHARACTER_SLOT_COUNT_SUCCESS /
// INC_BUY_CHARACTER_COUNT_SUCCESS. ENABLE_EQUIP_SLOT_EXT_SUCCESS is the lone
// exception in this file: mode + TWO uint16 fields (slotIndex, days).

// IncTrunkCountSuccess - INC_TRUNK_COUNT_SUCCESS arm (CCashShop::OnCashItemResIncTrunkCountDone):
// mode + trunkCount:Decode2 (uint16, new absolute m_nTrunkCount). Discrete per-mode
// struct: it fixes the INC_TRUNK_COUNT_SUCCESS operation key (the body func resolves
// it); never accepts the mode from the caller.
// packet-audit:fname CCashShop::OnCashItemResult#INC_TRUNK_COUNT_SUCCESS
type IncTrunkCountSuccess struct {
	mode       byte
	trunkCount uint16
}

func NewIncTrunkCountSuccess(mode byte, trunkCount uint16) IncTrunkCountSuccess {
	return IncTrunkCountSuccess{mode: mode, trunkCount: trunkCount}
}

func (m IncTrunkCountSuccess) Mode() byte         { return m.mode }
func (m IncTrunkCountSuccess) TrunkCount() uint16 { return m.trunkCount }
func (m IncTrunkCountSuccess) Operation() string  { return CashShopOperationWriter }

func (m IncTrunkCountSuccess) String() string {
	return fmt.Sprintf("cash inc-trunk-count success mode [%d] trunkCount [%d]", m.mode, m.trunkCount)
}

func (m IncTrunkCountSuccess) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteShort(m.trunkCount)
		return w.Bytes()
	}
}

func (m *IncTrunkCountSuccess) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.trunkCount = r.ReadUint16()
	}
}

// IncCharacterSlotCountSuccess - INC_CHARACTER_SLOT_COUNT_SUCCESS arm
// (CCashShop::OnCashItemResIncCharacterSlotCountDone): mode + slotCount:Decode2
// (uint16, new absolute m_nCharacterSlotCount). Discrete per-mode struct: fixes
// the INC_CHARACTER_SLOT_COUNT_SUCCESS operation key; never accepts the mode from
// the caller. Wire-identical shape to IncTrunkCountSuccess but a distinct mode arm.
// packet-audit:fname CCashShop::OnCashItemResult#INC_CHARACTER_SLOT_COUNT_SUCCESS
type IncCharacterSlotCountSuccess struct {
	mode      byte
	slotCount uint16
}

func NewIncCharacterSlotCountSuccess(mode byte, slotCount uint16) IncCharacterSlotCountSuccess {
	return IncCharacterSlotCountSuccess{mode: mode, slotCount: slotCount}
}

func (m IncCharacterSlotCountSuccess) Mode() byte        { return m.mode }
func (m IncCharacterSlotCountSuccess) SlotCount() uint16 { return m.slotCount }
func (m IncCharacterSlotCountSuccess) Operation() string { return CashShopOperationWriter }

func (m IncCharacterSlotCountSuccess) String() string {
	return fmt.Sprintf("cash inc-character-slot-count success mode [%d] slotCount [%d]", m.mode, m.slotCount)
}

func (m IncCharacterSlotCountSuccess) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteShort(m.slotCount)
		return w.Bytes()
	}
}

func (m *IncCharacterSlotCountSuccess) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.slotCount = r.ReadUint16()
	}
}

// IncBuyCharacterCountSuccess - INC_BUY_CHARACTER_COUNT_SUCCESS arm
// (CCashShop::OnCashItemResIncBuyCharacterCountDone): mode +
// buyCharacterCount:Decode2 (uint16, new absolute m_nBuyCharacterCount) on
// every MODERN version (v95/jms verified). Discrete per-mode struct: fixes
// the INC_BUY_CHARACTER_COUNT_SUCCESS operation key; never accepts the mode
// from the caller. n-a in v83/v84/v87 per the catalog.
//
// v72 (legacy, GMS only) is a MATERIALLY DIFFERENT wire shape, decompiled at
// CCashShop::OnCashItemResIncBuyCharacterCountDone@0x472967 (task-183 Wave 3
// batch MISC-L, see .superpowers/sdd/task-3.4-legacy-misc-report.md
// "SPECIAL" section): mode + slotIndex:Decode2 (uint16) +
// item:GW_ItemSlotBase::Decode (full polymorphic item-slot struct — a
// "buy a character slot by consuming one specific locker item" operation,
// not a bare absolute-counter update). This is region+version-gated
// (incBuyCharacterCountSuccessIsV72Shape), code-only — it CANNOT be
// registered as a gates.yaml row because this struct has no matrix row of
// its own (it's a dispatcher-family arm; the nxCashSpent precedent in
// shop_operation_result_gift.go established that dispatcher-family arms
// don't get gates.yaml entries). The gate mirrors CashItemMovedToInventory's
// slot+asset shape exactly (shop_item_moved.go).
// packet-audit:fname CCashShop::OnCashItemResult#INC_BUY_CHARACTER_COUNT_SUCCESS
type IncBuyCharacterCountSuccess struct {
	mode              byte
	buyCharacterCount uint16
	slotIndex         uint16
	asset             packetmodel.Asset
}

// NewIncBuyCharacterCountSuccess constructs the MODERN (v95/jms) shape:
// mode + bare uint16 absolute counter. Do not use this for v72.
func NewIncBuyCharacterCountSuccess(mode byte, buyCharacterCount uint16) IncBuyCharacterCountSuccess {
	return IncBuyCharacterCountSuccess{mode: mode, buyCharacterCount: buyCharacterCount}
}

// NewIncBuyCharacterCountSuccessV72 constructs the v72 LEGACY shape: mode +
// slotIndex + a full item-slot asset (the locker item being consumed).
func NewIncBuyCharacterCountSuccessV72(mode byte, slotIndex uint16, asset packetmodel.Asset) IncBuyCharacterCountSuccess {
	return IncBuyCharacterCountSuccess{mode: mode, slotIndex: slotIndex, asset: asset}
}

func (m IncBuyCharacterCountSuccess) Mode() byte                { return m.mode }
func (m IncBuyCharacterCountSuccess) BuyCharacterCount() uint16 { return m.buyCharacterCount }
func (m IncBuyCharacterCountSuccess) SlotIndex() uint16         { return m.slotIndex }
func (m IncBuyCharacterCountSuccess) Asset() packetmodel.Asset  { return m.asset }
func (m IncBuyCharacterCountSuccess) Operation() string         { return CashShopOperationWriter }

func (m IncBuyCharacterCountSuccess) String() string {
	return fmt.Sprintf("cash inc-buy-character-count success mode [%d] buyCharacterCount [%d] slotIndex [%d]", m.mode, m.buyCharacterCount, m.slotIndex)
}

// incBuyCharacterCountSuccessIsV72Shape reports whether the v72 legacy shape
// (slotIndex + item-slot asset) applies, versus the MODERN bare-counter
// shape used by every other present version (v95/jms). GMS/v72 exact-match,
// not MajorAtLeast — this is a one-version divergence, not a floor.
func incBuyCharacterCountSuccessIsV72Shape(t tenant.Model) bool {
	return t.Region() == "GMS" && t.MajorVersion() == 72
}

func (m IncBuyCharacterCountSuccess) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		if incBuyCharacterCountSuccessIsV72Shape(t) {
			w.WriteShort(m.slotIndex)
			w.WriteByteArray(m.asset.Encode(l, ctx)(options))
		} else {
			w.WriteShort(m.buyCharacterCount)
		}
		return w.Bytes()
	}
}

func (m *IncBuyCharacterCountSuccess) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		if incBuyCharacterCountSuccessIsV72Shape(t) {
			m.slotIndex = r.ReadUint16()
			m.asset.Decode(l, ctx)(r, options)
		} else {
			m.buyCharacterCount = r.ReadUint16()
		}
	}
}

// EnableEquipSlotExtSuccess - ENABLE_EQUIP_SLOT_EXT_SUCCESS arm
// (CCashShop::OnCashItemResEnableEquipSlotExtDone): mode + slotIndex:Decode2
// (uint16, indexes aEquipExtExpire[v3]/aEquipped2[v3+59+13]) +
// days:Decode2 (uint16, passed to Util::FTAddDay as day-count). TWO uint16
// fields, NOT one count and NOT byte+short — distinct shape from the other
// three counter arms in this file. Discrete per-mode struct: fixes the
// ENABLE_EQUIP_SLOT_EXT_SUCCESS operation key; never accepts the mode from
// the caller.
// packet-audit:fname CCashShop::OnCashItemResult#ENABLE_EQUIP_SLOT_EXT_SUCCESS
type EnableEquipSlotExtSuccess struct {
	mode      byte
	slotIndex uint16
	days      uint16
}

func NewEnableEquipSlotExtSuccess(mode byte, slotIndex uint16, days uint16) EnableEquipSlotExtSuccess {
	return EnableEquipSlotExtSuccess{mode: mode, slotIndex: slotIndex, days: days}
}

func (m EnableEquipSlotExtSuccess) Mode() byte        { return m.mode }
func (m EnableEquipSlotExtSuccess) SlotIndex() uint16 { return m.slotIndex }
func (m EnableEquipSlotExtSuccess) Days() uint16      { return m.days }
func (m EnableEquipSlotExtSuccess) Operation() string { return CashShopOperationWriter }

func (m EnableEquipSlotExtSuccess) String() string {
	return fmt.Sprintf("cash enable-equip-slot-ext success mode [%d] slotIndex [%d] days [%d]", m.mode, m.slotIndex, m.days)
}

func (m EnableEquipSlotExtSuccess) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteShort(m.slotIndex)
		w.WriteShort(m.days)
		return w.Bytes()
	}
}

func (m *EnableEquipSlotExtSuccess) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.slotIndex = r.ReadUint16()
		m.days = r.ReadUint16()
	}
}
