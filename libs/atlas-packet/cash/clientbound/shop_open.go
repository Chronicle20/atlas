package clientbound

import (
	"context"
	"fmt"

	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CashShopOpenWriter = "CashShopOpen"

type CashShopOpen struct {
	characterData charpkt.CharacterData
	accountName   string
}

func NewCashShopOpen(characterData charpkt.CharacterData, accountName string) CashShopOpen {
	return CashShopOpen{characterData: characterData, accountName: accountName}
}

func (m CashShopOpen) Operation() string { return CashShopOpenWriter }
func (m CashShopOpen) String() string {
	return fmt.Sprintf("account [%s]", m.accountName)
}

func (m CashShopOpen) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteByteArray(m.characterData.Encode(l, ctx)(options))

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

func (m *CashShopOpen) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		t := tenant.MustFromContext(ctx)

		m.characterData.Decode(l, ctx)(r, options)

		if t.Region() == "GMS" {
			_ = r.ReadBool() // bCashShopAuthorized
			m.accountName = r.ReadAsciiString()
		} else if t.Region() == "JMS" {
			m.accountName = r.ReadAsciiString()
		}

		// CWvsContext::SetSaleInfo
		if t.Region() == "GMS" {
			if t.MajorVersion() <= 12 {
				_ = r.ReadUint16() // nNotSaleCount
			} else {
				_ = r.ReadUint32() // nNotSaleCount
			}
		}

		if (t.Region() == "GMS" && t.MajorVersion() > 12) || t.Region() == "JMS" {
			_ = r.ReadUint16() // special cash items
		}

		if t.Region() == "JMS" {
			_ = r.ReadUint16()
		}

		if (t.Region() == "GMS" && t.MajorVersion() > 12) || t.Region() == "JMS" {
			_ = r.ReadByte() // category discounts
		}

		// Decode Best
		var categories uint32 = 8
		if (t.Region() == "GMS" && t.MajorVersion() > 12) || t.Region() == "JMS" {
			categories = 9
		}
		for i := uint32(0); i < categories; i++ {
			for j := uint32(0); j < 2; j++ {
				for k := 0; k < 5; k++ {
					_ = r.ReadUint32() // category
					_ = r.ReadUint32() // gender
					_ = r.ReadUint32() // commodity SN
				}
			}
		}

		// CCashShop::DecodeStock
		stockCount := r.ReadUint16()
		for i := uint16(0); i < stockCount; i++ {
			_ = r.ReadUint32() // commodity SN
			_ = r.ReadUint32() // stock state
		}

		// CCashShop::DecodeLimitGoods
		if (t.Region() == "GMS" && t.MajorVersion() > 12) || t.Region() == "JMS" {
			limitCount := r.ReadUint16()
			for i := uint16(0); i < limitCount; i++ {
				_ = r.ReadUint16() // limit size
			}
		}

		// CCashShop::DecodeZeroGoods
		if t.Region() == "GMS" && t.MajorVersion() > 12 {
			zeroCount := r.ReadUint16()
			for i := uint16(0); i < zeroCount; i++ {
				_ = r.ReadUint32() // zero goods entry
			}
		}

		if (t.Region() == "GMS" && t.MajorVersion() > 12) || t.Region() == "JMS" {
			_ = r.ReadBool() // bEventOn

			if t.Region() == "GMS" {
				_ = r.ReadUint32() // nHighestCharacterLevelInThisAccount
			}
		}
	}
}

func (m CashShopOpen) CharacterData() charpkt.CharacterData { return m.characterData }
func (m CashShopOpen) AccountName() string                  { return m.accountName }
