package wzsnapshot

import "testing"

func TestLoadSnapshot_HashPinned(t *testing.T) {
	skills, jobs, hash, err := LoadSnapshot("gms", 48, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) == 0 || len(jobs) == 0 {
		t.Fatal("empty snapshot")
	}
	if hash != HashIds(skills, jobs) {
		t.Fatalf("snapshot hash drift: file=%s computed=%s", hash, HashIds(skills, jobs))
	}
	// v0.48 has no 900/910 GM jobs at the *900* range only if audit says so; assert Pirate-range 5101004 skill exists
	if !contains32(skills, 5101004) {
		t.Fatal("expected wire 5101004 present in v48 snapshot")
	}
}
