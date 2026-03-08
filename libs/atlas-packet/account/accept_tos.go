package account

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const AcceptTosHandle = "AcceptTosHandle"

// AcceptTos - CLogin::OnAcceptLicense - CLogin::OnDenyLicense
type AcceptTos struct {
	accepted bool
}

func (m AcceptTos) Accepted() bool {
	return m.accepted
}

func (m AcceptTos) Operation() string {
	return AcceptTosHandle
}

func (m AcceptTos) String() string {
	return fmt.Sprintf("accepted [%t]", m.accepted)
}

func (m AcceptTos) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteBool(m.accepted)
		return w.Bytes()
	}
}

func (m AcceptTos) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.accepted = r.ReadBool()
	}
}
