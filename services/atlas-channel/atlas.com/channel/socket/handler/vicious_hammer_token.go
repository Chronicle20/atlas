package handler

// The CUIItemUpgrade round-trip token: the server chooses an arbitrary int32
// the client stores (m_nResult) and echoes verbatim in ITEM_UPGRADE_UPDATE.
// Atlas packs both slots into it so the confirm handler is stateless
// (design §4): high int16 = the hammer's cash-compartment slot, low int16 =
// the target equip slot (negative = equipped).

func packViciousHammerToken(hammerSlot int16, equipSlot int16) uint32 {
	return uint32(uint16(hammerSlot))<<16 | uint32(uint16(equipSlot))
}

func unpackViciousHammerToken(token uint32) (int16, int16) {
	return int16(uint16(token >> 16)), int16(uint16(token))
}
