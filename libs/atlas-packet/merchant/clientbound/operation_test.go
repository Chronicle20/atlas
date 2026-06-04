package clientbound

import (
	"encoding/binary"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func TestOpenShop(t *testing.T) {
	input := NewOpenShop(7)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestMerchantErrorSimple(t *testing.T) {
	input := NewMerchantErrorSimple(8)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestShopSearch(t *testing.T) {
	input := NewShopSearch(13, 12345)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestShopRename(t *testing.T) {
	input := NewShopRename(14, true)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestRemoteShopWarp(t *testing.T) {
	input := NewRemoteShopWarp(16, 12345, 3)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestConfirmManage(t *testing.T) {
	input := NewConfirmManage(17, 12345, 5, 9876543210)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestFreeFormNotice(t *testing.T) {
	input := NewFreeFormNotice(18, "Welcome to my shop!")
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// TestOpenShopWireShape proves the exact wire layout for mode 7 (OPEN_SHOP) of
// CWvsContext::OnEntrustedShopCheckResult (GMS v95 @ 0x9ffcb0):
//
//	Decode1 (mode byte only — client calls SendOpenShopRequest, no further reads)
//
// All versions share this single-byte layout.
func TestOpenShopWireShape(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := NewOpenShop(7)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			b := in.Encode(l, test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion))(nil)
			// 1 (mode)
			if len(b) != 1 {
				t.Fatalf("wire size = %d bytes, want 1: % x", len(b), b)
			}
			if b[0] != 0x07 {
				t.Errorf("byte[0] mode = 0x%02x, want 0x07 (OPEN_SHOP)", b[0])
			}
		})
	}
}

// TestMerchantErrorSimpleWireShape proves the exact wire layout for error modes
// (9, 10, 15) of CWvsContext::OnEntrustedShopCheckResult (GMS v95 @ 0x9ffcb0):
//
//	Decode1 (mode byte only — client shows a fixed string-pool notice)
//
// All versions share this single-byte layout.
func TestMerchantErrorSimpleWireShape(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	for _, mode := range []byte{9, 10, 15} {
		in := NewMerchantErrorSimple(mode)
		for _, v := range test.Variants {
			t.Run(v.Name, func(t *testing.T) {
				b := in.Encode(l, test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion))(nil)
				// 1 (mode)
				if len(b) != 1 {
					t.Fatalf("mode %d: wire size = %d bytes, want 1: % x", mode, len(b), b)
				}
				if b[0] != mode {
					t.Errorf("mode %d: byte[0] = 0x%02x, want 0x%02x", mode, b[0], mode)
				}
			})
		}
	}
}

// TestShopSearchWireShape proves the exact wire layout for mode 13 (SHOP_SEARCH)
// of CWvsContext::OnEntrustedShopCheckResult (GMS v95 @ 0x9ffcb0):
//
//	Decode1 (mode)
//	Decode4 (shopId — stored into CUIMiniMap::m_dwSearchedShop)
//
// All versions share this layout.
func TestShopSearchWireShape(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := NewShopSearch(13, 0xDEADBEEF)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			b := in.Encode(l, test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion))(nil)
			// 1 (mode) + 4 (shopId)
			if len(b) != 5 {
				t.Fatalf("wire size = %d bytes, want 5: % x", len(b), b)
			}
			if b[0] != 0x0D {
				t.Errorf("byte[0] mode = 0x%02x, want 0x0D (SHOP_SEARCH)", b[0])
			}
			shopId := binary.LittleEndian.Uint32(b[1:5])
			if shopId != 0xDEADBEEF {
				t.Errorf("shopId = 0x%08x, want 0xDEADBEEF", shopId)
			}
		})
	}
}

// TestShopRenameWireShape proves the exact wire layout for mode 14 (SHOP_RENAME)
// of CWvsContext::OnEntrustedShopCheckResult (GMS v95 @ 0x9ffcb0):
//
//	Decode1 (mode)
//	Decode1 (success flag — if 0 client returns; if 1 adds chat log)
//
// All versions share this layout.
func TestShopRenameWireShape(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	for _, success := range []bool{true, false} {
		in := NewShopRename(14, success)
		for _, v := range test.Variants {
			t.Run(v.Name, func(t *testing.T) {
				b := in.Encode(l, test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion))(nil)
				// 1 (mode) + 1 (success)
				if len(b) != 2 {
					t.Fatalf("wire size = %d bytes, want 2: % x", len(b), b)
				}
				if b[0] != 0x0E {
					t.Errorf("byte[0] mode = 0x%02x, want 0x0E (SHOP_RENAME)", b[0])
				}
				var wantBool byte
				if success {
					wantBool = 1
				}
				if b[1] != wantBool {
					t.Errorf("success byte = 0x%02x, want 0x%02x", b[1], wantBool)
				}
			})
		}
	}
}

// TestRemoteShopWarpWireShape proves the exact wire layout for mode 16 (REMOTE_SHOP_WARP)
// of CWvsContext::OnEntrustedShopCheckResult (GMS v95 @ 0x9ffcb0):
//
//	Decode1 (mode)
//	Decode4 (shopId)
//	Decode1 (channelId — 0xFE/0xFD/0xFF = error/unavailable; otherwise valid channel)
//
// All versions share this layout.
func TestRemoteShopWarpWireShape(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := NewRemoteShopWarp(16, 99999, 3)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			b := in.Encode(l, test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion))(nil)
			// 1 (mode) + 4 (shopId) + 1 (channelId)
			if len(b) != 6 {
				t.Fatalf("wire size = %d bytes, want 6: % x", len(b), b)
			}
			if b[0] != 0x10 {
				t.Errorf("byte[0] mode = 0x%02x, want 0x10 (REMOTE_SHOP_WARP)", b[0])
			}
			shopId := binary.LittleEndian.Uint32(b[1:5])
			if shopId != 99999 {
				t.Errorf("shopId = %d, want 99999", shopId)
			}
			if b[5] != 3 {
				t.Errorf("channelId = %d, want 3", b[5])
			}
		})
	}
}

