package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// LoadGiftFailed - LOAD_GIFT_FAILED arm (CCashShop::OnCashItemResLoadGiftFailed):
// mode + reason byte. Discrete per-mode struct: fixes the LOAD_GIFT_FAILED
// operation key; never accepts the mode from the caller.
// packet-audit:fname CCashShop::OnCashItemResult#LOAD_GIFT_FAILED
type LoadGiftFailed struct {
	mode      byte
	errorCode byte
}

func NewLoadGiftFailed(mode byte, errorCode byte) LoadGiftFailed {
	return LoadGiftFailed{mode: mode, errorCode: errorCode}
}

func (m LoadGiftFailed) Mode() byte        { return m.mode }
func (m LoadGiftFailed) ErrorCode() byte   { return m.errorCode }
func (m LoadGiftFailed) Operation() string { return CashShopOperationWriter }

func (m LoadGiftFailed) String() string {
	return fmt.Sprintf("cash load-gift failed mode [%d] errorCode [%d]", m.mode, m.errorCode)
}

func (m LoadGiftFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.errorCode)
		return w.Bytes()
	}
}

func (m *LoadGiftFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.errorCode = r.ReadByte()
	}
}

// LoadWishFailed - LOAD_WISH_FAILED arm (CCashShop::OnCashItemResLoadWishFailed):
// mode + reason byte.
// packet-audit:fname CCashShop::OnCashItemResult#LOAD_WISH_FAILED
type LoadWishFailed struct {
	mode      byte
	errorCode byte
}

func NewLoadWishFailed(mode byte, errorCode byte) LoadWishFailed {
	return LoadWishFailed{mode: mode, errorCode: errorCode}
}

func (m LoadWishFailed) Mode() byte        { return m.mode }
func (m LoadWishFailed) ErrorCode() byte   { return m.errorCode }
func (m LoadWishFailed) Operation() string { return CashShopOperationWriter }

func (m LoadWishFailed) String() string {
	return fmt.Sprintf("cash load-wish failed mode [%d] errorCode [%d]", m.mode, m.errorCode)
}

func (m LoadWishFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.errorCode)
		return w.Bytes()
	}
}

func (m *LoadWishFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.errorCode = r.ReadByte()
	}
}

// SetWishFailed - SET_WISH_FAILED arm (CCashShop::OnCashItemResSetWishFailed):
// mode + reason byte.
// packet-audit:fname CCashShop::OnCashItemResult#SET_WISH_FAILED
type SetWishFailed struct {
	mode      byte
	errorCode byte
}

func NewSetWishFailed(mode byte, errorCode byte) SetWishFailed {
	return SetWishFailed{mode: mode, errorCode: errorCode}
}

func (m SetWishFailed) Mode() byte        { return m.mode }
func (m SetWishFailed) ErrorCode() byte   { return m.errorCode }
func (m SetWishFailed) Operation() string { return CashShopOperationWriter }

func (m SetWishFailed) String() string {
	return fmt.Sprintf("cash set-wish failed mode [%d] errorCode [%d]", m.mode, m.errorCode)
}

func (m SetWishFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.errorCode)
		return w.Bytes()
	}
}

func (m *SetWishFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.errorCode = r.ReadByte()
	}
}

// BuyFailed - BUY_FAILED arm (CCashShop::OnCashItemResBuyFailed): mode + reason
// byte. Client reads an extra goodsSN:Decode4 when reason is one of a
// version-specific two-value trigger set (see arm-catalog.md "Per-version wire
// divergences" §1 goodsSN); atlas emits only the base mode+reason (no producer
// sends goodsSN — non-goal per task-183 design decision).
// packet-audit:fname CCashShop::OnCashItemResult#BUY_FAILED
type BuyFailed struct {
	mode      byte
	errorCode byte
}

func NewBuyFailed(mode byte, errorCode byte) BuyFailed {
	return BuyFailed{mode: mode, errorCode: errorCode}
}

