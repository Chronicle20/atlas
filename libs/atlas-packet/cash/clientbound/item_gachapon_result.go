package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// Cash Shop Surprise result — the STANDALONE opcode
// CASHSHOP_CASH_ITEM_GACHAPON_RESULT, not a CASHSHOP_OPERATION arm. The
// CASHSHOP_OPERATION GACHAPON_OPEN_* arms belong to CUICashGachapon (the
// Cash Gachapon UI), a different feature — see design.md §1.4. This packet's
// handler is CCashShop::OnCashItemGachaponResult; the trailing itemId/count/
// jackpot fields are read by CUICashItemGachapon, not CCashShop, and the
// server always writes them.
//
// Opcodes: v83 0x14D, v84 0x154, v87 0x15E, v92 0x180, v95 0x188,
// jms_v185 0x16D. v79 has NO result handler (n-a) and v48/v61/v72 have no
// CUICashItemGachapon at all (n-a).
const CashItemGachaponResultWriter = "CashItemGachaponResult"

const (
	CashItemGachaponModeSuccess = "SUCCESS"
	CashItemGachaponModeFailed  = "FAILED"
)

// CashItemGachaponSuccess - the SUCCESS arm: mode + sn:DecodeBuffer(8)
// (int64, SN of the consumed box, matched against m_aCashItemInfo[i].liSN) +
// remain:Decode4 (int32, the box's new quantity; 0 removes the locker row) +
// newItem:DecodeBuffer(0x37=55) (GW_CashItemInfo, UNCONDITIONAL — there is no
// isCashItem gate on this packet) + itemId:Decode4 (int32, rewarded template
// id, UI icon + chat log) + count:Decode1 + jackpot:Decode1 (selects the
// CashGachaponJackpot vs CashGachaponNormal sfx).
// packet-audit:fname CCashShop::OnCashItemGachaponResult#SUCCESS
type CashItemGachaponSuccess struct {
	mode    byte
	sn      int64
	remain  int32
	newItem CashInventoryItem
	itemId  int32
	count   byte
	jackpot byte
}

func NewCashItemGachaponSuccess(mode byte, sn int64, remain int32, newItem CashInventoryItem, itemId int32, count byte, jackpot byte) CashItemGachaponSuccess {
	return CashItemGachaponSuccess{mode: mode, sn: sn, remain: remain, newItem: newItem, itemId: itemId, count: count, jackpot: jackpot}
}

func (m CashItemGachaponSuccess) Mode() byte                 { return m.mode }
func (m CashItemGachaponSuccess) SN() int64                  { return m.sn }
func (m CashItemGachaponSuccess) Remain() int32              { return m.remain }
func (m CashItemGachaponSuccess) NewItem() CashInventoryItem { return m.newItem }
func (m CashItemGachaponSuccess) ItemId() int32              { return m.itemId }
func (m CashItemGachaponSuccess) Count() byte                { return m.count }
func (m CashItemGachaponSuccess) Jackpot() byte              { return m.jackpot }
func (m CashItemGachaponSuccess) Operation() string          { return CashItemGachaponResultWriter }

func (m CashItemGachaponSuccess) String() string {
	return fmt.Sprintf("cash-item-gachapon success mode [%d] sn [%d] remain [%d] itemId [%d] count [%d] jackpot [%d]", m.mode, m.sn, m.remain, m.itemId, m.count, m.jackpot)
}

func (m CashItemGachaponSuccess) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt64(m.sn)
		w.WriteInt32(m.remain)
		w.WriteByteArray(m.newItem.EncodeBytes(l))
		w.WriteInt32(m.itemId)
		w.WriteByte(m.count)
		w.WriteByte(m.jackpot)
		return w.Bytes()
	}
}

func (m *CashItemGachaponSuccess) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.sn = r.ReadInt64()
		m.remain = r.ReadInt32()
		m.newItem = decodeCashInventoryItemSkipPadding(r)
		m.itemId = r.ReadInt32()
		m.count = r.ReadByte()
		m.jackpot = r.ReadByte()
	}
}

// CashItemGachaponFailed - the FAILED arm. The client reads NOTHING after
// the mode byte: it calls StringPool::GetString(<fixed id>) and
// CUtilDlg::Notice. There is no error-code field on this wire (design.md
// §2.3), so the distinct failure reasons are logged server-side and carried
// on the status event, never sent to the client. The client also does not
// re-enable the dialog's Open button on failure — that is native behaviour,
// and we replicate it rather than inventing a recovery packet.
// packet-audit:fname CCashShop::OnCashItemGachaponResult#FAILED
type CashItemGachaponFailed struct {
	mode byte
}

func NewCashItemGachaponFailed(mode byte) CashItemGachaponFailed {
	return CashItemGachaponFailed{mode: mode}
}

func (m CashItemGachaponFailed) Mode() byte        { return m.mode }
func (m CashItemGachaponFailed) Operation() string { return CashItemGachaponResultWriter }

func (m CashItemGachaponFailed) String() string {
	return fmt.Sprintf("cash-item-gachapon failed mode [%d]", m.mode)
}

func (m CashItemGachaponFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *CashItemGachaponFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// CashItemGachaponSuccessBody resolves the SUCCESS mode byte from the tenant
// operations table at encode time. The byte differs on EVERY version
// (v83 0xE5, v84 0xEE, v87 0xF4, v92 0xBE, v95 0xC1, jms 0xEB), which is
// exactly the DOM-25 failure mode the rule exists for — never hard-code it.
func CashItemGachaponSuccessBody(sn int64, remain int32, newItem CashInventoryItem, itemId int32, count byte, jackpot byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashItemGachaponModeSuccess, func(mode byte) packet.Encoder {
		return NewCashItemGachaponSuccess(mode, sn, remain, newItem, itemId, count, jackpot)
	})
}

// CashItemGachaponFailedBody resolves the FAILED mode byte the same way
// (v83 0xE4, v84 0xED, v87 0xF3, v92 0xBD, v95 0xC0, jms 0xEA).
func CashItemGachaponFailedBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashItemGachaponModeFailed, func(mode byte) packet.Encoder {
		return NewCashItemGachaponFailed(mode)
	})
}
