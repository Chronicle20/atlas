package atlaspacket

import "testing"

func TestRecurseMarker(t *testing.T) {
	calls, err := AnalyzeFile("testdata/recurse_encode.go.txt", "Recursive", "Encode")
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 3 {
		t.Fatalf("calls=%d, want 3 (1 byte, 1 recurse, 1 int): %+v", len(calls), calls)
	}
	if calls[1].Kind != KindRecurse {
		t.Errorf("calls[1].Kind=%v, want KindRecurse", calls[1].Kind)
	}
	if calls[1].RecurseType == "" {
		t.Errorf("calls[1].RecurseType should not be empty")
	}
}

func TestLoopRepeat(t *testing.T) {
	calls, err := AnalyzeFile("testdata/loop_encode.go.txt", "Looped", "Encode")
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 {
		t.Fatalf("top-level calls=%d, want 2", len(calls))
	}
	if calls[1].Kind != KindRepeat {
		t.Fatalf("calls[1].Kind=%v, want KindRepeat", calls[1].Kind)
	}
	if len(calls[1].Body) != 2 {
		t.Fatalf("repeat body=%d, want 2", len(calls[1].Body))
	}
}
