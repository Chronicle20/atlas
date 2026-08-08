package skill_test

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/pet/skill"
)

func TestAllOrderAndBits(t *testing.T) {
	// Order and bit assignment are Atlas-canonical storage semantics (design §3.5):
	// the nine 0519 WZ spec keys, bits 1<<0 .. 1<<8 in that order.
	want := []struct {
		key skill.Key
		bit skill.Flag
	}{
		{skill.PickupItem, 1 << 0},
		{skill.ConsumeHP, 1 << 1},
		{skill.LongRange, 1 << 2},
		{skill.DropSweep, 1 << 3},
		{skill.PickupAll, 1 << 4},
		{skill.IgnorePickup, 1 << 5},
		{skill.ConsumeMP, 1 << 6},
		{skill.Recall, 1 << 7},
		{skill.AutoSpeaking, 1 << 8},
	}
	all := skill.All()
	if len(all) != len(want) {
		t.Fatalf("All() len = %d, want %d", len(all), len(want))
	}
	for i, w := range want {
		if all[i] != w.key {
			t.Errorf("All()[%d] = %q, want %q", i, all[i], w.key)
		}
		bit, ok := skill.BitFor(w.key)
		if !ok || bit != w.bit {
			t.Errorf("BitFor(%q) = %v,%v, want %v,true", w.key, bit, ok, w.bit)
		}
	}
}

func TestBitForUnknown(t *testing.T) {
	if _, ok := skill.BitFor(skill.Key("bogus")); ok {
		t.Error("BitFor(bogus) ok = true, want false")
	}
}

func TestHasApply(t *testing.T) {
	var f uint16
	f = skill.Apply(f, skill.ConsumeHP, true)
	if !skill.Has(f, skill.ConsumeHP) {
		t.Error("Has(consumeHP) = false after Apply(true)")
	}
	if skill.Has(f, skill.ConsumeMP) {
		t.Error("Has(consumeMP) = true, want false")
	}
	// idempotent set
	if skill.Apply(f, skill.ConsumeHP, true) != f {
		t.Error("Apply(true) not idempotent")
	}
	f = skill.Apply(f, skill.ConsumeHP, false)
	if skill.Has(f, skill.ConsumeHP) {
		t.Error("Has(consumeHP) = true after Apply(false)")
	}
	// Apply on unknown key is a no-op
	if skill.Apply(f, skill.Key("bogus"), true) != f {
		t.Error("Apply(unknown) mutated the mask")
	}
}
