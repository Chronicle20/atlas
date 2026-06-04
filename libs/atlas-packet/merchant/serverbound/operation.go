package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const HiredMerchantOperationHandle = "HiredMerchantOperationHandle"

// Hired-merchant / entrusted-shop serverbound dispatch modes.
//
// The opcode is a dedicated request (JMS v185 COutPacket type 0x37) — NOT a
// PLAYER_INTERACTION sub-op. The in-shop lifecycle (put item, buy, remove, exit,
// withdraw meso, blacklist, name change, ...) is sent through CharacterInteraction
// instead and is handled by socket/handler/character_interaction.go. This op covers
// only the cash-item permit trigger that opens the entrusted-shop flow.
//
// Confirmed by reading CWvsContext::SendEntrustedShopCheckRequest in the JMS v185 IDB,
// whose sole caller is CWvsContext::SendCashSlotItemUseRequest (cash-slot item type 37,
// the hired-merchant permit). The client always sends mode 0; no other serverbound mode
// for this opcode is emitted by the v185 client.
const (
	// ModeEntrustedShopCheck is sent when the player uses a hired-merchant permit
	// (a cash-shop slot item). Body: mode(0) + cashItemSerialNumber(uint64).
	ModeEntrustedShopCheck byte = 0
)

// Operation is the entrusted-shop (hired-merchant) serverbound request.
//
// Wire shape (mode 0, the only mode the v185 client sends):
//
//	Encode1(mode)                 // always 0
//	EncodeBuffer(&cashItemSN, 8)  // cash-item serial number (uint64)
//
// The inventory position and item id the client knows are NOT on the wire — the
// client stashes them locally and the server resolves them from the serial number.
type Operation struct {
	mode                 byte
	cashItemSerialNumber uint64
}

func (m Operation) Mode() byte { return m.mode }

func (m Operation) CashItemSerialNumber() uint64 { return m.cashItemSerialNumber }

func (m Operation) Operation() string {
	return HiredMerchantOperationHandle
}

func (m Operation) String() string {
	return fmt.Sprintf("mode [%d] cashItemSerialNumber [%d]", m.mode, m.cashItemSerialNumber)
}

func (m Operation) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteLong(m.cashItemSerialNumber)
		return w.Bytes()
	}
}

func (m *Operation) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.cashItemSerialNumber = r.ReadUint64()
	}
}
