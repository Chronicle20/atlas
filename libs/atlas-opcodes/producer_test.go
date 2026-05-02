package opcodes

import (
	"strings"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

type stubOpWriter struct{}

func (stubOpWriter) Write(_ uint16) func(w *response.Writer) {
	return func(_ *response.Writer) {}
}

func nilAdapter(_ string, _ interface{}, _ interface{}, _ map[string]interface{}) request.Handler {
	return nil
}

func warnContaining(h *test.Hook, subs ...string) bool {
	for _, e := range h.AllEntries() {
		if e.Level != logrus.WarnLevel {
			continue
		}
		matched := true
		for _, sub := range subs {
			if !strings.Contains(e.Message, sub) {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}

func warnCount(h *test.Hook) int {
	n := 0
	for _, e := range h.AllEntries() {
		if e.Level == logrus.WarnLevel {
			n++
		}
	}
	return n
}

func TestBuildHandlerMap_WarnsOnUnknownHandler(t *testing.T) {
	l, h := test.NewNullLogger()

	handlers := []HandlerConfig{
		{OpCode: "0x45", Validator: "NoOp", Handler: "MissingHandle"},
	}
	validatorMap := map[string]interface{}{"NoOp": struct{}{}}
	handlerMap := map[string]interface{}{}

	BuildHandlerMap(l, handlers, validatorMap, handlerMap, nilAdapter)

	if !warnContaining(h, "MissingHandle", "0x45") {
		t.Fatalf("expected warning naming missing handler and opcode; entries: %d", len(h.AllEntries()))
	}
}

func TestBuildHandlerMap_NoWarnWhenHandlerKnown(t *testing.T) {
	l, h := test.NewNullLogger()

	handlers := []HandlerConfig{
		{OpCode: "0x01", Validator: "NoOp", Handler: "LoginHandle"},
	}
	validatorMap := map[string]interface{}{"NoOp": struct{}{}}
	handlerMap := map[string]interface{}{"LoginHandle": struct{}{}}

	BuildHandlerMap(l, handlers, validatorMap, handlerMap, nilAdapter)

	if got := warnCount(h); got != 0 {
		t.Fatalf("expected zero warnings, got %d", got)
	}
}

func TestBuildWriterProducer_WarnsOnUnconfiguredAvailableWriter(t *testing.T) {
	l, h := test.NewNullLogger()

	writers := []WriterConfig{
		{OpCode: "0x10", Writer: "ConfiguredWriter"},
	}
	available := []string{"ConfiguredWriter", "OrphanWriter"}

	BuildWriterProducer(l, writers, available, stubOpWriter{})

	if !warnContaining(h, "OrphanWriter") {
		t.Fatalf("expected warning naming OrphanWriter")
	}
	if warnContaining(h, "ConfiguredWriter") {
		t.Fatalf("did not expect warning for configured writer")
	}
}

func TestBuildWriterProducer_NoWarnWhenAllAvailableConfigured(t *testing.T) {
	l, h := test.NewNullLogger()

	writers := []WriterConfig{
		{OpCode: "0x10", Writer: "WriterA"},
		{OpCode: "0x11", Writer: "WriterB"},
	}
	available := []string{"WriterA", "WriterB"}

	BuildWriterProducer(l, writers, available, stubOpWriter{})

	if got := warnCount(h); got != 0 {
		t.Fatalf("expected zero warnings, got %d", got)
	}
}
