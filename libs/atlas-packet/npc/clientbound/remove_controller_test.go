package clientbound

import (
	"bytes"
	"context"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// Read-order fixture for the remove arm of CNpcPool::OnNpcChangeController:
// Decode1 (flag=0) + Decode4 (dwNpcId) -> SetRemoteNpc. IDA: GMS v95
// 0x679730, GMS v83 0x6d9a83 (byte-identical across the version set).
func TestNpcRemoveControllerEncode(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	m := NewNpcRemoveController(0, 0x01020304)
	out := m.Encode(l, context.Background())(map[string]interface{}{})
	want := []byte{0x00, 0x04, 0x03, 0x02, 0x01}
	if !bytes.Equal(out, want) {
		t.Fatalf("encode mismatch: got % X want % X", out, want)
	}
}

func TestNpcRemoveControllerDecodeRoundTrip(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	m := NewNpcRemoveController(0, 42)
	raw := m.Encode(l, context.Background())(map[string]interface{}{})
	req := request.Request(raw)
	r := request.NewRequestReader(&req, 0)

	var d RemoveController
	d.Decode(l, context.Background())(&r, map[string]interface{}{})
	if d.Id() != 42 {
		t.Fatalf("round-trip id mismatch: got %d", d.Id())
	}
	if r.Available() > 0 {
		t.Fatalf("reader has %d unconsumed bytes after decode", r.Available())
	}
}