func (m BuyFailed) Mode() byte        { return m.mode }
func (m BuyFailed) ErrorCode() byte   { return m.errorCode }
func (m BuyFailed) Operation() string { return CashShopOperationWriter }

func (m BuyFailed) String() string {
	return fmt.Sprintf("cash buy failed mode [%d] errorCode [%d]", m.mode, m.errorCode)
}

func (m BuyFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.errorCode)
		return w.Bytes()
	}
}

func (m *BuyFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.errorCode = r.ReadByte()
	}
}

// UseCouponFailed - USE_COUPON_FAILED arm (CCashShop::OnCashItemResUseCouponFailed):
// mode + reason byte.
// packet-audit:fname CCashShop::OnCashItemResult#USE_COUPON_FAILED
type UseCouponFailed struct {
	mode      byte
	errorCode byte
}

func NewUseCouponFailed(mode byte, errorCode byte) UseCouponFailed {
	return UseCouponFailed{mode: mode, errorCode: errorCode}
}

func (m UseCouponFailed) Mode() byte        { return m.mode }
func (m UseCouponFailed) ErrorCode() byte   { return m.errorCode }
func (m UseCouponFailed) Operation() string { return CashShopOperationWriter }

func (m UseCouponFailed) String() string {
	return fmt.Sprintf("cash use-coupon failed mode [%d] errorCode [%d]", m.mode, m.errorCode)
}

func (m UseCouponFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.errorCode)
		return w.Bytes()
	}
}

func (m *UseCouponFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.errorCode = r.ReadByte()
	}
}

// GiftFailed - GIFT_FAILED arm (CCashShop::OnCashItemResGiftFailed): mode +
// reason byte.
// packet-audit:fname CCashShop::OnCashItemResult#GIFT_FAILED
type GiftFailed struct {
	mode      byte
	errorCode byte
}

func NewGiftFailed(mode byte, errorCode byte) GiftFailed {
	return GiftFailed{mode: mode, errorCode: errorCode}
}

func (m GiftFailed) Mode() byte        { return m.mode }
func (m GiftFailed) ErrorCode() byte   { return m.errorCode }
func (m GiftFailed) Operation() string { return CashShopOperationWriter }

func (m GiftFailed) String() string {
	return fmt.Sprintf("cash gift failed mode [%d] errorCode [%d]", m.mode, m.errorCode)
}

func (m GiftFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.errorCode)
		return w.Bytes()
	}
}

func (m *GiftFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.errorCode = r.ReadByte()
	}
}

// IncTrunkCountFailed - INC_TRUNK_COUNT_FAILED arm
// (CCashShop::OnCashItemResIncTrunkCountFailed): mode + reason byte.
// packet-audit:fname CCashShop::OnCashItemResult#INC_TRUNK_COUNT_FAILED
type IncTrunkCountFailed struct {
	mode      byte
	errorCode byte
}

func NewIncTrunkCountFailed(mode byte, errorCode byte) IncTrunkCountFailed {
	return IncTrunkCountFailed{mode: mode, errorCode: errorCode}
}

func (m IncTrunkCountFailed) Mode() byte        { return m.mode }
func (m IncTrunkCountFailed) ErrorCode() byte   { return m.errorCode }
func (m IncTrunkCountFailed) Operation() string { return CashShopOperationWriter }

func (m IncTrunkCountFailed) String() string {
	return fmt.Sprintf("cash inc-trunk-count failed mode [%d] errorCode [%d]", m.mode, m.errorCode)
}

func (m IncTrunkCountFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.errorCode)
		return w.Bytes()
	}
}

func (m *IncTrunkCountFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.errorCode = r.ReadByte()
	}
}

// IncCharacterSlotCountFailed - INC_CHARACTER_SLOT_COUNT_FAILED arm
// (CCashShop::OnCashItemResIncCharacterSlotCountFailed): mode + reason byte.
// packet-audit:fname CCashShop::OnCashItemResult#INC_CHARACTER_SLOT_COUNT_FAILED
type IncCharacterSlotCountFailed struct {
	mode      byte
	errorCode byte
}

