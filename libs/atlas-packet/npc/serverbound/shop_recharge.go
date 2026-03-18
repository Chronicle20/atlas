package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type ShopRecharge struct {
	slot uint16
}

func (m ShopRecharge) Slot() uint16 { return m.slot }

func (m ShopRecharge) Operation() string { return "ShopRecharge" }

func (m ShopRecharge) String() string {
	return fmt.Sprintf("slot [%d]", m.slot)
}

func (m ShopRecharge) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteShort(m.slot)
		return w.Bytes()
	}
}

func (m *ShopRecharge) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.slot = r.ReadUint16()
	}
}
