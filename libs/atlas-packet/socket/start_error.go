package socket

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const StartErrorHandle = "StartErrorHandle"

// StartError - CClientSocket::OnConnect
type StartError struct {
	length uint16
	bytes  []byte
}

func (m StartError) Length() uint16 {
	return m.length
}

func (m StartError) Bytes() []byte {
	return m.bytes
}

func (m StartError) Operation() string {
	return StartErrorHandle
}

func (m StartError) String() string {
	return fmt.Sprintf("length [%d], bytes [%s]", m.Length(), hex.EncodeToString(m.Bytes()))
}

func (m StartError) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteShort(m.Length())
		w.WriteByteArray(m.Bytes())
		return w.Bytes()
	}
}

func (m *StartError) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.length = r.ReadUint16()
		m.bytes = r.ReadBytes(int(m.length))
	}
}