func NewIncCharacterSlotCountFailed(mode byte, errorCode byte) IncCharacterSlotCountFailed {
	return IncCharacterSlotCountFailed{mode: mode, errorCode: errorCode}
}

func (m IncCharacterSlotCountFailed) Mode() byte        { return m.mode }
func (m IncCharacterSlotCountFailed) ErrorCode() byte   { return m.errorCode }
func (m IncCharacterSlotCountFailed) Operation() string { return CashShopOperationWriter }

func (m IncCharacterSlotCountFailed) String() string {
	return fmt.Sprintf("cash inc-character-slot-count failed mode [%d] errorCode [%d]", m.mode, m.errorCode)
}

func (m IncCharacterSlotCountFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.errorCode)
		return w.Bytes()
	}
}

func (m *IncCharacterSlotCountFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.errorCode = r.ReadByte()
	}
}

// IncBuyCharacterCountFailed - INC_BUY_CHARACTER_COUNT_FAILED arm
// (CCashShop::OnCashItemResIncBuyCharacterCountFailed): mode + reason byte.
// packet-audit:fname CCashShop::OnCashItemResult#INC_BUY_CHARACTER_COUNT_FAILED
type IncBuyCharacterCountFailed struct {
	mode      byte
	errorCode byte
}

func NewIncBuyCharacterCountFailed(mode byte, errorCode byte) IncBuyCharacterCountFailed {
	return IncBuyCharacterCountFailed{mode: mode, errorCode: errorCode}
}

func (m IncBuyCharacterCountFailed) Mode() byte        { return m.mode }
func (m IncBuyCharacterCountFailed) ErrorCode() byte   { return m.errorCode }
func (m IncBuyCharacterCountFailed) Operation() string { return CashShopOperationWriter }

func (m IncBuyCharacterCountFailed) String() string {
	return fmt.Sprintf("cash inc-buy-character-count failed mode [%d] errorCode [%d]", m.mode, m.errorCode)
}

func (m IncBuyCharacterCountFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.errorCode)
		return w.Bytes()
	}
}

func (m *IncBuyCharacterCountFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.errorCode = r.ReadByte()
	}
}

// EnableEquipSlotExtFailed - ENABLE_EQUIP_SLOT_EXT_FAILED arm
// (CCashShop::OnCashItemResEnableEquipSlotExtFailed): mode + reason byte.
// packet-audit:fname CCashShop::OnCashItemResult#ENABLE_EQUIP_SLOT_EXT_FAILED
type EnableEquipSlotExtFailed struct {
	mode      byte
	errorCode byte
}

func NewEnableEquipSlotExtFailed(mode byte, errorCode byte) EnableEquipSlotExtFailed {
	return EnableEquipSlotExtFailed{mode: mode, errorCode: errorCode}
}

func (m EnableEquipSlotExtFailed) Mode() byte        { return m.mode }
func (m EnableEquipSlotExtFailed) ErrorCode() byte   { return m.errorCode }
func (m EnableEquipSlotExtFailed) Operation() string { return CashShopOperationWriter }

func (m EnableEquipSlotExtFailed) String() string {
	return fmt.Sprintf("cash enable-equip-slot-ext failed mode [%d] errorCode [%d]", m.mode, m.errorCode)
}

func (m EnableEquipSlotExtFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.errorCode)
		return w.Bytes()
	}
}

func (m *EnableEquipSlotExtFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.errorCode = r.ReadByte()
	}
}

// MoveLToSFailed - MOVE_L_TO_S_FAILED arm (CCashShop::OnCashItemResMoveLtoSFailed):
// mode + reason byte.
// packet-audit:fname CCashShop::OnCashItemResult#MOVE_L_TO_S_FAILED
type MoveLToSFailed struct {
	mode      byte
	errorCode byte
}

func NewMoveLToSFailed(mode byte, errorCode byte) MoveLToSFailed {
	return MoveLToSFailed{mode: mode, errorCode: errorCode}
}

