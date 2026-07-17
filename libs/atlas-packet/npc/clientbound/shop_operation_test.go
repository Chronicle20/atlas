package clientbound

import (
	"bytes"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// Per-mode discrete level-requirement + generic-error arms. OVER/UNDER level
// requirement = dispatcher cases 14/15 (Decode1 mode + Decode4 level). GenericError
// (no reason) + GenericErrorWithReason = case 17 (Decode1 mode + Decode1 hasReason
// + optional DecodeStr reason); GenericErrorWithReason mode = 17 in gms_v83/v84/v87,
// 19 in gms_v95, VERSION-ABSENT in jms_v185 (jms case 0x13 has no with-reason arm).
// packet-audit:verify packet=npc/clientbound/NpcShopOperationOverLevelRequirement version=gms_v83 ida=0x756da7
// packet-audit:verify packet=npc/clientbound/NpcShopOperationOverLevelRequirement version=gms_v84 ida=0x77905b
// packet-audit:verify packet=npc/clientbound/NpcShopOperationOverLevelRequirement version=gms_v87 ida=0x7a290d
// packet-audit:verify packet=npc/clientbound/NpcShopOperationOverLevelRequirement version=gms_v95 ida=0x6eb7d0
// packet-audit:verify packet=npc/clientbound/NpcShopOperationOverLevelRequirement version=jms_v185 ida=0x7cb04e
// packet-audit:verify packet=npc/clientbound/NpcShopOperationUnderLevelRequirement version=gms_v83 ida=0x756da7
// packet-audit:verify packet=npc/clientbound/NpcShopOperationUnderLevelRequirement version=gms_v84 ida=0x77905b
// packet-audit:verify packet=npc/clientbound/NpcShopOperationUnderLevelRequirement version=gms_v87 ida=0x7a290d
// packet-audit:verify packet=npc/clientbound/NpcShopOperationUnderLevelRequirement version=gms_v95 ida=0x6eb7d0
// packet-audit:verify packet=npc/clientbound/NpcShopOperationUnderLevelRequirement version=jms_v185 ida=0x7cb04e
// packet-audit:verify packet=npc/clientbound/NpcShopOperationGenericError version=gms_v83 ida=0x756da7
// packet-audit:verify packet=npc/clientbound/NpcShopOperationGenericError version=gms_v84 ida=0x77905b
// packet-audit:verify packet=npc/clientbound/NpcShopOperationGenericError version=gms_v87 ida=0x7a290d
// packet-audit:verify packet=npc/clientbound/NpcShopOperationGenericError version=gms_v95 ida=0x6eb7d0
// packet-audit:verify packet=npc/clientbound/NpcShopOperationGenericError version=jms_v185 ida=0x7cb04e
// packet-audit:verify packet=npc/clientbound/NpcShopOperationGenericErrorWithReason version=gms_v83 ida=0x756da7
// packet-audit:verify packet=npc/clientbound/NpcShopOperationGenericErrorWithReason version=gms_v84 ida=0x77905b
// packet-audit:verify packet=npc/clientbound/NpcShopOperationGenericErrorWithReason version=gms_v87 ida=0x7a290d
// packet-audit:verify packet=npc/clientbound/NpcShopOperationGenericErrorWithReason version=gms_v95 ida=0x6eb7d0
//
// Per-mode discrete notice arms (each a single mode byte). The mode byte is the
// dispatcher discriminator from docs/packets/dispatchers/npc_shop_operation.yaml;
// the body func resolves it from the tenant template per version (the value
// itself is version-stable for all nine Simple arms). Markers per version:
// packet-audit:verify packet=npc/clientbound/NpcShopOperationOk version=gms_v83 ida=0x756da7
// packet-audit:verify packet=npc/clientbound/NpcShopOperationOk version=gms_v84 ida=0x77905b
// packet-audit:verify packet=npc/clientbound/NpcShopOperationOk version=gms_v87 ida=0x7a290d
// packet-audit:verify packet=npc/clientbound/NpcShopOperationOk version=gms_v95 ida=0x6eb7d0
// packet-audit:verify packet=npc/clientbound/NpcShopOperationOk version=jms_v185 ida=0x7cb04e
// packet-audit:verify packet=npc/clientbound/NpcShopOperationOutOfStock version=gms_v83 ida=0x756da7
// packet-audit:verify packet=npc/clientbound/NpcShopOperationOutOfStock version=gms_v84 ida=0x77905b
// packet-audit:verify packet=npc/clientbound/NpcShopOperationOutOfStock version=gms_v87 ida=0x7a290d
// packet-audit:verify packet=npc/clientbound/NpcShopOperationOutOfStock version=gms_v95 ida=0x6eb7d0
// packet-audit:verify packet=npc/clientbound/NpcShopOperationOutOfStock version=jms_v185 ida=0x7cb04e
// packet-audit:verify packet=npc/clientbound/NpcShopOperationNotEnoughMoney version=gms_v83 ida=0x756da7
// packet-audit:verify packet=npc/clientbound/NpcShopOperationNotEnoughMoney version=gms_v84 ida=0x77905b
// packet-audit:verify packet=npc/clientbound/NpcShopOperationNotEnoughMoney version=gms_v87 ida=0x7a290d
// packet-audit:verify packet=npc/clientbound/NpcShopOperationNotEnoughMoney version=gms_v95 ida=0x6eb7d0
// packet-audit:verify packet=npc/clientbound/NpcShopOperationNotEnoughMoney version=jms_v185 ida=0x7cb04e
// packet-audit:verify packet=npc/clientbound/NpcShopOperationInventoryFull version=gms_v83 ida=0x756da7
// packet-audit:verify packet=npc/clientbound/NpcShopOperationInventoryFull version=gms_v84 ida=0x77905b
// packet-audit:verify packet=npc/clientbound/NpcShopOperationInventoryFull version=gms_v87 ida=0x7a290d
// packet-audit:verify packet=npc/clientbound/NpcShopOperationInventoryFull version=gms_v95 ida=0x6eb7d0
// packet-audit:verify packet=npc/clientbound/NpcShopOperationInventoryFull version=jms_v185 ida=0x7cb04e
// packet-audit:verify packet=npc/clientbound/NpcShopOperationOutOfStock2 version=gms_v83 ida=0x756da7
// packet-audit:verify packet=npc/clientbound/NpcShopOperationOutOfStock2 version=gms_v84 ida=0x77905b
// packet-audit:verify packet=npc/clientbound/NpcShopOperationOutOfStock2 version=gms_v87 ida=0x7a290d
// packet-audit:verify packet=npc/clientbound/NpcShopOperationOutOfStock2 version=gms_v95 ida=0x6eb7d0
// packet-audit:verify packet=npc/clientbound/NpcShopOperationOutOfStock2 version=jms_v185 ida=0x7cb04e
// packet-audit:verify packet=npc/clientbound/NpcShopOperationOutOfStock3 version=gms_v83 ida=0x756da7
// packet-audit:verify packet=npc/clientbound/NpcShopOperationOutOfStock3 version=gms_v84 ida=0x77905b
// packet-audit:verify packet=npc/clientbound/NpcShopOperationOutOfStock3 version=gms_v87 ida=0x7a290d
// packet-audit:verify packet=npc/clientbound/NpcShopOperationOutOfStock3 version=gms_v95 ida=0x6eb7d0
// packet-audit:verify packet=npc/clientbound/NpcShopOperationOutOfStock3 version=jms_v185 ida=0x7cb04e
// packet-audit:verify packet=npc/clientbound/NpcShopOperationNotEnoughMoney2 version=gms_v83 ida=0x756da7
// packet-audit:verify packet=npc/clientbound/NpcShopOperationNotEnoughMoney2 version=gms_v84 ida=0x77905b
// packet-audit:verify packet=npc/clientbound/NpcShopOperationNotEnoughMoney2 version=gms_v87 ida=0x7a290d
// packet-audit:verify packet=npc/clientbound/NpcShopOperationNotEnoughMoney2 version=gms_v95 ida=0x6eb7d0
// packet-audit:verify packet=npc/clientbound/NpcShopOperationNotEnoughMoney2 version=jms_v185 ida=0x7cb04e
// packet-audit:verify packet=npc/clientbound/NpcShopOperationNeedMoreItems version=gms_v83 ida=0x756da7
// packet-audit:verify packet=npc/clientbound/NpcShopOperationNeedMoreItems version=gms_v84 ida=0x77905b
// packet-audit:verify packet=npc/clientbound/NpcShopOperationNeedMoreItems version=gms_v87 ida=0x7a290d
// packet-audit:verify packet=npc/clientbound/NpcShopOperationNeedMoreItems version=gms_v95 ida=0x6eb7d0
// packet-audit:verify packet=npc/clientbound/NpcShopOperationNeedMoreItems version=jms_v185 ida=0x7cb04e
// packet-audit:verify packet=npc/clientbound/NpcShopOperationTradeLimit version=gms_v83 ida=0x756da7
// packet-audit:verify packet=npc/clientbound/NpcShopOperationTradeLimit version=gms_v84 ida=0x77905b
// packet-audit:verify packet=npc/clientbound/NpcShopOperationTradeLimit version=gms_v87 ida=0x7a290d
// packet-audit:verify packet=npc/clientbound/NpcShopOperationTradeLimit version=gms_v95 ida=0x6eb7d0
// packet-audit:verify packet=npc/clientbound/NpcShopOperationTradeLimit version=jms_v185 ida=0x7cb04e
// The CShopDlg::OnPacket arm bodies are version-stable (no version gate in any
// arm), so the golden bytes below hold across every variant. Each test asserts
// the exact wire bytes (not just encode/decode symmetry) and then round-trips.

func TestShopOperationOk(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	// OK arm = dispatcher mode 0 (npc_shop_operation.yaml: OK = 0 all versions).
	input := NewShopOperationOk(0x00)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			b := input.Encode(l, ctx)(nil)
			if want := []byte{0x00}; !bytes.Equal(b, want) {
				t.Fatalf("Ok body: got % x, want % x", b, want)
			}
			output := ShopOperationOk{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

func TestShopOperationOutOfStock(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	// OUT_OF_STOCK arm = dispatcher mode 1 (yaml: OUT_OF_STOCK = 1 all versions).
	input := NewShopOperationOutOfStock(0x01)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			b := input.Encode(l, ctx)(nil)
			if want := []byte{0x01}; !bytes.Equal(b, want) {
				t.Fatalf("OutOfStock body: got % x, want % x", b, want)
			}
			output := ShopOperationOutOfStock{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

func TestShopOperationNotEnoughMoney(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	// NOT_ENOUGH_MONEY arm = dispatcher mode 2 (yaml all versions).
	input := NewShopOperationNotEnoughMoney(0x02)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			b := input.Encode(l, ctx)(nil)
			if want := []byte{0x02}; !bytes.Equal(b, want) {
				t.Fatalf("NotEnoughMoney body: got % x, want % x", b, want)
			}
			output := ShopOperationNotEnoughMoney{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

func TestShopOperationInventoryFull(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	// INVENTORY_FULL arm = dispatcher mode 3 (yaml all versions).
	input := NewShopOperationInventoryFull(0x03)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			b := input.Encode(l, ctx)(nil)
			if want := []byte{0x03}; !bytes.Equal(b, want) {
				t.Fatalf("InventoryFull body: got % x, want % x", b, want)
			}
			output := ShopOperationInventoryFull{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

func TestShopOperationOutOfStock2(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	// OUT_OF_STOCK_2 arm = dispatcher mode 5 (yaml all versions).
	input := NewShopOperationOutOfStock2(0x05)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			b := input.Encode(l, ctx)(nil)
			if want := []byte{0x05}; !bytes.Equal(b, want) {
				t.Fatalf("OutOfStock2 body: got % x, want % x", b, want)
			}
			output := ShopOperationOutOfStock2{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

func TestShopOperationOutOfStock3(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	// OUT_OF_STOCK_3 arm = dispatcher mode 9 (yaml all versions).
	input := NewShopOperationOutOfStock3(0x09)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			b := input.Encode(l, ctx)(nil)
			if want := []byte{0x09}; !bytes.Equal(b, want) {
				t.Fatalf("OutOfStock3 body: got % x, want % x", b, want)
			}
			output := ShopOperationOutOfStock3{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

func TestShopOperationNotEnoughMoney2(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	// NOT_ENOUGH_MONEY_2 arm = dispatcher mode 10 (yaml all versions).
	input := NewShopOperationNotEnoughMoney2(0x0A)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			b := input.Encode(l, ctx)(nil)
			if want := []byte{0x0A}; !bytes.Equal(b, want) {
				t.Fatalf("NotEnoughMoney2 body: got % x, want % x", b, want)
			}
			output := ShopOperationNotEnoughMoney2{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

func TestShopOperationNeedMoreItems(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	// NEED_MORE_ITEMS arm = dispatcher mode 13 (yaml all versions).
	input := NewShopOperationNeedMoreItems(0x0D)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			b := input.Encode(l, ctx)(nil)
			if want := []byte{0x0D}; !bytes.Equal(b, want) {
				t.Fatalf("NeedMoreItems body: got % x, want % x", b, want)
			}
			output := ShopOperationNeedMoreItems{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

func TestShopOperationTradeLimit(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	// TRADE_LIMIT arm = dispatcher mode 16 (yaml all versions).
	input := NewShopOperationTradeLimit(0x10)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			b := input.Encode(l, ctx)(nil)
			if want := []byte{0x10}; !bytes.Equal(b, want) {
				t.Fatalf("TradeLimit body: got % x, want % x", b, want)
			}
			output := ShopOperationTradeLimit{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

func TestShopOperationOverLevelRequirement(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	// OVER_LEVEL_REQUIREMENT arm = dispatcher mode 14 (yaml all versions).
	input := NewShopOperationOverLevelRequirement(0x0E, 200)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			b := input.Encode(l, ctx)(nil)
			// mode(0x0E) + Encode4(levelLimit=200) uint32 LE = C8 00 00 00.
			if want := []byte{0x0E, 0xC8, 0x00, 0x00, 0x00}; !bytes.Equal(b, want) {
				t.Fatalf("OverLevelRequirement body: got % x, want % x", b, want)
			}
			output := ShopOperationOverLevelRequirement{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

func TestShopOperationUnderLevelRequirement(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	// UNDER_LEVEL_REQUIREMENT arm = dispatcher mode 15 (yaml all versions).
	input := NewShopOperationUnderLevelRequirement(0x0F, 200)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			b := input.Encode(l, ctx)(nil)
			// mode(0x0F) + Encode4(levelLimit=200) uint32 LE = C8 00 00 00.
			if want := []byte{0x0F, 0xC8, 0x00, 0x00, 0x00}; !bytes.Equal(b, want) {
				t.Fatalf("UnderLevelRequirement body: got % x, want % x", b, want)
			}
			output := ShopOperationUnderLevelRequirement{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

func TestShopOperationGenericError(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	// GENERIC_ERROR arm = dispatcher mode 17 (yaml all versions); hasReason=false.
	input := NewShopOperationGenericError(0x11)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			b := input.Encode(l, ctx)(nil)
			// mode(0x11) + hasReason bool(0x00); no reason string follows.
			if want := []byte{0x11, 0x00}; !bytes.Equal(b, want) {
				t.Fatalf("GenericError(no reason) body: got % x, want % x", b, want)
			}
			output := ShopOperationGenericError{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

func TestShopOperationGenericErrorWithReason(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	// GENERIC_ERROR_WITH_REASON arm = dispatcher mode 17 (gms_v83/v84/v87), 19 (gms_v95).
	// jms_v185 has no with-reason arm; the GMS wire shape is what is verified here.
	input := NewShopOperationGenericErrorWithReason(0x11, "test error")
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			b := input.Encode(l, ctx)(nil)
			// mode(0x11) + hasReason(0x01) + EncodeStr("test error"):
			// uint16 LE length 0x0A 0x00 then the 10 ASCII bytes.
			want := []byte{0x11, 0x01, 0x0A, 0x00, 't', 'e', 's', 't', ' ', 'e', 'r', 'r', 'o', 'r'}
			if !bytes.Equal(b, want) {
				t.Fatalf("GenericError(with reason) body: got % x, want % x", b, want)
			}
			output := ShopOperationGenericErrorWithReason{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}
