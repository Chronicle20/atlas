package interaction

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type OperationMerchantRemoveFromBlackList struct {
	name string
}

func (m OperationMerchantRemoveFromBlackList) Name() string { return m.name }

func (m OperationMerchantRemoveFromBlackList) Operation() string {
	return "OperationMerchantRemoveFromBlackList"
}

func (m OperationMerchantRemoveFromBlackList) String() string {
	return fmt.Sprintf("name [%s]", m.name)
}

func (m OperationMerchantRemoveFromBlackList) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.name)
		return w.Bytes()
	}
}

func (m *OperationMerchantRemoveFromBlackList) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.name = r.ReadAsciiString()
	}
}