func (m MoveLToSFailed) Mode() byte        { return m.mode }
func (m MoveLToSFailed) ErrorCode() byte   { return m.errorCode }
func (m MoveLToSFailed) Operation() string { return CashShopOperationWriter }

func (m MoveLToSFailed) String() string {
	return fmt.Sprintf("cash move-l-to-s failed mode [%d] errorCode [%d]", m.mode, m.errorCode)
}

func (m MoveLToSFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.errorCode)
		return w.Bytes()
	}
}

func (m *MoveLToSFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.errorCode = r.ReadByte()
	}
}

// MoveSToLFailed - MOVE_S_TO_L_FAILED arm (CCashShop::OnCashItemResMoveStoLFailed):
// mode + reason byte.
// packet-audit:fname CCashShop::OnCashItemResult#MOVE_S_TO_L_FAILED
type MoveSToLFailed struct {
	mode      byte
	errorCode byte
}

func NewMoveSToLFailed(mode byte, errorCode byte) MoveSToLFailed {
	return MoveSToLFailed{mode: mode, errorCode: errorCode}
}

func (m MoveSToLFailed) Mode() byte        { return m.mode }
func (m MoveSToLFailed) ErrorCode() byte   { return m.errorCode }
func (m MoveSToLFailed) Operation() string { return CashShopOperationWriter }

func (m MoveSToLFailed) String() string {
	return fmt.Sprintf("cash move-s-to-l failed mode [%d] errorCode [%d]", m.mode, m.errorCode)
}

func (m MoveSToLFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.errorCode)
		return w.Bytes()
	}
}

func (m *MoveSToLFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.errorCode = r.ReadByte()
	}
}

// DestroyFailed - DESTROY_FAILED arm (CCashShop::OnCashItemResDestroyFailed):
// mode + reason byte.
// packet-audit:fname CCashShop::OnCashItemResult#DESTROY_FAILED
type DestroyFailed struct {
	mode      byte
	errorCode byte
}

func NewDestroyFailed(mode byte, errorCode byte) DestroyFailed {
	return DestroyFailed{mode: mode, errorCode: errorCode}
}

func (m DestroyFailed) Mode() byte        { return m.mode }
func (m DestroyFailed) ErrorCode() byte   { return m.errorCode }
func (m DestroyFailed) Operation() string { return CashShopOperationWriter }

func (m DestroyFailed) String() string {
	return fmt.Sprintf("cash destroy failed mode [%d] errorCode [%d]", m.mode, m.errorCode)
}

func (m DestroyFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.errorCode)
		return w.Bytes()
	}
}

func (m *DestroyFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.errorCode = r.ReadByte()
	}
}

// RebateFailed - REBATE_FAILED arm (CCashShop::OnCashItemResRebateFailed):
// mode + reason byte.
// packet-audit:fname CCashShop::OnCashItemResult#REBATE_FAILED
type RebateFailed struct {
	mode      byte
	errorCode byte
}

func NewRebateFailed(mode byte, errorCode byte) RebateFailed {
	return RebateFailed{mode: mode, errorCode: errorCode}
}

func (m RebateFailed) Mode() byte        { return m.mode }
func (m RebateFailed) ErrorCode() byte   { return m.errorCode }
func (m RebateFailed) Operation() string { return CashShopOperationWriter }

func (m RebateFailed) String() string {
	return fmt.Sprintf("cash rebate failed mode [%d] errorCode [%d]", m.mode, m.errorCode)
}

func (m RebateFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.errorCode)
		return w.Bytes()
	}
}

func (m *RebateFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.errorCode = r.ReadByte()
	}
}

// CoupleFailed - COUPLE_FAILED arm (CCashShop::OnCashItemResCoupleFailed): mode
// + reason byte. Client reads an extra goodsSN:Decode4 when reason is one of a
// version-specific two-value trigger set (see arm-catalog.md "Per-version wire
// divergences" §1 goodsSN); atlas emits only the base mode+reason (no producer
// sends goodsSN — non-goal per task-183 design decision).
// packet-audit:fname CCashShop::OnCashItemResult#COUPLE_FAILED
type CoupleFailed struct {
	mode      byte
	errorCode byte
}

