package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

// packet-audit:fname CField::SendSetFriendMsg
//
// Wire layout (after the BUDDYLIST_MODIFY mode byte, consumed by the
// dispatcher): the buddy "group" name was introduced with the buddy-group
// feature after GMS v61. IDA-verified: GMS v48 (CField add send @0x4c6452)
// and GMS v61 (@0x4e9c03) send ONLY the buddy name; GMS v72 (@0x515575),
// v79 (@0x51c614) and v87 (CField::SendSetFriendMsg @0x558844) append the
// group name. So the group field is gated on MajorVersion() > 61.
type OperationAdd struct {
	name  string
	group string
}

func (m OperationAdd) Name() string {
	return m.name
}

func (m OperationAdd) Group() string {
	return m.group
}

func (m OperationAdd) Operation() string {
	return "OperationAdd"
}

func (m OperationAdd) String() string {
	return fmt.Sprintf("name [%s] group [%s]", m.name, m.group)
}

func (m OperationAdd) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.name)
		if t.MajorVersion() > 61 {
			w.WriteAsciiString(m.group)
		}
		return w.Bytes()
	}
}

func (m *OperationAdd) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.name = r.ReadAsciiString()
		if t.MajorVersion() > 61 {
			m.group = r.ReadAsciiString()
		}
	}
}
