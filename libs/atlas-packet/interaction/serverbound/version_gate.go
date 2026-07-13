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
// IDA-verified against GMS_v79 (mode bytes sent to CClientSocket, opcode 120):
//   - CTradingRoomDlg::Trade @0x73709a  — Encode1(0x11) only, no entry list.
//   - CCashTradingRoomDlg::Trade @0x47e5f5 — Encode1(0x11) only, no entry list.
//   - CPersonalShopDlg::BuyItem @0x689ce7 — Encode1(mode),Encode1(index),
//     Encode2(quantity); no trailing itemCRC Encode4.
// The GMS v83 senders (already fixture-verified) include the entry list / CRC,
// so the boundary sits between v79 and v83.
func tradeCrcPresent(t tenant.Model) bool {
	return (t.Region() == "GMS" && t.MajorVersion() >= 83) || t.Region() != "GMS"
}