func NewCoupleFailed(mode byte, errorCode byte) CoupleFailed {
	return CoupleFailed{mode: mode, errorCode: errorCode}
}

func (m CoupleFailed) Mode() byte        { return m.mode }
func (m CoupleFailed) ErrorCode() byte   { return m.errorCode }
func (m CoupleFailed) Operation() string { return CashShopOperationWriter }

func (m CoupleFailed) String() string {
	return fmt.Sprintf("cash couple failed mode [%d] errorCode [%d]", m.mode, m.errorCode)
}

func (m CoupleFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.errorCode)
		return w.Bytes()
	}
}

func (m *CoupleFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.errorCode = r.ReadByte()
	}
}

// BuyPackageFailed - BUY_PACKAGE_FAILED arm
// (CCashShop::OnCashItemResBuyPackageFailed): mode + reason byte. Client reads
// an extra goodsSN:Decode4 when reason is one of a version-specific two-value
// trigger set (see arm-catalog.md "Per-version wire divergences" §1 goodsSN);
// atlas emits only the base mode+reason (no producer sends goodsSN — non-goal
// per task-183 design decision).
// packet-audit:fname CCashShop::OnCashItemResult#BUY_PACKAGE_FAILED
type BuyPackageFailed struct {
	mode      byte
	errorCode byte
}

func NewBuyPackageFailed(mode byte, errorCode byte) BuyPackageFailed {
	return BuyPackageFailed{mode: mode, errorCode: errorCode}
}

func (m BuyPackageFailed) Mode() byte        { return m.mode }
func (m BuyPackageFailed) ErrorCode() byte   { return m.errorCode }
func (m BuyPackageFailed) Operation() string { return CashShopOperationWriter }

func (m BuyPackageFailed) String() string {
	return fmt.Sprintf("cash buy-package failed mode [%d] errorCode [%d]", m.mode, m.errorCode)
}

func (m BuyPackageFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.errorCode)
		return w.Bytes()
	}
}

func (m *BuyPackageFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.errorCode = r.ReadByte()
	}
}

// GiftPackageFailed - GIFT_PACKAGE_FAILED arm
// (CCashShop::OnCashItemResGiftPackageFailed): mode + reason byte. Client reads
// an extra goodsSN:Decode4 when reason is one of a version-specific two-value
// trigger set (see arm-catalog.md "Per-version wire divergences" §1 goodsSN);
// atlas emits only the base mode+reason (no producer sends goodsSN — non-goal
// per task-183 design decision).
// packet-audit:fname CCashShop::OnCashItemResult#GIFT_PACKAGE_FAILED
type GiftPackageFailed struct {
	mode      byte
	errorCode byte
}

func NewGiftPackageFailed(mode byte, errorCode byte) GiftPackageFailed {
	return GiftPackageFailed{mode: mode, errorCode: errorCode}
}

func (m GiftPackageFailed) Mode() byte        { return m.mode }
func (m GiftPackageFailed) ErrorCode() byte   { return m.errorCode }
func (m GiftPackageFailed) Operation() string { return CashShopOperationWriter }

func (m GiftPackageFailed) String() string {
	return fmt.Sprintf("cash gift-package failed mode [%d] errorCode [%d]", m.mode, m.errorCode)
}

func (m GiftPackageFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.errorCode)
		return w.Bytes()
	}
}

func (m *GiftPackageFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.errorCode = r.ReadByte()
	}
}

// BuyNormalFailed - BUY_NORMAL_FAILED arm
// (CCashShop::OnCashItemResBuyNormalFailed): mode + reason byte. Client reads
// an extra goodsSN:Decode4 when reason is one of a version-specific two-value
// trigger set (see arm-catalog.md "Per-version wire divergences" §1 goodsSN);
// atlas emits only the base mode+reason (no producer sends goodsSN — non-goal
// per task-183 design decision).
// packet-audit:fname CCashShop::OnCashItemResult#BUY_NORMAL_FAILED
type BuyNormalFailed struct {
	mode      byte
	errorCode byte
}

