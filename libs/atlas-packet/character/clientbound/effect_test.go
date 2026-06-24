package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestEffectSimpleByteOutput is the v83 golden-byte fixture for the mode-only
// OnEffect arms (LevelUp/JobChanged/QuestClear/MonsterBookCardGet/...). These
// cases (CUser::OnEffect v83 @0x9377d9 cases 0/8/9/13/...) read ONLY the leading
// Decode1 effect-mode byte and play a client-side animation — no further wire
// fields. EffectSimple.Encode writes exactly that one byte.
//
// EffectSimple shares the CUser::OnEffect demux with EffectQuest/EffectSkillUse;
// the EffectQuest op-cell grades worst-of all three, so this sibling carries its
// own v83 marker+fixture+evidence to let the demux promote.
//
// packet-audit:verify packet=character/clientbound/EffectSimple version=gms_v83 ida=0x9377d9
// packet-audit:verify packet=character/clientbound/EffectSimple version=gms_v84 ida=0x96ea92
// packet-audit:verify packet=character/clientbound/EffectSimple version=jms_v185 ida=0x9f6395
func TestEffectSimpleByteOutput(t *testing.T) {
	v83 := pt.Variants[1] // GMS v83
	ctx := pt.CreateContext(v83.Region, v83.MajorVersion, v83.MinorVersion)

	// self: mode 0 (LevelUp) -> single mode byte (Decode1) /*0x9377ec*/
	gotSelf := NewEffectSimple(0).Encode(nil, ctx)(nil)
	if wantSelf := []byte{0x00}; !bytes.Equal(gotSelf, wantSelf) {
		t.Errorf("self bytes: got %x want %x", gotSelf, wantSelf)
	}

	// foreign: characterId prefix (read by CUserPool::OnUserRemotePacket) + mode byte
	gotForeign := NewEffectSimpleForeign(0x12345678, 8).Encode(nil, ctx)(nil)
	if wantForeign := []byte{0x78, 0x56, 0x34, 0x12, 0x08}; !bytes.Equal(gotForeign, wantForeign) {
		t.Errorf("foreign bytes: got %x want %x", gotForeign, wantForeign)
	}
}

// TestEffectSimpleByteOutputV84 is the v84 golden-byte fixture for the mode-only
// OnEffect arms. The read order is byte-identical to v83 (v84 body ≡ v83 below
// ~0x3D, IDA-confirmed): CUser::OnEffect (v84 @0x96ea92) case 0 (@0x96eac9) reads
// ONLY the leading Decode1 effect-mode byte (@0x96eaa5) and plays a client-side
// animation — no further wire fields. EffectSimple.Encode writes exactly that byte.
//
// EffectSimple shares the CUser::OnEffect demux with EffectQuest/EffectSkillUse;
// the EffectQuest op-cell grades worst-of all three, so this sibling carries its
// own v84 marker+fixture+evidence to let the demux promote.
func TestEffectSimpleByteOutputV84(t *testing.T) {
	v84 := pt.Variants[5] // GMS v84
	ctx := pt.CreateContext(v84.Region, v84.MajorVersion, v84.MinorVersion)

	// self: mode 0 (LevelUp) -> single mode byte (Decode1) /*0x96eaa5*/
	gotSelf := NewEffectSimple(0).Encode(nil, ctx)(nil)
	if wantSelf := []byte{0x00}; !bytes.Equal(gotSelf, wantSelf) {
		t.Errorf("self v84 bytes: got %x want %x", gotSelf, wantSelf)
	}

	// foreign: characterId prefix (read by CUserPool::OnUserRemotePacket) + mode byte
	gotForeign := NewEffectSimpleForeign(0x12345678, 8).Encode(nil, ctx)(nil)
	if wantForeign := []byte{0x78, 0x56, 0x34, 0x12, 0x08}; !bytes.Equal(gotForeign, wantForeign) {
		t.Errorf("foreign v84 bytes: got %x want %x", gotForeign, wantForeign)
	}
}

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