// TestConfirmManageWireShape proves the exact wire layout for mode 17 (CONFIRM_MANAGE)
// of CWvsContext::OnEntrustedShopCheckResult (GMS v95 @ 0x9ffcb0):
//
//	Decode1        (mode)
//	Decode4        (shopId / dwCharacterID — v24)
//	Decode2        (position — v25 / slot index)
//	DecodeBuffer 8 (liCashItemSN — 8-byte serial number, stored as _LARGE_INTEGER)
//
// The client then checks if the player has a birthday set and shows a YES/NO dialog
// before building a PLAYER_INTERACTION outbound packet containing the serial number.
// All versions share this layout (data-dependent dialog does not change wire fields).
func TestConfirmManageWireShape(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := NewConfirmManage(17, 12345, 5, 9876543210)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			b := in.Encode(l, test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion))(nil)
			// 1 (mode) + 4 (shopId) + 2 (position) + 8 (serialNumber)
			if len(b) != 15 {
				t.Fatalf("wire size = %d bytes, want 15: % x", len(b), b)
			}
			if b[0] != 0x11 {
				t.Errorf("byte[0] mode = 0x%02x, want 0x11 (CONFIRM_MANAGE)", b[0])
			}
			shopId := binary.LittleEndian.Uint32(b[1:5])
			if shopId != 12345 {
				t.Errorf("shopId = %d, want 12345", shopId)
			}
			pos := binary.LittleEndian.Uint16(b[5:7])
			if pos != 5 {
				t.Errorf("position = %d, want 5", pos)
			}
			sn := binary.LittleEndian.Uint64(b[7:15])
			if sn != 9876543210 {
				t.Errorf("serialNumber = %d, want 9876543210", sn)
			}
		})
	}
}

// TestFreeFormNoticeWireShape proves the exact wire layout for mode 18 (FREE_FORM_NOTICE)
// of CWvsContext::OnEntrustedShopCheckResult (GMS v95 @ 0x9ffcb0):
//
//	Decode1   (mode)
//	Decode1   (flag — if 0 client returns immediately; atlas always sends 1)
//	DecodeStr (message — 2-byte LE length prefix + UTF-8/ShiftJIS bytes)
//
// The flag is hardcoded true (1) by the atlas encoder; a 0-flag packet would be
// accepted by Decode but would not be generated by this struct.
// All versions share this layout.
func TestFreeFormNoticeWireShape(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	msg := "Hi!"
	in := NewFreeFormNotice(18, msg)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			b := in.Encode(l, test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion))(nil)
			// 1 (mode) + 1 (flag) + 2 (len prefix) + 3 (msg bytes)
			if len(b) != 7 {
				t.Fatalf("wire size = %d bytes, want 7: % x", len(b), b)
			}
			if b[0] != 0x12 {
				t.Errorf("byte[0] mode = 0x%02x, want 0x12 (FREE_FORM_NOTICE)", b[0])
			}
			if b[1] != 0x01 {
				t.Errorf("byte[1] flag = 0x%02x, want 0x01 (always true)", b[1])
			}
			msgLen := int(binary.LittleEndian.Uint16(b[2:4]))
			if msgLen != 3 {
				t.Errorf("message length prefix = %d, want 3", msgLen)
			}
			if string(b[4:7]) != msg {
				t.Errorf("message bytes = %q, want %q", string(b[4:7]), msg)
			}
		})
	}
}

// TestEntrustedShopUnknownChannel proves the exact wire layout for mode 8
// (the "unknown channel" notice) of CWvsContext::OnEntrustedShopCheckResult
// (JMS185 @ 0xb0ee59):
//
//	Decode1   (mode == 8)
//	Decode4   (shopId — int, little-endian)
//	Decode1   (channelId)
//
// The client uses shopId + channelId to redirect the player toward the channel
// where the shop actually lives.
func TestEntrustedShopUnknownChannel(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := NewEntrustedShopUnknownChannel(123456, 5) // shopId, channelId
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			b := in.Encode(l, test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion))(nil)
			// mode(1) + shopId(4) + channelId(1) = 6 bytes.
			if len(b) != 6 {
				t.Fatalf("wire size = %d bytes, want 6: % x", len(b), b)
			}
			if b[0] != 8 {
				t.Errorf("byte[0] mode = %d, want 8", b[0])
			}
			if shopId := binary.LittleEndian.Uint32(b[1:5]); shopId != 123456 {
				t.Errorf("shopId = %d, want 123456", shopId)
			}
			if b[5] != 5 {
				t.Errorf("byte[5] channelId = %d, want 5", b[5])
			}
		})
	}
}

// TestEntrustedShopRoundTrip exercises Encode/Decode symmetry for the mode-8 emitter.
func TestEntrustedShopRoundTrip(t *testing.T) {
	input := NewEntrustedShopUnknownChannel(987654, 3)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