func NewBuyNormalFailed(mode byte, errorCode byte) BuyNormalFailed {
	return BuyNormalFailed{mode: mode, errorCode: errorCode}
}

func (m BuyNormalFailed) Mode() byte        { return m.mode }
func (m BuyNormalFailed) ErrorCode() byte   { return m.errorCode }
func (m BuyNormalFailed) Operation() string { return CashShopOperationWriter }

func (m BuyNormalFailed) String() string {
	return fmt.Sprintf("cash buy-normal failed mode [%d] errorCode [%d]", m.mode, m.errorCode)
}

func (m BuyNormalFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.errorCode)
		return w.Bytes()
	}
}

func (m *BuyNormalFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.errorCode = r.ReadByte()
	}
}

// FriendshipFailed - FRIENDSHIP_FAILED arm
// (CCashShop::OnCashItemResFriendShipFailed): mode + reason byte. Client reads
// an extra goodsSN:Decode4 when reason is one of a version-specific two-value
// trigger set (see arm-catalog.md "Per-version wire divergences" §1 goodsSN);
// atlas emits only the base mode+reason (no producer sends goodsSN — non-goal
// per task-183 design decision).
// packet-audit:fname CCashShop::OnCashItemResult#FRIENDSHIP_FAILED
type FriendshipFailed struct {
	mode      byte
	errorCode byte
}

func NewFriendshipFailed(mode byte, errorCode byte) FriendshipFailed {
	return FriendshipFailed{mode: mode, errorCode: errorCode}
}

func (m FriendshipFailed) Mode() byte        { return m.mode }
func (m FriendshipFailed) ErrorCode() byte   { return m.errorCode }
func (m FriendshipFailed) Operation() string { return CashShopOperationWriter }

func (m FriendshipFailed) String() string {
	return fmt.Sprintf("cash friendship failed mode [%d] errorCode [%d]", m.mode, m.errorCode)
}

func (m FriendshipFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.errorCode)
		return w.Bytes()
	}
}

func (m *FriendshipFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.errorCode = r.ReadByte()
	}
}

// PurchaseRecordFailed - PURCHASE_RECORD_FAILED arm
// (CCashShop::OnCashItemResPurchaseRecordFailed): mode + reason byte. The client
// reads the reason byte and discards it (no further use) — the byte is present
// on the wire, so this is modeled as a normal mode+reason failure arm (task-183
// design decision #3).
// packet-audit:fname CCashShop::OnCashItemResult#PURCHASE_RECORD_FAILED
type PurchaseRecordFailed struct {
	mode      byte
	errorCode byte
}

func NewPurchaseRecordFailed(mode byte, errorCode byte) PurchaseRecordFailed {
	return PurchaseRecordFailed{mode: mode, errorCode: errorCode}
}

func (m PurchaseRecordFailed) Mode() byte        { return m.mode }
func (m PurchaseRecordFailed) ErrorCode() byte   { return m.errorCode }
func (m PurchaseRecordFailed) Operation() string { return CashShopOperationWriter }

func (m PurchaseRecordFailed) String() string {
	return fmt.Sprintf("cash purchase-record failed mode [%d] errorCode [%d]", m.mode, m.errorCode)
}

func (m PurchaseRecordFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.errorCode)
		return w.Bytes()
	}
}

func (m *PurchaseRecordFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.errorCode = r.ReadByte()
	}
}

// TransferWorldFailed - TRANSFER_WORLD_FAILED arm
// (CCashShop::OnCashItemResTransferWorldFailed): mode + reason byte.
// packet-audit:fname CCashShop::OnCashItemResult#TRANSFER_WORLD_FAILED
type TransferWorldFailed struct {
	mode      byte
	errorCode byte
}

func NewTransferWorldFailed(mode byte, errorCode byte) TransferWorldFailed {
	return TransferWorldFailed{mode: mode, errorCode: errorCode}
}

