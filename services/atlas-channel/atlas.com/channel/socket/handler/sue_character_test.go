package handler

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"

	fieldsb "github.com/Chronicle20/atlas/libs/atlas-packet/field/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestSueCharacterDecodeLegacy pins the v83 wire format SueCharacterHandleFunc
// consumes: the leading field is the accused character id (uint32 LE), not
// the v95 sub-command string.
func TestSueCharacterDecodeLegacy(t *testing.T) {
	ten := mustTenant(t, "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), ten)

	raw := []byte{
		0x39, 0x30, 0x00, 0x00, // characterId = 12345
		0x05,                 // flag = 5
		0x02, 0x00, 'h', 'i', // reason = "hi"
	}
	req := request.Request(raw)
	reader := request.NewRequestReader(&req, 0)

	p := fieldsb.SueCharacter{}
	p.Decode(logrus.New(), ctx)(&reader, map[string]interface{}{})

	if p.CharacterId() != 12345 {
		t.Errorf("characterId: got %d, want 12345", p.CharacterId())
	}
	if p.SubCommand() != "" {
		t.Errorf("subCommand: got %q, want empty", p.SubCommand())
	}
	if p.Flag() != 5 {
		t.Errorf("flag: got %d, want 5", p.Flag())
	}
	if p.Reason() != "hi" {
		t.Errorf("reason: got %q, want %q", p.Reason(), "hi")
	}
}

// TestSueCharacterDecodeV95 pins the v95 wire format SueCharacterHandleFunc
// consumes: the leading field is a sub-command string, not the legacy
// accused character id.
func TestSueCharacterDecodeV95(t *testing.T) {
	ten := mustTenant(t, "GMS", 95, 1)
	ctx := tenant.WithContext(context.Background(), ten)

	raw := []byte{
		0x05, 0x00, 'a', 'l', 'i', 'c', 'e', // subCommand = "alice"
		0x05,                 // flag = 5
		0x02, 0x00, 'h', 'i', // reason = "hi"
	}
	req := request.Request(raw)
	reader := request.NewRequestReader(&req, 0)

	p := fieldsb.SueCharacter{}
	p.Decode(logrus.New(), ctx)(&reader, map[string]interface{}{})

	if p.CharacterId() != 0 {
		t.Errorf("characterId: got %d, want 0", p.CharacterId())
	}
	if p.SubCommand() != "alice" {
		t.Errorf("subCommand: got %q, want %q", p.SubCommand(), "alice")
	}
	if p.Flag() != 5 {
		t.Errorf("flag: got %d, want 5", p.Flag())
	}
	if p.Reason() != "hi" {
		t.Errorf("reason: got %q, want %q", p.Reason(), "hi")
	}
}

// TestSueCharacterHandleFuncSymbol verifies the handler constructor returns a
// non-nil closure with the standard handler signature.
func TestSueCharacterHandleFuncSymbol(t *testing.T) {
	got := SueCharacterHandleFunc(logrus.New(), context.Background(), nil)
	if got == nil {
		t.Fatal("SueCharacterHandleFunc returned nil closure")
	}
}
