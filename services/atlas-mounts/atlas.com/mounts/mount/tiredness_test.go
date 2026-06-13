package mount

import "testing"

func TestTickTirednessClampsAt99(t *testing.T) {
	n, tooTired := TickTiredness(98)
	if n != 99 || tooTired {
		t.Fatalf("98→%d tooTired=%v (want 99,false)", n, tooTired)
	}
	n, tooTired = TickTiredness(99)
	if n != 99 || !tooTired {
		t.Fatalf("99→%d tooTired=%v (want 99,true)", n, tooTired)
	}
}

func TestTickTirednessMidRange(t *testing.T) {
	n, tooTired := TickTiredness(0)
	if n != 1 || tooTired {
		t.Fatalf("0→%d tooTired=%v (want 1,false)", n, tooTired)
	}
}
