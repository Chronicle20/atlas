package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// ItemUsePetSkill is the type-28 sub-body of CUser::SendCashItemUseRequest /
// CWvsContext::SendConsumeCashItemUseRequest: a 0519 pet skill pouch use.
// Only the jms client has a case-28 arm (GMS builds have no such case).
//
// Verified via IDA jms_v185 (session 3c4bb8b1, MapleStory_dump_SCY.exe.i64,
// CWvsContext::SendConsumeCashItemUseRequest @0xaef2f5): the jump-table
// case-28 arm (entry @0xaf16df) contains exactly one encode call on its
// success path -- EncodeBuffer(pet+0x18, 8) @0xaf1a42:
//
//	0xaf1a39  push  8            ; uSize
//	0xaf1a3b  add   eax, 18h     ; eax = pet object + 0x18
//	0xaf1a3e  push  eax          ; p
//	0xaf1a3f  lea   ecx, var_18  ; this (COutPacket)
//	0xaf1a42  call  EncodeBuffer@COutPacket
//
// No leading or trailing updateTime is written in this arm: update_time is
// already consumed once in the common SendConsumeCashItemUseRequest header
// (Encode4 @0xaef375, ahead of the switch), which cashsb.ItemUse already
// decodes for MajorVersion()>=87 (jms_v185 included). So the sub-body here
// is nothing but the raw 8-byte pet locker SN, which round-trips as the
// Atlas pet id.
// packet-audit:fname CWvsContext::SendConsumeCashItemUseRequest
type ItemUsePetSkill struct {
	petId uint64
}

func NewItemUsePetSkill() *ItemUsePetSkill {
	return &ItemUsePetSkill{}
}

func (m ItemUsePetSkill) PetId() uint64 { return m.petId }

func (m ItemUsePetSkill) Operation() string { return "ItemUsePetSkill" }

func (m ItemUsePetSkill) String() string {
	return fmt.Sprintf("petId [%d]", m.petId)
}

func (m ItemUsePetSkill) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteLong(m.petId)
		return w.Bytes()
	}
}

func (m *ItemUsePetSkill) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.petId = r.ReadUint64()
	}
}
