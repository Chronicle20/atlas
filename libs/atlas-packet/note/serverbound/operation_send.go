package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// sendHasGiftDetails reports whether the gift-forward writer appends the
// giftIndex (uint32) and giftSN (8 bytes) after the giftFlag. IDA-verified by
// behavior across every version: the earliest client, v48, sends giftFlag only
// (its gift-forward lives in OnCashItemResLoadLockerDone and predates these
// fields); every later build — v61/v72/v79/v83/v84/v87/v95 and jms, in
// OnCashItemResLoadGiftDone — appends both. (Builds between v48 and v61 are not
// in scope; v48 is the only verified outlier.) A code-resolved version fact.
func sendHasGiftDetails(t tenant.Model) bool {
	return !(t.Region() == "GMS" && t.MajorVersion() <= 48)
}

// OperationSend is the NOTE_ACTION mode-0 arm. Its only client-side writer is
// the cash-shop gift-forward flow — CCashShop::OnCashItemResLoadGiftDone on
// v61/v72/v79/v83/v84/v87/v95/jms, and, on the early v48 client, the combined
// CCashShop::OnCashItemResLoadLockerDone (which loads the locker AND forwards
// gifts). After the recipient name and message it appends the gift payload: a
// hardcoded flag byte, then — on every build except v48 — the 0-based gift-list
// index (uint32) and the 8-byte cash-item serial number (see sendHasGiftDetails).
// The player note path is USE_CASH_ITEM, and gift memos are out of scope (design
// §2.3) — the channel handler gates this arm on Note-item ownership using only
// toName+message and never acts on the gift fields; they are decoded here so the
// codec models the complete wire.
//
// packet-audit:fname CCashShop::OnCashItemResLoadGiftDone
type OperationSend struct {
	toName    string
	message   string
	giftFlag  byte
	giftIndex uint32
	giftSN    uint64
}

func (m OperationSend) ToName() string {
	return m.toName
}

func (m OperationSend) Message() string {
	return m.message
}

// GiftFlag is the constant flag byte the gift-forward writer emits (always 1 in
// every version's writer; why the client never varies it is a server-side
// concern not determinable from the client).
func (m OperationSend) GiftFlag() byte {
	return m.giftFlag
}

// GiftIndex is the 0-based position of the gift in the client's just-loaded gift
// list (an enumeration counter, not stored packet data).
func (m OperationSend) GiftIndex() uint32 {
	return m.giftIndex
}

// GiftSN is the 8-byte cash-item serial number of the forwarded gift, copied
// verbatim from the gift-list entry.
func (m OperationSend) GiftSN() uint64 {
	return m.giftSN
}

func (m OperationSend) Operation() string {
	return "OperationSend"
}

func (m OperationSend) String() string {
	return fmt.Sprintf("toName [%s] message [%s] giftFlag [%d] giftIndex [%d] giftSN [%d]", m.toName, m.message, m.giftFlag, m.giftIndex, m.giftSN)
}

func (m OperationSend) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	giftDetails := sendHasGiftDetails(tenant.MustFromContext(ctx))
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.toName)
		w.WriteAsciiString(m.message)
		w.WriteByte(m.giftFlag)
		if giftDetails {
			w.WriteInt(m.giftIndex)
			w.WriteLong(m.giftSN)
		}
		return w.Bytes()
	}
}

func (m *OperationSend) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	giftDetails := sendHasGiftDetails(tenant.MustFromContext(ctx))
	return func(r *request.Reader, options map[string]interface{}) {
		m.toName = r.ReadAsciiString()
		m.message = r.ReadAsciiString()
		m.giftFlag = r.ReadByte()
		if giftDetails {
			m.giftIndex = r.ReadUint32()
			m.giftSN = r.ReadUint64()
		}
	}
}
