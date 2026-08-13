package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const CashShopOperationIncreaseStorageHandle = "CashShopOperationIncreaseStorageHandle"

// ShopOperationIncreaseStorage - CCashShop::SendIncTrunkCount
// hasIncStorageCurrency reports whether the increase-storage request carries the
// 4-byte currency field between isPoints and the item flag.
//
// VERIFIED ABSENT ON v48 ONLY. IDA v48 CCashShop::OnIncTrunkCount @0x44aad1
// builds COutPacket(160) and encodes Encode1(6) @0x44abf4 (the CASHSHOP_OPERATION
// mode), Encode1(isPoints) @0x44ac04 and Encode1(0) @0x44ac0d - three bytes, no
// Encode4. v83 @0x46c55b and v87 @0x4763e0 read Decode1, Decode1, Decode4,
// Decode1; v95 @0x48dc70 keeps the same three-field body after the mode.
//
// v61/v72/v79 are UNRESOLVED - their exports carry an "Unresolved" stub for this
// function and the IDBs do not name a distinct send-site - so the gate excludes
// v48 only and leaves 61..79 on the pre-existing (v83) shape rather than guessing
// a boundary. If a legacy tenant ever routes CASHSHOP_OPERATION, derive the
// send-site for that version before trusting this.
func hasIncStorageCurrency(t tenant.Model) bool {
	return !t.IsRegion("GMS") || t.MajorAtLeast(61)
}

// packet-audit:fname CCashShop::OnIncTrunkCount
type ShopOperationIncreaseStorage struct {
	isPoints     bool
	currency     uint32
	item         bool
	serialNumber uint32
}

func (m ShopOperationIncreaseStorage) IsPoints() bool       { return m.isPoints }
func (m ShopOperationIncreaseStorage) Currency() uint32     { return m.currency }
func (m ShopOperationIncreaseStorage) Item() bool           { return m.item }
func (m ShopOperationIncreaseStorage) SerialNumber() uint32 { return m.serialNumber }

func (m ShopOperationIncreaseStorage) Operation() string {
	return CashShopOperationIncreaseStorageHandle
}

func (m ShopOperationIncreaseStorage) String() string {
	return fmt.Sprintf("isPoints [%t], currency [%d], item [%t], serialNumber [%d]", m.isPoints, m.currency, m.item, m.serialNumber)
}

func (m ShopOperationIncreaseStorage) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteBool(m.isPoints)
		if hasIncStorageCurrency(t) {
			w.WriteInt(m.currency)
		}
		w.WriteBool(m.item)
		if m.item {
			w.WriteInt(m.serialNumber)
		}
		return w.Bytes()
	}
}

func (m *ShopOperationIncreaseStorage) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.isPoints = r.ReadBool()
		if hasIncStorageCurrency(t) {
			m.currency = r.ReadUint32()
		}
		m.item = r.ReadBool()
		if m.item {
			m.serialNumber = r.ReadUint32()
		}
	}
}
