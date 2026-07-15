package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/clientbound/ItemUpgrade version=jms_v185 ida=0x9f1a92
// packet-audit:verify packet=character/clientbound/ItemUpgrade version=gms_v83 ida=0x93354d
// packet-audit:verify packet=character/clientbound/ItemUpgrade version=gms_v87 ida=0x9adb79
// packet-audit:verify packet=character/clientbound/ItemUpgrade version=gms_v95 ida=0x8e7b00
// packet-audit:verify packet=character/clientbound/ItemUpgrade version=gms_v84 ida=0x96a87f
// packet-audit:verify packet=character/clientbound/ItemUpgrade version=gms_v79 ida=0x88d1c4
func TestItemUpgradeRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewItemUpgrade(12345, true, false, true, false)
			output := ItemUpgrade{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.CharacterId() != input.CharacterId() {
				t.Errorf("characterId: got %v, want %v", output.CharacterId(), input.CharacterId())
			}
			if output.Success() != input.Success() {
				t.Errorf("success: got %v, want %v", output.Success(), input.Success())
			}
			if output.Cursed() != input.Cursed() {
				t.Errorf("cursed: got %v, want %v", output.Cursed(), input.Cursed())
			}
			if output.LegendarySpirit() != input.LegendarySpirit() {
				t.Errorf("legendarySpirit: got %v, want %v", output.LegendarySpirit(), input.LegendarySpirit())
			}
			if output.WhiteScroll() != input.WhiteScroll() {
				t.Errorf("whiteScroll: got %v, want %v", output.WhiteScroll(), input.WhiteScroll())
			}
			// enchantCategory (Decode4): GMS>87 only — absent in v83/v87 and JMS v185.
			// enchantResultFlag (Decode1/v6): GMS>87 and JMS v185 — absent in v83/v87 only.
			// IDA JMS v185 CUser::ShowItemUpgradeEffect@0x9f1a92: reads Decode1×5 (no Decode4).
			hasEnchantCategory := v.Region == "GMS" && v.MajorVersion > 87
			hasEnchantResultFlag := (v.Region == "GMS" && v.MajorVersion > 87) || v.Region == "JMS"
			if hasEnchantCategory {
				if output.EnchantCategory() != input.EnchantCategory() {
					t.Errorf("enchantCategory: got %v, want %v", output.EnchantCategory(), input.EnchantCategory())
				}
			} else {
				if output.EnchantCategory() != 0 {
					t.Errorf("enchantCategory: expected 0 for v83/v87/JMS, got %v", output.EnchantCategory())
				}
			}
			if hasEnchantResultFlag {
				if output.EnchantResultFlag() != input.EnchantResultFlag() {
					t.Errorf("enchantResultFlag: got %v, want %v", output.EnchantResultFlag(), input.EnchantResultFlag())
				}
			} else {
				if output.EnchantResultFlag() != 0 {
					t.Errorf("enchantResultFlag: expected 0 for v83/v87, got %v", output.EnchantResultFlag())
				}
			}
		})
	}
}

// TestItemUpgradeByteFixtureV79 pins the exact SHOW_SCROLL_EFFECT (op 156) wire
// bytes against CUser::ShowItemUpgradeEffect (v79 @0x88d1c4). v79 is GMS-79
// (region GMS, major 79 <= 87), so it takes the pre-v95 path: no enchantCategory
// (Decode4) and no enchantResultFlag (Decode1). The client reads exactly four
// Decode1 bools after the dispatcher-consumed characterId:
//
//	characterId     = Decode4  // consumed by dispatcher before the handler
//	success (v34)   = Decode1  /*0x88d1f9*/
//	cursed  (v6/v31)= Decode1  /*0x88d203*/
//	legendarySpirit (v32) = Decode1 /*0x88d214*/
//	whiteScroll (v30) = Decode1 /*0x88d21f*/  (no further reads on v79)
func TestItemUpgradeByteFixtureV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	// characterId=12345 (0x00003039 LE), success=1, cursed=0, legendarySpirit=1, whiteScroll=0
	got := pt.Encode(t, ctx, NewItemUpgrade(12345, true, false, true, false).Encode, nil)
	want := []byte{
		0x39, 0x30, 0x00, 0x00, // characterId (dispatcher prefix)
		0x01, // success = 1        /*0x88d1f9*/
		0x00, // cursed = 0         /*0x88d203*/
		0x01, // legendarySpirit = 1 /*0x88d214*/
		0x00, // whiteScroll = 0    /*0x88d21f*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v79 bytes:\n got %x\nwant %x", got, want)
	}
}

func TestItemUpgradeEnchantRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewItemUpgradeEnchant(99999, true, false, true, 2, true, 1)
			output := ItemUpgrade{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.CharacterId() != input.CharacterId() {
				t.Errorf("characterId: got %v, want %v", output.CharacterId(), input.CharacterId())
			}
			// enchantCategory (Decode4): GMS>87 only — absent in v83/v87 and JMS v185.
			// enchantResultFlag (Decode1/v6): GMS>87 and JMS v185 — absent in v83/v87 only.
			// IDA JMS v185 CUser::ShowItemUpgradeEffect@0x9f1a92: reads Decode1×5 (no Decode4).
			hasEnchantCategory := v.Region == "GMS" && v.MajorVersion > 87
			hasEnchantResultFlag := (v.Region == "GMS" && v.MajorVersion > 87) || v.Region == "JMS"
			if hasEnchantCategory {
				if output.EnchantCategory() != input.EnchantCategory() {
					t.Errorf("enchantCategory: got %v, want %v", output.EnchantCategory(), input.EnchantCategory())
				}
			} else {
				if output.EnchantCategory() != 0 {
					t.Errorf("enchantCategory: expected 0 for v83/v87/JMS, got %v", output.EnchantCategory())
				}
			}
			if hasEnchantResultFlag {
				if output.EnchantResultFlag() != input.EnchantResultFlag() {
					t.Errorf("enchantResultFlag: got %v, want %v", output.EnchantResultFlag(), input.EnchantResultFlag())
				}
			} else {
				if output.EnchantResultFlag() != 0 {
					t.Errorf("enchantResultFlag: expected 0 for v83/v87, got %v", output.EnchantResultFlag())
				}
			}
		})
	}
}
