package consumable

import (
	"testing"
	"time"

	consumable3 "atlas-consumables/data/consumable"
)

func rw(itemId, count, prob uint32) consumable3.RewardModel {
	return consumable3.RewardModelBuilder().SetItemId(itemId).SetCount(count).SetProb(prob).Build()
}

func TestRollRewardSingleEntry(t *testing.T) {
	got, err := rollReward([]consumable3.RewardModel{rw(2000000, 1, 100)})
	if err != nil {
		t.Fatal(err)
	}
	if got.ItemId() != 2000000 {
		t.Fatalf("got %d, want 2000000", got.ItemId())
	}
}

func TestRollRewardSkipsZeroProb(t *testing.T) {
	// Only the second entry has weight; it must always win.
	for i := 0; i < 200; i++ {
		got, err := rollReward([]consumable3.RewardModel{rw(111, 1, 0), rw(222, 1, 5)})
		if err != nil {
			t.Fatal(err)
		}
		if got.ItemId() != 222 {
			t.Fatalf("iteration %d: got %d, want 222 (zero-prob entry must never win)", i, got.ItemId())
		}
	}
}

func TestRollRewardTotalZeroErrors(t *testing.T) {
	if _, err := rollReward([]consumable3.RewardModel{rw(1, 1, 0), rw(2, 1, 0)}); err == nil {
		t.Fatal("expected error when total prob is 0")
	}
	if _, err := rollReward(nil); err == nil {
		t.Fatal("expected error for empty reward table")
	}
}

func TestRollRewardDistribution(t *testing.T) {
	// 10:90 split over 10k rolls; the rare entry should land roughly in-band.
	const n = 10000
	rare := 0
	for i := 0; i < n; i++ {
		got, err := rollReward([]consumable3.RewardModel{rw(1, 1, 100), rw(2, 1, 900)})
		if err != nil {
			t.Fatal(err)
		}
		if got.ItemId() == 1 {
			rare++
		}
	}
	// Expected ~1000; allow a wide band to avoid flakiness.
	if rare < 700 || rare > 1300 {
		t.Fatalf("rare count %d out of expected ~1000 band [700,1300]", rare)
	}
}

func TestRewardExpiration(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	// 7200 minutes = exactly 5 days.
	got := rewardExpiration(7200, now)
	if !got.Equal(now.Add(5 * 24 * time.Hour)) {
		t.Fatalf("period=7200 → %v, want now+5d", got)
	}
	if !rewardExpiration(-1, now).IsZero() {
		t.Fatalf("period=-1 must yield zero time")
	}
	if !rewardExpiration(0, now).IsZero() {
		t.Fatalf("period=0 must yield zero time")
	}
}

func TestSubstituteWorldMsg(t *testing.T) {
	got := substituteWorldMsg("/name has obtained /item from a box! /name is lucky.", "Hero", "Golden Apple")
	want := "Hero has obtained Golden Apple from a box! Hero is lucky."
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
