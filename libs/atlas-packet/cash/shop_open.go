package cash

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CashShopOpenWriter = "CashShopOpen"

// CashShopOpen takes pre-encoded character info bytes from the service layer.
// The character info encoding (WriteCharacterInfo) depends on the full character model
// and stays in the service layer.
type CashShopOpen struct {
	characterInfoBytes []byte
	accountName        string
}

func NewCashShopOpen(characterInfoBytes []byte, accountName string) CashShopOpen {
	return CashShopOpen{characterInfoBytes: characterInfoBytes, accountName: accountName}
}

func (m CashShopOpen) Operation() string { return CashShopOpenWriter }
func (m CashShopOpen) String() string {
	return fmt.Sprintf("account [%s]", m.accountName)
}

func (m CashShopOpen) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteByteArray(m.characterInfoBytes)

		if t.Region() == "GMS" {
			w.WriteBool(true) // bCashShopAuthorized
			w.WriteAsciiString(m.accountName)
		} else if t.Region() == "JMS" {
			w.WriteAsciiString(m.accountName)
		}

		// CWvsContext::SetSaleInfo
		if t.Region() == "GMS" {
			if t.MajorVersion() <= 12 {
				w.WriteShort(0) // nNotSaleCount
			} else {
				w.WriteInt(0) // nNotSaleCount
			}
		}

		if (t.Region() == "GMS" && t.MajorVersion() > 12) || t.Region() == "JMS" {
			w.WriteShort(0) // special cash items
		}

		if t.Region() == "JMS" {
			w.WriteShort(0)
		}

		if (t.Region() == "GMS" && t.MajorVersion() > 12) || t.Region() == "JMS" {
			w.WriteByte(0) // category discounts
		}

		// Decode Best
		var categories uint32 = 8
		if (t.Region() == "GMS" && t.MajorVersion() > 12) || t.Region() == "JMS" {
			categories = 9
		}

		cd := []uint32{50200004, 50200069, 50200117, 50100008, 50000047}
		for i := uint32(0); i < categories; i++ {
			for j := uint32(0); j < 2; j++ {
				for _, ci := range cd {
					w.WriteInt(i)
					w.WriteInt(j)
					w.WriteInt(ci)
				}
			}
		}

		// CCashShop::DecodeStock
		w.WriteShort(0)

		// CCashShop::DecodeLimitGoods
		if (t.Region() == "GMS" && t.MajorVersion() > 12) || t.Region() == "JMS" {
			w.WriteShort(0)
		}

		// CCashShop::DecodeZeroGoods
		if t.Region() == "GMS" && t.MajorVersion() > 12 {
			w.WriteShort(0)
		}

		if (t.Region() == "GMS" && t.MajorVersion() > 12) || t.Region() == "JMS" {
			w.WriteBool(false) // bEventOn

			if t.Region() == "GMS" {
				w.WriteInt(200) // nHighestCharacterLevelInThisAccount
			}
		}
		return w.Bytes()
	}
}

func (m *CashShopOpen) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		// No-op: CashShopOpen is server-send-only with pre-encoded character info.
	}
}
