package atlaspacket

import "testing"

func TestSimpleEncodeExtractsThreeCalls(t *testing.T) {
	calls, err := AnalyzeFile("testdata/simple_encode.go.txt", "Simple", "Encode")
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 3 {
		t.Fatalf("calls: got %d, want 3 (%+v)", len(calls), calls)
	}
	if calls[0].Op != Encode1 || calls[1].Op != Encode4 || calls[2].Op != EncodeStr {
		t.Errorf("ops: got %v %v %v", calls[0].Op, calls[1].Op, calls[2].Op)
	}
}
