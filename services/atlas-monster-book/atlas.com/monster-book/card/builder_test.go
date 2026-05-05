package card

import (
	"testing"

	"github.com/google/uuid"
)

func TestBuilderRejectsZeroCharacter(t *testing.T) {
	_, err := NewModelBuilder().SetCardId(2380000).SetLevel(1).Build()
	if err == nil {
		t.Fatal("expected error: characterId required")
	}
}

func TestBuilderRejectsOutOfRangeCardId(t *testing.T) {
	for _, badId := range []uint32{0, 2370000, 2390000, 2389999 + 1} {
		if _, err := NewModelBuilder().SetCharacterId(1).SetCardId(badId).SetLevel(1).Build(); err == nil {
			t.Fatalf("expected reject for cardId %d", badId)
		}
	}
}

func TestBuilderRejectsLevelOutOfRange(t *testing.T) {
	for _, l := range []uint8{0, 6, 255} {
		if _, err := NewModelBuilder().SetCharacterId(1).SetCardId(2380000).SetLevel(l).Build(); err == nil {
			t.Fatalf("expected reject for level %d", l)
		}
	}
}

func TestIsSpecialDerivation(t *testing.T) {
	cases := map[uint32]bool{
		2380000: false,
		2387999: false,
		2388000: true,
		2389999: true,
	}
	for cid, want := range cases {
		m, err := NewModelBuilder().
			SetTenantId(uuid.New()).SetCharacterId(1).SetCardId(cid).SetLevel(1).Build()
		if err != nil {
			t.Fatalf("build cid %d: %v", cid, err)
		}
		if m.IsSpecial() != want {
			t.Fatalf("cardId %d: want isSpecial=%v got %v", cid, want, m.IsSpecial())
		}
	}
}
