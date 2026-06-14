package door

import "testing"

func TestResolveTownMapForcedWins(t *testing.T) {
	if got := ResolveTownMap(104000000, 100000000); got != 100000000 {
		t.Fatalf("forced should win, got %d", got)
	}
}
func TestResolveTownMapReturnWhenNoForced(t *testing.T) {
	if got := ResolveTownMap(104000000, noMap); got != 104000000 {
		t.Fatalf("want return map, got %d", got)
	}
}
func TestHasReturnMapFalseWhenNone(t *testing.T) {
	if HasValidReturn(noMap, noMap) {
		t.Fatalf("expected no valid return")
	}
}
