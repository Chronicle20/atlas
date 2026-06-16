package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// packet-audit:verify packet=npc/clientbound/NpcShopOperationGenericError version=gms_v83 ida=0x756da7
// packet-audit:verify packet=npc/clientbound/NpcShopOperationGenericError version=gms_v87 ida=0x7a290d
// packet-audit:verify packet=npc/clientbound/NpcShopOperationGenericError version=gms_v95 ida=0x6eb7d0
// packet-audit:verify packet=npc/clientbound/NpcShopOperationSimple version=gms_v83 ida=0x756da7
// packet-audit:verify packet=npc/clientbound/NpcShopOperationLevelRequirement version=gms_v83 ida=0x756da7
// packet-audit:verify packet=npc/clientbound/NpcShopOperationSimple version=gms_v87 ida=0x7a290d
// packet-audit:verify packet=npc/clientbound/NpcShopOperationLevelRequirement version=gms_v87 ida=0x7a290d
// packet-audit:verify packet=npc/clientbound/NpcShopOperationSimple version=gms_v95 ida=0x6eb7d0
// packet-audit:verify packet=npc/clientbound/NpcShopOperationLevelRequirement version=gms_v95 ida=0x6eb7d0
// packet-audit:verify packet=npc/clientbound/NpcShopOperationGenericError version=jms_v185 ida=0x7cb04e
// packet-audit:verify packet=npc/clientbound/NpcShopOperationLevelRequirement version=jms_v185 ida=0x7cb04e
// packet-audit:verify packet=npc/clientbound/NpcShopOperationSimple version=jms_v185 ida=0x7cb04e
// packet-audit:verify packet=npc/clientbound/NpcShopOperationGenericError version=gms_v84 ida=0x77905b
// packet-audit:verify packet=npc/clientbound/NpcShopOperationLevelRequirement version=gms_v84 ida=0x77905b
// packet-audit:verify packet=npc/clientbound/NpcShopOperationSimple version=gms_v84 ida=0x77905b
// The CShopDlg::OnPacket arm bodies are version-stable (no version gate in any
// arm), so the golden bytes below hold across every variant. Each test asserts
// the exact wire bytes (not just encode/decode symmetry) and then round-trips.

func TestShopOperationSimple(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := NewShopOperationSimple(0x0E)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			b := input.Encode(l, ctx)(nil)
			// Simple arm = the dispatcher mode byte and nothing else.
			if want := []byte{0x0E}; !bytes.Equal(b, want) {
				t.Fatalf("Simple body: got % x, want % x", b, want)
			}
			output := ShopOperationSimple{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

func TestShopOperationGenericError(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := NewShopOperationGenericError(11)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			b := input.Encode(l, ctx)(nil)
			// mode(0x0B) + hasReason bool(0x00); no reason string follows.
			if want := []byte{0x0B, 0x00}; !bytes.Equal(b, want) {
				t.Fatalf("GenericError(no reason) body: got % x, want % x", b, want)
			}
			output := ShopOperationGenericError{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

func TestShopOperationGenericErrorWithReason(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := NewShopOperationGenericErrorWithReason(12, "test error")
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			b := input.Encode(l, ctx)(nil)
			// mode(0x0C) + hasReason(0x01) + EncodeStr("test error"):
			// uint16 LE length 0x0A 0x00 then the 10 ASCII bytes.
			want := []byte{0x0C, 0x01, 0x0A, 0x00, 't', 'e', 's', 't', ' ', 'e', 'r', 'r', 'o', 'r'}
			if !bytes.Equal(b, want) {
				t.Fatalf("GenericError(with reason) body: got % x, want % x", b, want)
			}
			output := ShopOperationGenericError{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

func TestShopOperationLevelRequirement(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := NewShopOperationLevelRequirement(9, 200)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			b := input.Encode(l, ctx)(nil)
			// mode(0x09) + Encode4(levelLimit=200) uint32 LE = C8 00 00 00.
			if want := []byte{0x09, 0xC8, 0x00, 0x00, 0x00}; !bytes.Equal(b, want) {
				t.Fatalf("LevelRequirement body: got % x, want % x", b, want)
			}
			output := ShopOperationLevelRequirement{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}
