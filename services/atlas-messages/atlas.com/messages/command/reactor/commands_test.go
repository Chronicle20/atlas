package reactor

import (
	"atlas-messages/character"
	"context"
	"testing"

	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

func testCharacter(isGm bool) character.Model {
	gm := 0
	if isGm {
		gm = 1
	}
	return character.NewModelBuilder().SetId(1).SetGm(gm).Build()
}

func TestReactorDestroyAllCommandProducer_GmGate(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := context.Background()
	f := field.NewBuilder(1, 1, 610030400).Build()

	testCases := []struct {
		name        string
		isGm        bool
		message     string
		expectFound bool
	}{
		{"GM destroy all matches", true, "@reactor destroy all", true},
		{"non-GM does not match", false, "@reactor destroy all", false},
		{"GM trailing junk does not match", true, "@reactor destroy all now", false},
		{"GM missing all does not match", true, "@reactor destroy", false},
		{"GM plain chat does not match", true, "hi", false},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			char := testCharacter(tc.isGm)
			executor, found := ReactorDestroyAllCommandProducer(logger)(ctx)(f, char, tc.message)
			if found != tc.expectFound {
				t.Fatalf("found = %v, want %v", found, tc.expectFound)
			}
			if tc.expectFound && executor == nil {
				t.Error("expected non-nil executor when found")
			}
			if !tc.expectFound && executor != nil {
				t.Error("expected nil executor when not found")
			}
		})
	}
}
