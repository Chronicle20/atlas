package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterCheckNameHandle = "CharacterCheckNameHandle"

// CheckName - CLogin::SendCheckDuplicateIDPacket
type CheckName struct {
	name string
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
