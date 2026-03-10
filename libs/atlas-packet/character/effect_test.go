package character

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestEffectSimpleRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewEffectSimple(3)
			output := EffectSimple{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

func TestEffectSimpleForeignRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewEffectSimpleForeign(12345, 3)
			output := EffectSimpleForeign{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.CharacterId() != input.CharacterId() {
				t.Errorf("characterId: got %v, want %v", output.CharacterId(), input.CharacterId())
			}
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

func TestEffectSkillAffectedRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewEffectSkillAffected(1, 1001004, 10)
			output := EffectSkillAffected{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.SkillId() != input.SkillId() {
				t.Errorf("skillId: got %v, want %v", output.SkillId(), input.SkillId())
			}
			if output.SkillLevel() != input.SkillLevel() {
				t.Errorf("skillLevel: got %v, want %v", output.SkillLevel(), input.SkillLevel())
			}
		})
	}
}

func TestEffectPetRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewEffectPet(5, 2, 1)
			output := EffectPet{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.EffectType() != input.EffectType() {
				t.Errorf("effectType: got %v, want %v", output.EffectType(), input.EffectType())
			}
			if output.PetIndex() != input.PetIndex() {
				t.Errorf("petIndex: got %v, want %v", output.PetIndex(), input.PetIndex())
			}
		})
	}
}

func TestEffectWithIdRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewEffectWithId(2, 2022007)
			output := EffectWithId{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.Id() != input.Id() {
				t.Errorf("id: got %v, want %v", output.Id(), input.Id())
			}
		})
	}
}

func TestEffectWithMessageRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewEffectWithMessage(15, "effect/showIntro")
			output := EffectWithMessage{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.Message() != input.Message() {
				t.Errorf("message: got %v, want %v", output.Message(), input.Message())
			}
		})
	}
}

func TestEffectProtectOnDieSafetyCharmRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewEffectProtectOnDie(9, true, 5, 30, 0)
			output := EffectProtectOnDie{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.SafetyCharm() != input.SafetyCharm() {
				t.Errorf("safetyCharm: got %v, want %v", output.SafetyCharm(), input.SafetyCharm())
			}
			if output.UsesRemaining() != input.UsesRemaining() {
				t.Errorf("usesRemaining: got %v, want %v", output.UsesRemaining(), input.UsesRemaining())
			}
			if output.Days() != input.Days() {
				t.Errorf("days: got %v, want %v", output.Days(), input.Days())
			}
		})
	}
}

func TestEffectProtectOnDieItemRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewEffectProtectOnDie(9, false, 3, 7, 5130000)
			output := EffectProtectOnDie{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.SafetyCharm() != input.SafetyCharm() {
				t.Errorf("safetyCharm: got %v, want %v", output.SafetyCharm(), input.SafetyCharm())
			}
			if output.ItemId() != input.ItemId() {
				t.Errorf("itemId: got %v, want %v", output.ItemId(), input.ItemId())
			}
		})
	}
}

func TestEffectIncDecHPRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewEffectIncDecHP(10, -5)
			output := EffectIncDecHP{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.Delta() != input.Delta() {
				t.Errorf("delta: got %v, want %v", output.Delta(), input.Delta())
			}
		})
	}
}

func TestEffectShowInfoRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewEffectShowInfo(17, "Map/Effect.img/quest/party/clear")
			output := EffectShowInfo{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.Path() != input.Path() {
				t.Errorf("path: got %v, want %v", output.Path(), input.Path())
			}
		})
	}
}

func TestEffectLotteryUseSuccessRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewEffectLotteryUse(14, 5220000, true, "You won!")
			output := EffectLotteryUse{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.ItemId() != input.ItemId() {
				t.Errorf("itemId: got %v, want %v", output.ItemId(), input.ItemId())
			}
			if output.Success() != input.Success() {
				t.Errorf("success: got %v, want %v", output.Success(), input.Success())
			}
			if output.Message() != input.Message() {
				t.Errorf("message: got %v, want %v", output.Message(), input.Message())
			}
		})
	}
}

func TestEffectLotteryUseFailureRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewEffectLotteryUse(14, 5220000, false, "")
			output := EffectLotteryUse{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Success() != input.Success() {
				t.Errorf("success: got %v, want %v", output.Success(), input.Success())
			}
		})
	}
}

func TestEffectItemMakerRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewEffectItemMaker(13, 1)
			output := EffectItemMaker{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.State() != input.State() {
				t.Errorf("state: got %v, want %v", output.State(), input.State())
			}
		})
	}
}

func TestEffectUpgradeTombRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewEffectUpgradeTomb(12, 7)
			output := EffectUpgradeTomb{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.UsesRemaining() != input.UsesRemaining() {
				t.Errorf("usesRemaining: got %v, want %v", output.UsesRemaining(), input.UsesRemaining())
			}
		})
	}
}

func TestEffectIncubatorUseRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewEffectIncubatorUse(16, 5000028, "A new pet appeared!")
			output := EffectIncubatorUse{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.ItemId() != input.ItemId() {
				t.Errorf("itemId: got %v, want %v", output.ItemId(), input.ItemId())
			}
			if output.Message() != input.Message() {
				t.Errorf("message: got %v, want %v", output.Message(), input.Message())
			}
		})
	}
}
