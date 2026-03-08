package account

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const SetGenderHandle = "SetGenderHandle"

// SetGender - CLogin::SendSetGenderPacket - CLogin::SendCancelGenderPacket
type SetGender struct {
	set    bool
	gender byte
}

func (m SetGender) Set() bool {
	return m.set
}

func (m SetGender) Gender() byte {
	return m.gender
}

func (m SetGender) Operation() string {
	return SetGenderHandle
}

func (m SetGender) String() string {
	return fmt.Sprintf("set [%t], gender [%d]", m.set, m.gender)
}

func (m SetGender) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteBool(m.set)
		if m.set {
			w.WriteByte(m.gender)
		}
		return w.Bytes()
	}
}

func (m SetGender) Decode(l logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.set = r.ReadBool()
		if !m.set {
			m.gender = r.ReadByte()
		}
	}
}
