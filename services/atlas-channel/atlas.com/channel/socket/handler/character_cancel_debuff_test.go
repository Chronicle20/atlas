package handler

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"

	charsb "github.com/Chronicle20/atlas/libs/atlas-packet/character/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// TestCancelDebuffDecode pins the wire format CancelDebuffHandleFunc consumes:
// nothing. CWvsContext::CheckTemporaryStatDuration constructs COutPacket(opcode)
// and sends it with no encode calls on all ten client versions
// (investigation.md §8.1), so the decode must consume zero bytes and must not
// fail on an empty reader.
func TestCancelDebuffDecode(t *testing.T) {
	req := request.Request([]byte{})
	reader := request.NewRequestReader(&req, 0)

	p := charsb.CancelDebuff{}
	p.Decode(logrus.New(), context.Background())(&reader, map[string]interface{}{})

	if p.Operation() != charsb.CancelDebuffHandle {
		t.Errorf("operation: got %q, want %q", p.Operation(), charsb.CancelDebuffHandle)
	}
	if p.String() != "" {
		t.Errorf("String(): got %q, want %q", p.String(), "")
	}
}

// TestCancelDebuffHandleFuncSymbol verifies the handler constructor returns a
// non-nil closure with the standard handler signature.
func TestCancelDebuffHandleFuncSymbol(t *testing.T) {
	got := CancelDebuffHandleFunc(logrus.New(), context.Background(), nil)
	if got == nil {
		t.Fatal("CancelDebuffHandleFunc returned nil closure")
	}
}
