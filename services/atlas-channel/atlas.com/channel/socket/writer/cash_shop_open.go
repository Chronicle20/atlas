package writer

import (
	"atlas-channel/account"
	"atlas-channel/buddylist"
	"atlas-channel/character"
	"atlas-channel/socket/model"
	"context"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CashShopOpen = "CashShopOpen"

func CashShopOpenBody(a account.Model, c character.Model, bl buddylist.Model) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		t := tenant.MustFromContext(ctx)
		return func(options map[string]interface{}) []byte {
			WriteCharacterInfo(l, ctx, options)(w)(c, bl)

			if t.Region() == "GMS" {
				var bCashShopAuthorized = true
				w.WriteBool(bCashShopAuthorized)
				if bCashShopAuthorized {
					w.WriteAsciiString(a.Name())
				}
			} else if t.Region() == "JMS" {
				w.WriteAsciiString(a.Name())
			}

			// CWvsContext::SetSaleInfo
			if t.Region() == "GMS" {
				var nNotSaleCount = uint32(0)
				if t.MajorVersion() <= 12 {
					w.WriteShort(uint16(nNotSaleCount)) // nNotSaleCount
					for i := uint32(0); i < nNotSaleCount; i++ {
						w.WriteInt(0)
					}
				} else {
					w.WriteInt(nNotSaleCount) // nNotSaleCount
					for i := uint32(0); i < nNotSaleCount; i++ {
						w.WriteInt(0)
					}
				}
			}

			if (t.Region() == "GMS" && t.MajorVersion() > 12) || t.Region() == "JMS" {
				var scis []model.SpecialCashItem
				w.WriteShort(uint16(len(scis)))
				for _, sci := range scis {
					w.WriteByteArray(sci.Encoder(l, ctx)(options))
				}
			}

			if t.Region() == "JMS" {
				w.WriteShort(0)
				//w.WriteInt(0)
				//w.WriteAsciiString("")
			}

			if (t.Region() == "GMS" && t.MajorVersion() > 12) || t.Region() == "JMS" {
				var cds []model.CategoryDiscount
				w.WriteByte(byte(len(cds)))
				for _, cd := range cds {
					w.WriteByteArray(cd.Encoder(l, ctx)(options))
				}
			}

			// Decode Best
			// TODO figure out why this does this so many times
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
}
