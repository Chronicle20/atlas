package opcodes

import (
	"testing"
)

func TestNewRegistry(t *testing.T) {
	handlers := []HandlerConfig{
		{OpCode: "0x01", Validator: "NoOp", Handler: "LoginHandle", Options: nil},
		{OpCode: "0x04", Validator: "LoggedIn", Handler: "ServerListRequestHandle", Options: nil},
	}
	writers := []WriterConfig{
		{OpCode: "0x00", Writer: "AuthSuccess", Options: map[string]interface{}{"codes": map[string]interface{}{"OK": float64(0)}}},
		{OpCode: "0x03", Writer: "ServerStatus", Options: nil},
	}

	r := NewRegistry(handlers, writers)

	// Writer lookups
	op, ok := r.WriterOpCode("AuthSuccess")
	if !ok || op != 0x00 {
		t.Errorf("WriterOpCode(AuthSuccess) = %d, %v; want 0, true", op, ok)
	}

	name, ok := r.WriterName(0x03)
	if !ok || name != "ServerStatus" {
		t.Errorf("WriterName(0x03) = %s, %v; want ServerStatus, true", name, ok)
	}

	opts := r.WriterOptions("AuthSuccess")
	if opts == nil {
		t.Error("WriterOptions(AuthSuccess) = nil; want non-nil")
	}

	// Handler lookups
	op, ok = r.HandlerOpCode("LoginHandle")
	if !ok || op != 0x01 {
		t.Errorf("HandlerOpCode(LoginHandle) = %d, %v; want 1, true", op, ok)
	}

	name, ok = r.HandlerName(0x04)
	if !ok || name != "ServerListRequestHandle" {
		t.Errorf("HandlerName(0x04) = %s, %v; want ServerListRequestHandle, true", name, ok)
	}

	// Missing lookups
	_, ok = r.WriterOpCode("NonExistent")
	if ok {
		t.Error("WriterOpCode(NonExistent) should return false")
	}

	_, ok = r.HandlerName(0xFF)
	if ok {
		t.Error("HandlerName(0xFF) should return false")
	}
}

func TestRegistryInvalidOpCode(t *testing.T) {
	handlers := []HandlerConfig{
		{OpCode: "invalid", Validator: "NoOp", Handler: "LoginHandle"},
	}
	writers := []WriterConfig{
		{OpCode: "invalid", Writer: "AuthSuccess"},
	}

	r := NewRegistry(handlers, writers)

	_, ok := r.HandlerOpCode("LoginHandle")
	if ok {
		t.Error("HandlerOpCode should return false for invalid opcode")
	}

	_, ok = r.WriterOpCode("AuthSuccess")
	if ok {
		t.Error("WriterOpCode should return false for invalid opcode")
	}
}
