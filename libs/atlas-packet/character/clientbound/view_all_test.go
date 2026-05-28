package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestCharacterViewAllCountRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := CharacterViewAllCount{code: 3, worldCount: 5, unk: 0}
			output := CharacterViewAllCount{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Code() != input.Code() {
				t.Errorf("code: got %v, want %v", output.Code(), input.Code())
			}
			if output.WorldCount() != input.WorldCount() {
				t.Errorf("worldCount: got %v, want %v", output.WorldCount(), input.WorldCount())
			}
		})
	}
}

func TestCharacterViewAllCharactersRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			stats := model.NewCharacterStatistics(
				99, "ViewAllChar", 0, 2, 20000, 30000,
				[3]uint64{10, 20, 30},
				40, 100,
				30, 25, 20, 15,
				1000, 1000, 500, 500,
				3, false, 2,
				50000, 50, 1000,
				100000000, 0,
			)
			avatar := model.NewAvatar(0, 2, 20000, false, 30000, nil, nil, nil)
			// viewAll=true: no family byte; gm=false: rank fields are written
			entry := model.NewCharacterListEntry(stats, avatar, true, false, 5, 1, 3, 2)
			input := NewCharacterViewAllCharacters(0, world.Id(0), []model.CharacterListEntry{entry})
			output := CharacterViewAllCharacters{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Code() != input.Code() {
				t.Errorf("code: got %v, want %v", output.Code(), input.Code())
			}
			if output.WorldId() != input.WorldId() {
				t.Errorf("worldId: got %v, want %v", output.WorldId(), input.WorldId())
			}
			if len(output.Characters()) != len(input.Characters()) {
				t.Errorf("characters len: got %v, want %v", len(output.Characters()), len(input.Characters()))
			}
		})
	}
}

func TestCharacterViewAllSearchFailedRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := CharacterViewAllSearchFailed{code: 4}
			output := CharacterViewAllSearchFailed{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Code() != input.Code() {
				t.Errorf("code: got %v, want %v", output.Code(), input.Code())
			}
		})
	}
}

func TestCharacterViewAllErrorRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := CharacterViewAllError{code: 5}
			output := CharacterViewAllError{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Code() != input.Code() {
				t.Errorf("code: got %v, want %v", output.Code(), input.Code())
			}
		})
	}
}
