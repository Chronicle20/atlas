package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type OperationMerchantNameChange struct {
	unk1 uint32
}

func (m OperationMerchantNameChange) Unk1() uint32 { return m.unk1 }

func (m OperationMerchantNameChange) Operation() string { return "OperationMerchantNameChange" }

func (m OperationMerchantNameChange) String() string {
	return fmt.Sprintf("unk1 [%d]", m.unk1)
}

func (m OperationMerchantNameChange) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.unk1)
		return w.Bytes()
	}
}

func (m *OperationMerchantNameChange) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.unk1 = r.ReadUint32()
	}
}
