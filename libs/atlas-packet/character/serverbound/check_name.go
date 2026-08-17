package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const CharacterCheckNameHandle = "CharacterCheckNameHandle"

// CheckName - CLogin::SendCheckDuplicateIDPacket
//
// The cash shop sends the SAME op from its rename dialog
// (CCashShop::SendCheckDuplicateIDPacket); that half is decoded by
// cash/serverbound.CheckNameChange, which answers with a different clientbound
// op. See that type's doc comment for why the channel binds its own handler
// name to this opcode instead of reusing CharacterCheckNameHandle.
type CheckName struct {
	name string
}

func NewCheckName(name string) CheckName {
	return CheckName{name: name}
}

func (m CheckName) Name() string {
	return m.name
}

func (m CheckName) Operation() string {
	return CharacterCheckNameHandle
}

func (m CheckName) String() string {
	return fmt.Sprintf("name [%s]", m.name)
}

func (m CheckName) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.Name())
		return w.Bytes()
	}
}

func (m *CheckName) Decode(l logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.name = r.ReadAsciiString()
	}
}
