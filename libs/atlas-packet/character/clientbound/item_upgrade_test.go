package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

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
