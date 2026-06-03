package monster

import (
	"context"
	"testing"

	"atlas-messages/character"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/sirupsen/logrus/hooks/test"
)

func testCharacter(isGm bool) character.Model {
	gm := 0
	if isGm {
		gm = 1
	}
	return character.NewModelBuilder().SetId(1).SetGm(gm).SetMapId(100000000).Build()
}

func TestParseSpawnArgs(t *testing.T) {
	testCases := []struct {
		name    string
		message string
		wantOk  bool
		wantId  uint32
		wantRaw int
	}{
		{"single", "@mob spawn 100100", true, 100100, 1},
		{"with count", "@mob spawn 100100 5", true, 100100, 5},
		{"extra whitespace", "@mob spawn   100100   5", true, 100100, 5},
		{"count zero", "@mob spawn 100100 0", true, 100100, 0},
		{"kill all not matched", "@mob kill all", false, 0, 0},
		{"mobstatus not matched", "@mobstatus 100", false, 0, 0},
		{"mobclear not matched", "@mobclear", false, 0, 0},
		{"plain chat not matched", "hello world", false, 0, 0},
		{"trailing junk not matched", "@mob spawn 100100 5 extra", false, 0, 0},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			id, raw, ok := parseSpawnArgs(tc.message)
			if ok != tc.wantOk {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOk)
			}
			if !tc.wantOk {
				return
			}
			if id != tc.wantId {
				t.Errorf("id = %d, want %d", id, tc.wantId)
			}
			if raw != tc.wantRaw {
				t.Errorf("raw = %d, want %d", raw, tc.wantRaw)
			}
		})
	}
}

func TestNormalizeCount(t *testing.T) {
	testCases := []struct {
		name       string
		raw        int
		wantCount  int
		wantCapped bool
		wantValid  bool
	}{
		{"one", 1, 1, false, true},
		{"mid", 5, 5, false, true},
		{"at cap", 20, 20, false, true},
		{"over cap", 21, 20, true, true},
		{"zero invalid", 0, 0, false, false},
		{"negative invalid", -3, 0, false, false},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			count, capped, valid := normalizeCount(tc.raw)
			if count != tc.wantCount || capped != tc.wantCapped || valid != tc.wantValid {
				t.Errorf("normalizeCount(%d) = (%d, %v, %v), want (%d, %v, %v)",
					tc.raw, count, capped, valid, tc.wantCount, tc.wantCapped, tc.wantValid)
			}
		})
	}
}

func TestMobSpawnCommandProducer_GmGate(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := context.Background()
	f := field.NewBuilder(1, 1, 100000000).Build()

	testCases := []struct {
		name        string
		isGm        bool
		message     string
		expectFound bool
	}{
		{"GM spawn matches", true, "@mob spawn 100100", true},
		{"GM spawn with count matches", true, "@mob spawn 100100 5", true},
		{"GM count zero still matches (executor reports error)", true, "@mob spawn 100100 0", true},
		{"non-GM does not match", false, "@mob spawn 100100", false},
		{"GM kill all does not match this producer", true, "@mob kill all", false},
		{"GM plain chat does not match", true, "hi", false},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			char := testCharacter(tc.isGm)
			executor, found := MobSpawnCommandProducer(logger)(ctx)(f, char, tc.message)
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
