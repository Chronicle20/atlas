package clientbound

import (
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// JMS-only arm byte fixtures (task-183 follow-up). Every arm below exists ONLY
// in the JMS v185 dispatcher switch (CCashShop::OnCashItemResult @ 0x48b5a5),
// so each modes map carries a single JMS/v185 entry (n-a on every GMS/legacy
// version) and the fixture skips all other variants. Mode bytes = the raw
// Decode1 switch case value (jms_v185 column of cash_shop_operation.yaml).

// ---- mode 76: GIFT_RESULT_NOTICE (mode + reason byte) ----

// packet-audit:verify packet=cash/clientbound/CashGiftResultNotice version=jms_v185 ida=0x48ba24
func TestGiftResultNoticeByteFixture(t *testing.T) {
	mode := byte(76)
	errorCode := byte(214)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			if variantKey(v) != "JMS/v185" {
				t.Skipf("GIFT_RESULT_NOTICE is JMS/v185-only; skipping %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewGiftResultNotice(mode, errorCode)
			got := pt.Encode(t, ctx, input.Encode, nil)
			want := []byte{mode, errorCode}
			if !bytesEqual(got, want) {
				t.Errorf("GIFT_RESULT_NOTICE bytes: got %v, want %v", got, want)
			}
			output := GiftResultNotice{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode || output.ErrorCode() != errorCode {
				t.Errorf("round-trip mismatch: %+v", output)
			}
		})
	}
}

// ---- mode 77: LOAD_RECEIVED_GIFT_SUCCESS (mode + flag + count:u32 + N×176-byte record) ----

// packet-audit:verify packet=cash/clientbound/CashLoadReceivedGiftDone version=jms_v185 ida=0x48ba3f
func TestLoadReceivedGiftDoneByteFixture(t *testing.T) {
	mode := byte(77)
	flag := byte(1)
	gift := ReceivedGiftEntry{
		ItemId:   5220000,
		Data1:    7,
		Data2:    9,
		GiftType: 1,
		Text:     "thanks!",
		Sender:   "Alice",
		ItemName: "Cash Hat",
	}
	l, _ := testlog.NewNullLogger()
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			if variantKey(v) != "JMS/v185" {
				t.Skipf("LOAD_RECEIVED_GIFT_SUCCESS is JMS/v185-only; skipping %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewLoadReceivedGiftDone(mode, flag, []ReceivedGiftEntry{gift})
			got := pt.Encode(t, ctx, input.Encode, nil)

			record := gift.EncodeBytes(l)
			if len(record) != 176 {
				t.Fatalf("ReceivedGiftEntry must be 176 bytes, got %d", len(record))
			}
			// itemId sits at record offset 0x0C (disasm 0x48baa1 mov edi,[eax+0Ch]).
			if !bytesEqual(record[12:16], le32(nil, gift.ItemId)) {
				t.Errorf("itemId not at record offset 12: got %v", record[12:16])
			}
			want := []byte{mode, flag}
			want = le32(want, int32(1)) // count:uint32 = 1
			want = append(want, record...)
			if !bytesEqual(got, want) {
				t.Errorf("LOAD_RECEIVED_GIFT_SUCCESS bytes: got %v (len %d), want %v (len %d)", got, len(got), want, len(want))
			}

			output := LoadReceivedGiftDone{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode || output.Flag() != flag || len(output.Gifts()) != 1 {
				t.Fatalf("round-trip header mismatch: %+v", output)
			}
			g := output.Gifts()[0]
			if g.ItemId != gift.ItemId || g.GiftType != gift.GiftType || g.Text != gift.Text || g.Sender != gift.Sender || g.ItemName != gift.ItemName {
				t.Errorf("gift round-trip mismatch: got %+v, want %+v", g, gift)
			}
		})
	}
}

// ---- mode 96: LIMIT_GOODS_STOCK_CHANGED (mode + result + [205/206] itemId:u32) ----

// packet-audit:verify packet=cash/clientbound/CashLimitGoodsStockChanged version=jms_v185 ida=0x48d4d4
func TestLimitGoodsStockChangedByteFixture(t *testing.T) {
	mode := byte(96)
	itemId := uint32(5390000)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			if variantKey(v) != "JMS/v185" {
				t.Skipf("LIMIT_GOODS_STOCK_CHANGED is JMS/v185-only; skipping %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)

			// result 206 => the conditional itemId IS present on the wire.
			input := NewLimitGoodsStockChanged(mode, 206, itemId)
			got := pt.Encode(t, ctx, input.Encode, nil)
			want := le32([]byte{mode, 206}, int32(itemId))
			if !bytesEqual(got, want) {
				t.Errorf("result=206 bytes: got %v, want %v", got, want)
			}
			output := LimitGoodsStockChanged{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode || output.Result() != 206 || output.ItemId() != itemId {
				t.Errorf("result=206 round-trip mismatch: %+v", output)
			}

			// result 100 (not 205/206) => NO itemId on the wire.
			input2 := NewLimitGoodsStockChanged(mode, 100, itemId)
			got2 := pt.Encode(t, ctx, input2.Encode, nil)
			want2 := []byte{mode, 100}
			if !bytesEqual(got2, want2) {
				t.Errorf("result=100 bytes: got %v, want %v", got2, want2)
			}
			output2 := LimitGoodsStockChanged{}
			pt.RoundTrip(t, ctx, input2.Encode, output2.Decode, nil)
			if output2.Mode() != mode || output2.Result() != 100 || output2.ItemId() != 0 {
				t.Errorf("result=100 round-trip mismatch: %+v", output2)
			}
		})
	}
}

// ---- mode 146: SHOW_NOTICE_1089 (bodyless: mode only) ----

// packet-audit:verify packet=cash/clientbound/CashShowNotice1089 version=jms_v185 ida=0x48e6c9
func TestShowNotice1089ByteFixture(t *testing.T) {
	mode := byte(146)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			if variantKey(v) != "JMS/v185" {
				t.Skipf("SHOW_NOTICE_1089 is JMS/v185-only; skipping %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewShowNotice1089(mode)
			got := pt.Encode(t, ctx, input.Encode, nil)
			if !bytesEqual(got, []byte{mode}) {
				t.Errorf("SHOW_NOTICE_1089 bytes: got %v, want %v", got, []byte{mode})
			}
			output := ShowNotice1089{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode {
				t.Errorf("round-trip mismatch: %+v", output)
			}
		})
	}
}

// ---- mode 147: TRANSFER_WORLD_NOTICE_REASON (mode + reason byte) ----

// packet-audit:verify packet=cash/clientbound/CashTransferWorldNoticeReason version=jms_v185 ida=0x48e6f7
func TestTransferWorldNoticeReasonByteFixture(t *testing.T) {
	mode := byte(147)
	errorCode := byte(177)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			if variantKey(v) != "JMS/v185" {
				t.Skipf("TRANSFER_WORLD_NOTICE_REASON is JMS/v185-only; skipping %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewTransferWorldNoticeReason(mode, errorCode)
			got := pt.Encode(t, ctx, input.Encode, nil)
			want := []byte{mode, errorCode}
			if !bytesEqual(got, want) {
				t.Errorf("TRANSFER_WORLD_NOTICE_REASON bytes: got %v, want %v", got, want)
			}
			output := TransferWorldNoticeReason{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode || output.ErrorCode() != errorCode {
				t.Errorf("round-trip mismatch: %+v", output)
			}
		})
	}
}

// ---- mode 162: REFRESH_LOCKER (mode + 55-byte GW_CashItemInfo) ----

// packet-audit:verify packet=cash/clientbound/CashRefreshLocker version=jms_v185 ida=0x48c321
func TestRefreshLockerByteFixture(t *testing.T) {
	mode := byte(162)
	item := CashInventoryItem{
		CashId: 555, AccountId: 1, CharacterId: 2, TemplateId: 5040000,
		CommodityId: 300, Quantity: 1, GiftFrom: "", Expiration: 0,
	}
	l, _ := testlog.NewNullLogger()
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			if variantKey(v) != "JMS/v185" {
				t.Skipf("REFRESH_LOCKER is JMS/v185-only; skipping %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewRefreshLocker(mode, item)
			got := pt.Encode(t, ctx, input.Encode, nil)
			want := append([]byte{mode}, item.EncodeBytes(l)...)
			if !bytesEqual(got, want) {
				t.Errorf("REFRESH_LOCKER bytes: got %v, want %v", got, want)
			}
			output := RefreshLocker{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode || output.Item().TemplateId != item.TemplateId || output.Item().CashId != item.CashId {
				t.Errorf("round-trip mismatch: %+v", output)
			}
		})
	}
}

// ---- mode 164: CLIENT_NO_OP (bodyless: mode only) ----

// packet-audit:verify packet=cash/clientbound/CashClientNoOp version=jms_v185 ida=0x48c370
func TestClientNoOpByteFixture(t *testing.T) {
	mode := byte(164)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			if variantKey(v) != "JMS/v185" {
				t.Skipf("CLIENT_NO_OP is JMS/v185-only; skipping %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewClientNoOp(mode)
			got := pt.Encode(t, ctx, input.Encode, nil)
			if !bytesEqual(got, []byte{mode}) {
				t.Errorf("CLIENT_NO_OP bytes: got %v, want %v", got, []byte{mode})
			}
			output := ClientNoOp{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode {
				t.Errorf("round-trip mismatch: %+v", output)
			}
		})
	}
}

// ---- mode 166: SHOW_NOTICE_1465 (bodyless: mode only) ----

// packet-audit:verify packet=cash/clientbound/CashShowNotice1465 version=jms_v185 ida=0x48c26e
func TestShowNotice1465ByteFixture(t *testing.T) {
	mode := byte(166)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			if variantKey(v) != "JMS/v185" {
				t.Skipf("SHOW_NOTICE_1465 is JMS/v185-only; skipping %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewShowNotice1465(mode)
			got := pt.Encode(t, ctx, input.Encode, nil)
			if !bytesEqual(got, []byte{mode}) {
				t.Errorf("SHOW_NOTICE_1465 bytes: got %v, want %v", got, []byte{mode})
			}
			output := ShowNotice1465{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode {
				t.Errorf("round-trip mismatch: %+v", output)
			}
		})
	}
}

// ---- mode 167: REFRESH_LOCKER_OR_NOTICE (mode + flag + 55-byte GW_CashItemInfo) ----

// packet-audit:verify packet=cash/clientbound/CashRefreshLockerOrNotice version=jms_v185 ida=0x48c373
func TestRefreshLockerOrNoticeByteFixture(t *testing.T) {
	mode := byte(167)
	flag := byte(1)
	item := CashInventoryItem{
		CashId: 777, AccountId: 3, CharacterId: 4, TemplateId: 5041000,
		CommodityId: 400, Quantity: 1, GiftFrom: "", Expiration: 0,
	}
	l, _ := testlog.NewNullLogger()
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			if variantKey(v) != "JMS/v185" {
				t.Skipf("REFRESH_LOCKER_OR_NOTICE is JMS/v185-only; skipping %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewRefreshLockerOrNotice(mode, flag, item)
			got := pt.Encode(t, ctx, input.Encode, nil)
			want := append([]byte{mode, flag}, item.EncodeBytes(l)...)
			if !bytesEqual(got, want) {
				t.Errorf("REFRESH_LOCKER_OR_NOTICE bytes: got %v, want %v", got, want)
			}
			output := RefreshLockerOrNotice{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode || output.Flag() != flag || output.Item().TemplateId != item.TemplateId || output.Item().CashId != item.CashId {
				t.Errorf("round-trip mismatch: %+v", output)
			}
		})
	}
}

// ---- mode 168: SHOW_NOTICE_1464 (bodyless: mode only) ----

// packet-audit:verify packet=cash/clientbound/CashShowNotice1464 version=jms_v185 ida=0x48c413
func TestShowNotice1464ByteFixture(t *testing.T) {
	mode := byte(168)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			if variantKey(v) != "JMS/v185" {
				t.Skipf("SHOW_NOTICE_1464 is JMS/v185-only; skipping %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewShowNotice1464(mode)
			got := pt.Encode(t, ctx, input.Encode, nil)
			if !bytesEqual(got, []byte{mode}) {
				t.Errorf("SHOW_NOTICE_1464 bytes: got %v, want %v", got, []byte{mode})
			}
			output := ShowNotice1464{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode {
				t.Errorf("round-trip mismatch: %+v", output)
			}
		})
	}
}
