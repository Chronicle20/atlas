package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const CharacterUseDeathItemHandle = "CharacterUseDeathItemHandle"

// UseDeathItem - CUserLocal::RequestUpgradeTombEffect
// packet-audit:fname CUserLocal::RequestUpgradeTombEffect
//
// Sent by CUIRevive::OnCreate — at death-dialog construction time, before the
// player has chosen anything — when the field allows the wheel and the player
// owns at least one. It is a request to play the tomb effect for bystanders,
// NOT a request to revive or to consume the item: the revive itself is
// CUIRevive::Revive -> CField::SendTransferFieldRequest (MAP_CHANGE). The
// client plays its own copy of the effect locally via
// CUser::ShowUpgradeTombEffect immediately after sending, so the server must
// not echo the effect back to the sender.
//
// Wire layout — identical on every version that carries the op (gms v72, v79,
// v83, v84, v87, v92, v95, jms v185); only the opcode differs, and that is
// tenant configuration. There is deliberately no version gate.
//
//	Encode4  itemId   — hard-coded 5510000 (0x541370) by the client
//	Encode4  x        — m_ptRevive.x
//	Encode4  y        — m_ptRevive.y
//
// IDA gms_v95 @0x908320, gms_v92 @0x8ee9f0, gms_v87 @0x9dd673, gms_v84
// @0x999277, gms_v83 @0x95af8e, gms_v79 @0x8b2ff0, gms_v72 @0x867654,
// jms_v185 @0xa25fc9.
type UseDeathItem struct {
	itemId uint32
	x      int32
	y      int32
}

func NewUseDeathItem(itemId uint32, x int32, y int32) UseDeathItem {
	return UseDeathItem{itemId: itemId, x: x, y: y}
}

func (m UseDeathItem) ItemId() uint32    { return m.itemId }
func (m UseDeathItem) X() int32          { return m.x }
func (m UseDeathItem) Y() int32          { return m.y }
func (m UseDeathItem) Operation() string { return CharacterUseDeathItemHandle }
func (m UseDeathItem) String() string {
	return fmt.Sprintf("itemId [%d], x [%d], y [%d]", m.itemId, m.x, m.y)
}

func (m UseDeathItem) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.itemId)
		w.WriteInt32(m.x)
		w.WriteInt32(m.y)
		return w.Bytes()
	}
}

func (m *UseDeathItem) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.itemId = r.ReadUint32()
		m.x = r.ReadInt32()
		m.y = r.ReadInt32()
	}
}
