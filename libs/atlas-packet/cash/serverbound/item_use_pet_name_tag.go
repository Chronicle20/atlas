package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// ItemUsePetNameTag is the CashSlotItemType(17) sub-body of USE_CASH_ITEM —
// the new name a player types for their pet with a classification-517 Pet Name
// Tag (5170000).
//
// Derived from the case-17 arm of CWvsContext::SendConsumeCashItemUseRequest
// (gms_v83 @0xa0a63f; arm entry @0xa0ba15, labelled by IDA as
// "jumptable 00A0A6E6 case 17"). The arm resolves the pet at locker index 0
// (sub_46D2D5(this, 0) @0xa0ba47), prompts twice (CUtilDlg::YesNo @0xa0baa2 and
// @0xa0bc88), reads the name from a 4..12-bounded input dialog
// (sub_9AC7CB(dlg, NULL, 4, 12, 0, 1) @0xa0bb2f, GetInputStr_Result @0xa0bb68),
// screens it through CCurseProcess::ProcessString @0xa0bb9a, and then performs
// its ONLY encode: COutPacket::EncodeStr @0xa0bcb5.
//
// So the sub-body is exactly one length-prefixed string. NO pet identifier is on
// the wire — not an index, not a locker SN, not a slot. SetUtilDlgEx_Pet
// (@0x9acb27, the pet-picker) is never called from this send path; its only
// callers are CDraggableItem::OnDoubleClicked, CScriptMan::OnAskPet, and
// CScriptMan::OnAskPetAll. The server therefore resolves the target pet itself
// (the lead pet, slot 0 — matching the client's own sub_46D2D5(…, 0) choice).
//
// updateTimeFirst mirrors ItemUse.UpdateTimeFirst: GMS <= v84 trails
// update_time after the sub-body (this arm falls through to the shared tail at
// loc_A0E9EC), GMS v87+ and JMS lead it in the header.
type ItemUsePetNameTag struct {
	name            string
	updateTime      uint32
	updateTimeFirst bool
}

func NewItemUsePetNameTag(updateTimeFirst bool) *ItemUsePetNameTag {
	return &ItemUsePetNameTag{updateTimeFirst: updateTimeFirst}
}

func (m ItemUsePetNameTag) Name() string       { return m.name }
func (m ItemUsePetNameTag) UpdateTime() uint32 { return m.updateTime }

func (m ItemUsePetNameTag) Operation() string { return "ItemUsePetNameTag" }

func (m ItemUsePetNameTag) String() string {
	return fmt.Sprintf("name [%s] updateTime [%d]", m.name, m.updateTime)
}

func (m ItemUsePetNameTag) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.name)
		if !m.updateTimeFirst {
			w.WriteInt(m.updateTime)
		}
		return w.Bytes()
	}
}

func (m *ItemUsePetNameTag) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.name = r.ReadAsciiString()
		if !m.updateTimeFirst {
			m.updateTime = r.ReadUint32()
		}
	}
}
