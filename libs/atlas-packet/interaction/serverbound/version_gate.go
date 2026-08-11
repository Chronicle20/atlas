package serverbound

import (
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// tradeCrcPresent reports whether the PlayerInteraction trade/shop confirm and
// buy packets carry the per-item CRC anti-hack payload (trade/transaction
// confirm entry lists, and the trailing item CRC on a personal-store/merchant
// buy).
//
// Absent in GMS v79 (and earlier); present from GMS v83 onward and in JMS.
//
// The boundary is anchored on the CONFIRM senders, IDA-verified on GMS_v79
// (mode bytes sent to CClientSocket, opcode 120):
//   - CTradingRoomDlg::Trade @0x73709a — Encode1(0x11) only, no entry list.
//   - CPersonalShopDlg::BuyItem @0x689ce7 — Encode1(mode),Encode1(index),
//     Encode2(quantity); no trailing itemCRC Encode4.
//
// The GMS v83 confirm senders (fixture-verified) append Encode1(count) +
// per-entry Encode4(itemId),Encode4(crc), so the boundary sits between v79 and
// v83.
//
// NOTE (task-205): an earlier revision of this comment also cited
// CCashTradingRoomDlg::Trade @0x47e5f5 as if it were the TRANSACTION sender. It
// is not — on every version that function is the CASH room's Trade BUTTON
// handler and encodes Encode1(0x11), i.e. TRADE_CONFIRM, verified on gms_v83
// @0x485dcd and gms_v95 @0x49e180. It remains valid evidence for THIS gate
// (it is a confirm sender, and on v79 it too carries no entry list), just not
// for TRANSACTION.
//
// For OperationTransaction the gate is belt-and-braces below v83 rather than
// load-bearing: that packet does not exist there at all. Its only sender is the
// clientbound confirm arm's auto-reply, CTradingRoomDlg::OnTrade, which
// constructs no COutPacket on gms_v48 @0x5e6bd3, v61 @0x68b484, v72 @0x6fddec
// or v79 @0x7358c4 — see each version's _unimplemented.json and
// docs/tasks/task-205-player-trade/version-matrix.md §2.
func tradeCrcPresent(t tenant.Model) bool {
	return (t.Region() == "GMS" && t.MajorVersion() >= 83) || t.Region() != "GMS"
}
