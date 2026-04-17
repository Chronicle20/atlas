package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type OperationMerchantAddToBlackList struct {
	name string
}

func (m OperationMerchantAddToBlackList) Name() string { return m.name }

func (m OperationMerchantAddToBlackList) Operation() string {
	return "OperationMerchantAddToBlackList"
}

func (m OperationMerchantAddToBlackList) String() string {
	return fmt.Sprintf("name [%s]", m.name)
}

func (m OperationMerchantAddToBlackList) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.name)
		return w.Bytes()
	}
}

func (m *OperationMerchantAddToBlackList) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.name = r.ReadAsciiString()
	}
}
