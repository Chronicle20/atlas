package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const DragonRemoveWriter = "DragonRemove"

// DragonRemove is the server -> client REMOVE_DRAGON packet. The body is EMPTY:
// the only field is the owner character id, and even that is consumed upstream
// by CUserPool::OnUserCommonPacket before the family dispatch.
//
// THE CLIENT HAS NO HANDLER ARM FOR THIS OPCODE. CUser::OnDragonPacket
// (GMS v95.0 @0x8e5c00) switches on nType: 206 -> spawn, 207 -> move, and
// nothing else. The pool routes 206..208 into it and 208 falls through to no
// code. The same shape holds in v83 (@0x93908f), v84, v87 (@0x9b3880),
// v92 (@0x8ce880) and JMS185 (@0x9f822f).
//
// An xref sweep on ZRef<CDragon>::_ReleaseRaw (v95 @0x8decb0) returns four
// callers: the ZRef destructor, ZRef::operator=, OnDragonPacket's respawn path,
// and CUser::~CUser. The ONLY client-side dragon teardown is destroying the
// CUser — i.e. the owner leaving the field.
//
// So: sending this packet is correct and harmless, but it is NOT the mechanism
// that removes the dragon. Do not "fix" the apparently-missing body.
//
// packet-audit:fname CUser::OnDragonPacket
type DragonRemove struct {
	ownerCharacterId uint32
}

func NewDragonRemove(ownerCharacterId uint32) DragonRemove {
	return DragonRemove{ownerCharacterId: ownerCharacterId}
}

func (m DragonRemove) OwnerCharacterId() uint32 { return m.ownerCharacterId }
func (m DragonRemove) Operation() string        { return DragonRemoveWriter }
func (m DragonRemove) String() string {
	return fmt.Sprintf("ownerCharacterId [%d]", m.ownerCharacterId)
}

func (m DragonRemove) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.ownerCharacterId)
		return w.Bytes()
	}
}

func (m *DragonRemove) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.ownerCharacterId = r.ReadUint32()
	}
}
