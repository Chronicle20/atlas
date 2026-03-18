package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CashShopOperationSetWishlistHandle = "CashShopOperationSetWishlistHandle"

// ShopOperationSetWishlist - CCashShop::SendSetWishList
type ShopOperationSetWishlist struct {
	serialNumbers []uint32
}

func (m ShopOperationSetWishlist) SerialNumbers() []uint32 { return m.serialNumbers }

func (m ShopOperationSetWishlist) Operation() string {
	return CashShopOperationSetWishlistHandle
}

func (m ShopOperationSetWishlist) String() string {
	return fmt.Sprintf("serialNumbers [%v]", m.serialNumbers)
}

func (m ShopOperationSetWishlist) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		for _, sn := range m.serialNumbers {
			w.WriteInt(sn)
		}
		return w.Bytes()
	}
}

func (m *ShopOperationSetWishlist) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.serialNumbers = make([]uint32, 10)
		for i := 0; i < 10; i++ {
			m.serialNumbers[i] = r.ReadUint32()
		}
	}
}
