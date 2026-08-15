package registry

import (
	"context"
	"encoding/json"
	"testing"
)

type stubHandler struct{ t string }

func (h stubHandler) Type() string                                { return h.t }
func (h stubHandler) ValidateConfiguration(json.RawMessage) error { return nil }
func (h stubHandler) ConcurrencyKey(context.Context, json.RawMessage) (string, error) {
	return "", nil
}
func (h stubHandler) ConcurrencyKeyIsConstant() bool                            { return false }
func (h stubHandler) Evaluate(context.Context, Definition, Work) (*Seed, error) { return nil, nil }
func (h stubHandler) Start(context.Context, Occurrence) (Progress, error)       { return Progress{}, nil }
func (h stubHandler) Advance(context.Context, Occurrence, Work) (Progress, error) {
	return Progress{}, nil
}

func TestRegisterAndGet(t *testing.T) {
	reset()
	Register(stubHandler{t: "TEST_EVENT"})

	h, ok := Get("TEST_EVENT")
	if !ok {
		t.Fatalf("registered handler not found")
	}
	if h.Type() != "TEST_EVENT" {
		t.Fatalf("Type = %q", h.Type())
	}
}

// An unregistered type is a FAILURE, not a fallback: a definition row whose
// type has no handler must make its work rows fail loudly rather than silently
// succeed with no occurrence (design §3.2).
func TestGetUnknownTypeReportsMissing(t *testing.T) {
	reset()
	if _, ok := Get("NOPE"); ok {
		t.Fatalf("Get returned ok for an unregistered type")
	}
}

func TestRegisterDuplicatePanics(t *testing.T) {
	reset()
	Register(stubHandler{t: "DUP"})
	defer func() {
		if recover() == nil {
			t.Fatalf("expected a panic on duplicate registration")
		}
	}()
	Register(stubHandler{t: "DUP"})
}
