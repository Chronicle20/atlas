package serverbound

import (
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// hasLeadingPetId reports whether a client -> server pet action packet carries
// the leading 8-byte pet identity (cash-item SN) before its body. Present on
// GMS v61+ and all JMS; ABSENT on the oldest GMS build v48, which predates
// multi-pet and identifies the (single) pet implicitly.
//
// IDA-verified at v48 (GMS_v48_1_DEVM.exe, port 13337) across the whole pet
// action cluster — every send-site opens COutPacket(op) and writes its body
// with NO leading EncodeBuffer(petId,8):
//
//	MOVE_PET     op113 sub_6E5BD6 @0x6e5bff   COutPacket(113)+CMovePath::Flush only
//	PET_CHAT     op114 CPet::DoAction @0x58e90b  Encode1(type)+Encode1(action)+EncodeStr(msg)
//	PET_COMMAND  op115 sub_58DF8A @0x58e1b8   Encode1(byName)+Encode1(command)
//	PET_LOOT     op116 sub_58ED98 @0x58edb0   Encode1(fieldKey)+Encode4(time)+…(no petId)
//
// The v61 twins all lead with EncodeBuffer(petId,8) (e.g. PET_COMMAND
// @0x613d66, PET_CHAT @0x61456f), so the gate is "present iff not GMS<61".
func hasLeadingPetId(t tenant.Model) bool {
	return !(t.IsRegion("GMS") && !t.MajorAtLeast(61))
}
