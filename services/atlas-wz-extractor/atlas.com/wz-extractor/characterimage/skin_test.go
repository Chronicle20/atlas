package characterimage

import "testing"

func TestMapInternalSkin(t *testing.T) {
	cases := map[int]int{0: 2000, 5: 2005, 6: 2009, 10: 2013}
	for in, want := range cases {
		got, err := MapInternalSkin(in)
		if err != nil {
			t.Fatalf("MapInternalSkin(%d) errored: %v", in, err)
		}
		if got != want {
			t.Fatalf("MapInternalSkin(%d) = %d, want %d", in, got, want)
		}
	}
	if _, err := MapInternalSkin(11); err == nil {
		t.Fatalf("expected error for skin 11")
	}
}
