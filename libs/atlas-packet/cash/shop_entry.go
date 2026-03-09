package cash

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CashShopEntryHandle = "CashShopEntryHandle"

// ShopEntry - CCashShop::SendEntry
type ShopEntry struct {
	updateTime uint32
}

func (m ShopEntry) UpdateTime() uint32 {
	return m.updateTime
}

func (m ShopEntry) Operation() string {
	return CashShopEntryHandle
}

func (m ShopEntry) String() string {
	return fmt.Sprintf("updateTime [%d]", m.updateTime)
}

func (m ShopEntry) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.updateTime)
		return w.Bytes()
	}
}

func (m *ShopEntry) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.updateTime = r.ReadUint32()
	}
}
