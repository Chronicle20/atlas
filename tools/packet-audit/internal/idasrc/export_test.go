package idasrc

import (
	"context"
	"testing"
)

func TestExportSourceResolve(t *testing.T) {
	src, err := NewExportSource("testdata/gms_v95_mini.json")
	if err != nil {
		t.Fatal(err)
	}
	f, err := src.Resolve(context.Background(), "CLogin::OnCheckPasswordResult")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(f.Calls) != 8 {
		t.Errorf("calls: got %d, want 8", len(f.Calls))
	}
	if f.Calls[7].Op != Decode2 {
		t.Errorf("calls[7]: got %v, want Decode2", f.Calls[7].Op)
	}
	if f.Direction != DirClientbound {
		t.Errorf("direction: got %v", f.Direction)
	}
}