func (m TransferWorldFailed) Mode() byte        { return m.mode }
func (m TransferWorldFailed) ErrorCode() byte   { return m.errorCode }
func (m TransferWorldFailed) Operation() string { return CashShopOperationWriter }

func (m TransferWorldFailed) String() string {
	return fmt.Sprintf("cash transfer-world failed mode [%d] errorCode [%d]", m.mode, m.errorCode)
}

func (m TransferWorldFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.errorCode)
		return w.Bytes()
	}
}

func (m *TransferWorldFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.errorCode = r.ReadByte()
	}
}

// GachaponOpenFailed - GACHAPON_OPEN_FAILED arm
// (CCashShop::OnCashItemResCashGachaponOpenFailed): mode + reason byte.
// packet-audit:fname CCashShop::OnCashItemResult#GACHAPON_OPEN_FAILED
type GachaponOpenFailed struct {
	mode      byte
	errorCode byte
}

func NewGachaponOpenFailed(mode byte, errorCode byte) GachaponOpenFailed {
	return GachaponOpenFailed{mode: mode, errorCode: errorCode}
}

func (m GachaponOpenFailed) Mode() byte        { return m.mode }
func (m GachaponOpenFailed) ErrorCode() byte   { return m.errorCode }
func (m GachaponOpenFailed) Operation() string { return CashShopOperationWriter }

func (m GachaponOpenFailed) String() string {
	return fmt.Sprintf("cash gachapon-open failed mode [%d] errorCode [%d]", m.mode, m.errorCode)
}

func (m GachaponOpenFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.errorCode)
		return w.Bytes()
	}
}

func (m *GachaponOpenFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.errorCode = r.ReadByte()
	}
}

// GachaponCopyFailed - GACHAPON_COPY_FAILED arm
// (CCashShop::OnCashItemResCashGachaponCopyFailed): mode + reason byte.
// packet-audit:fname CCashShop::OnCashItemResult#GACHAPON_COPY_FAILED
type GachaponCopyFailed struct {
	mode      byte
	errorCode byte
}

func NewGachaponCopyFailed(mode byte, errorCode byte) GachaponCopyFailed {
	return GachaponCopyFailed{mode: mode, errorCode: errorCode}
}

func (m GachaponCopyFailed) Mode() byte        { return m.mode }
func (m GachaponCopyFailed) ErrorCode() byte   { return m.errorCode }
func (m GachaponCopyFailed) Operation() string { return CashShopOperationWriter }

func (m GachaponCopyFailed) String() string {
	return fmt.Sprintf("cash gachapon-copy failed mode [%d] errorCode [%d]", m.mode, m.errorCode)
}

func (m GachaponCopyFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.errorCode)
		return w.Bytes()
	}
}

func (m *GachaponCopyFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.errorCode = r.ReadByte()
	}
}

// ChangeMaplePointFailed - CHANGE_MAPLE_POINT_FAILED arm
// (CCashShop::OnCashItemResChangeMaplePointFailed): BODYLESS — mode byte only.
// RE confirmed zero Decode calls beyond the dispatcher's own mode-byte read
// (arm-catalog.md CHANGE_MAPLE_POINT_FAILED row: "NO Decode1/4 call in handler").
// packet-audit:fname CCashShop::OnCashItemResult#CHANGE_MAPLE_POINT_FAILED
type ChangeMaplePointFailed struct {
	mode byte
}

func NewChangeMaplePointFailed(mode byte) ChangeMaplePointFailed {
	return ChangeMaplePointFailed{mode: mode}
}

func (m ChangeMaplePointFailed) Mode() byte        { return m.mode }
func (m ChangeMaplePointFailed) Operation() string { return CashShopOperationWriter }

func (m ChangeMaplePointFailed) String() string {
	return fmt.Sprintf("cash change-maple-point failed mode [%d]", m.mode)
}

func (m ChangeMaplePointFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *ChangeMaplePointFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}
