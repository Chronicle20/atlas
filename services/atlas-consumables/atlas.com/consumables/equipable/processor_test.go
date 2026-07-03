package equipable

import (
	"testing"

	"atlas-consumables/asset"

	"github.com/google/uuid"
)

func TestAddHammersAppliedChange(t *testing.T) {
	b := asset.NewBuilder(uuid.New(), 1302000).SetHammersApplied(1)
	AddHammersApplied(1)(b)
	if got := b.Build().HammersApplied(); got != 2 {
		t.Errorf("hammersApplied: got %d, want 2", got)
	}
}
