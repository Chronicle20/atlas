package writer

import (
	"atlas-channel/account"
	"atlas-channel/buddylist"
	"atlas-channel/character"
	"atlas-channel/socket/model"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CashShopOpen = "CashShopOpen"

func Skip(amount int) func(w *response.Writer) {
	return func(w *response.Writer) {
		ba := make([]byte, 0)
		for i := 0; i < amount; i++ {
			ba = append(ba, 0)
		}
		w.WriteByteArray(ba)
	}
}

func CashShopOpenBody(l logrus.FieldLogger) func(tenant tenant.Model, a account.Model, c character.Model, bl buddylist.Model) BodyProducer {
	return func(tenant tenant.Model, a account.Model, c character.Model, bl buddylist.Model) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			WriteCharacterInfo(tenant)(w)(c, bl)

			if tenant.Region() == "GMS" {
				var bCashShopAuthorized = true
				w.WriteBool(bCashShopAuthorized)
				if bCashShopAuthorized {
					w.WriteAsciiString(a.Name())
				}
			} else if tenant.Region() == "JMS" {
				w.WriteAsciiString(a.Name())
			}

			// CWvsContext::SetSaleInfo
			if tenant.Region() == "GMS" {
				var nNotSaleCount = uint32(0)
				if tenant.MajorVersion() <= 12 {
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

			if (tenant.Region() == "GMS" && tenant.MajorVersion() > 12) || tenant.Region() == "JMS" {
				var scis []model.SpecialCashItem
				w.WriteShort(uint16(len(scis)))
				for _, sci := range scis {
					sci.Encode(l, tenant, options)(w)
				}
			}

			if tenant.Region() == "JMS" {
				w.WriteShort(0)
				//w.WriteInt(0)
				//w.WriteAsciiString("")
			}

			if (tenant.Region() == "GMS" && tenant.MajorVersion() > 12) || tenant.Region() == "JMS" {
				var cds []model.CategoryDiscount
				w.WriteByte(byte(len(cds)))
				for _, cd := range cds {
					cd.Encode(l, tenant, options)(w)
				}
			}

			// Decode Best
			// TODO figure out why this does this so many times
			var categories uint32 = 8
			if (tenant.Region() == "GMS" && tenant.MajorVersion() > 12) || tenant.Region() == "JMS" {
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
			if (tenant.Region() == "GMS" && tenant.MajorVersion() > 12) || tenant.Region() == "JMS" {
				w.WriteShort(0)
			}

			// CCashShop::DecodeZeroGoods
			if tenant.Region() == "GMS" && tenant.MajorVersion() > 12 {
				w.WriteShort(0)
			}

			if (tenant.Region() == "GMS" && tenant.MajorVersion() > 12) || tenant.Region() == "JMS" {
				w.WriteBool(false) // bEventOn

				if tenant.Region() == "GMS" {
					w.WriteInt(200) // nHighestCharacterLevelInThisAccount
				}
			}
			return w.Bytes()
		}
	}
}
