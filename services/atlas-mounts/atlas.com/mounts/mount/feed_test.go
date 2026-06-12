package mount

import "testing"

func TestExpNeededForLevelTableMatchesPinnedValues(t *testing.T) {
	cases := map[int]int32{0: 1, 1: 24, 2: 50, 10: 430, 18: 1701, 28: 4550} // from §8.2
	for lvl, want := range cases {
		if got := ExpNeededForLevel(lvl); got != want {
			t.Fatalf("ExpNeededForLevel(%d)=%d want %d", lvl, got, want)
		}
	}
}

func TestExpNeededForLevelBeyondTableDoesNotPanic(t *testing.T) {
	// level 29, 30, 35 are past the 29-entry table — must return sentinel, not panic
	for _, lvl := range []int{29, 30, 35} {
		if got := ExpNeededForLevel(lvl); got < 4550 {
			t.Fatalf("ExpNeededForLevel(%d)=%d expected large sentinel (no further level-up)", lvl, got)
		}
	}
}

func TestApplyFeedHealsAndGainsExp(t *testing.T) {
	// healMax 30, tiredness 20, level 1 → heal=20, tiredness=0,
	// gain=ceil((20/30)*(2*1+6))=ceil(5.333)=6, no level-up (need 24)
	res := ApplyFeed(FeedInput{Level: 1, Exp: 0, Tiredness: 20, HealMax: 30})
	if res.Tiredness != 0 || res.Exp != 6 || res.LevelUp {
		t.Fatalf("got %+v", res)
	}
}

func TestApplyFeedLevelsUp(t *testing.T) {
	// level 1, exp 22, tiredness 30, healMax 30 → gain=ceil((30/30)*(2+6))=8
	// → exp 30 >= need(1)=24 → level 2, exp 6, LevelUp
	res := ApplyFeed(FeedInput{Level: 1, Exp: 22, Tiredness: 30, HealMax: 30})
	if res.Level != 2 || res.Exp != 6 || !res.LevelUp {
		t.Fatalf("got %+v", res)
	}
}

func TestApplyFeedAtCapDoesNotLevel(t *testing.T) {
	res := ApplyFeed(FeedInput{Level: CAP, Exp: 0, Tiredness: 99, HealMax: 30})
	if res.Level != CAP || res.LevelUp {
		t.Fatalf("cap exceeded: %+v", res)
	}
}

func TestApplyFeedNearTableEndDoesNotPanic(t *testing.T) {
	// level 29 with huge exp must not panic and must not level past where the table allows.
	// need(29)=sentinel, so exp can never reach it — level stays bounded.
	res := ApplyFeed(FeedInput{Level: 29, Exp: 999999, Tiredness: 99, HealMax: 30})
	if res.Level < 29 || res.Level > CAP {
		t.Fatalf("level out of bounds: %+v", res)
	}
}

func TestApplyFeedNoTirednessNoGain(t *testing.T) {
	// tiredness 0 → heal 0 → gain ceil(0)=0, no change
	res := ApplyFeed(FeedInput{Level: 5, Exp: 100, Tiredness: 0, HealMax: 30})
	if res.Tiredness != 0 || res.Exp != 100 || res.Level != 5 || res.LevelUp {
		t.Fatalf("got %+v", res)
	}
}

func TestApplyFeedHealCappedByHealMax(t *testing.T) {
	// tiredness 50 > healMax 30 → heal=30, tiredness=20 remaining
	res := ApplyFeed(FeedInput{Level: 1, Exp: 0, Tiredness: 50, HealMax: 30})
	if res.Tiredness != 20 {
		t.Fatalf("expected tiredness 20, got %+v", res)
	}
}
