package opcodes

import (
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
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

func TestBuildHandlerMap_IgnoresForeignHandler(t *testing.T) {
	l, h := test.NewNullLogger()

	// The shared tenant socket config lists a handler this service does not
	// implement (e.g. a channel handler in the list read by login). It must be
	// skipped silently — not warned about and not mapped.
	handlers := []HandlerConfig{
		{OpCode: "0x01", Validator: "NoOp", Handler: "LoginHandle"},
		{OpCode: "0x45", Validator: "NoOp", Handler: "ForeignChannelHandle"},
	}
	validatorMap := map[string]interface{}{"NoOp": struct{}{}}
	handlerMap := map[string]interface{}{"LoginHandle": struct{}{}}

	result := BuildHandlerMap(l, handlers, validatorMap, handlerMap, nilAdapter)

	if _, ok := result[0x01]; !ok {
		t.Fatalf("expected own handler LoginHandle (0x01) to be mapped")
	}
	if _, ok := result[0x45]; ok {
		t.Fatalf("did not expect foreign handler (0x45) to be mapped")
	}
	if got := warnCount(h); got != 0 {
		t.Fatalf("expected zero warnings for a foreign handler, got %d", got)
	}
}

func TestBuildHandlerMap_NoWarnOnUnroutedRegisteredHandler(t *testing.T) {
	l, h := test.NewNullLogger()

	// A handler this service registers but that no config opcode routes to is
	// legitimate — older/partial versions route only a subset of features, and
	// utility handlers (DebugHandle, NoOpHandler) are wired ad hoc. Must not warn.
	handlers := []HandlerConfig{
		{OpCode: "0x01", Validator: "NoOp", Handler: "LoginHandle"},
	}
	validatorMap := map[string]interface{}{"NoOp": struct{}{}}
	handlerMap := map[string]interface{}{"LoginHandle": struct{}{}, "DebugHandle": struct{}{}}

	result := BuildHandlerMap(l, handlers, validatorMap, handlerMap, nilAdapter)

	if _, ok := result[0x01]; !ok {
		t.Fatalf("expected LoginHandle (0x01) to be mapped")
	}
	if got := warnCount(h); got != 0 {
		t.Fatalf("expected zero warnings for an unrouted registered handler, got %d", got)
	}
}

func TestBuildHandlerMap_WarnsOnMissingValidatorForOwnHandler(t *testing.T) {
	l, h := test.NewNullLogger()

	// Missing validator for a handler THIS service implements is a genuine config
	// error and still warns — and it must be the ONLY warning (no spurious
	// unrouted/foreign warnings).
	handlers := []HandlerConfig{
		{OpCode: "0x01", Validator: "Missing", Handler: "LoginHandle"},
	}
	validatorMap := map[string]interface{}{}
	handlerMap := map[string]interface{}{"LoginHandle": struct{}{}}

	BuildHandlerMap(l, handlers, validatorMap, handlerMap, nilAdapter)

	if !warnContaining(h, "validator", "LoginHandle") {
		t.Fatalf("expected missing-validator warning naming LoginHandle")
	}
	if got := warnCount(h); got != 1 {
		t.Fatalf("expected exactly one warning (missing validator), got %d", got)
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

func TestBuildHandlerMap_WarnsOnUnparseableOpcode(t *testing.T) {
	l, h := test.NewNullLogger()

	// An own handler whose opcode can't be parsed is a genuine config error: warn
	// once, don't map it, and don't also spuriously report it as unrouted.
	handlers := []HandlerConfig{
		{OpCode: "0xZZ", Validator: "NoOp", Handler: "LoginHandle"},
	}
	validatorMap := map[string]interface{}{"NoOp": struct{}{}}
	handlerMap := map[string]interface{}{"LoginHandle": struct{}{}}

	result := BuildHandlerMap(l, handlers, validatorMap, handlerMap, nilAdapter)

	if len(result) != 0 {
		t.Fatalf("expected no handler mapped for an unparseable opcode, got %d", len(result))
	}
	if !warnContaining(h, "LoginHandle") {
		t.Fatalf("expected a warning naming LoginHandle for the bad opcode")
	}
	if got := warnCount(h); got != 1 {
		t.Fatalf("expected exactly one warning (unparseable opcode), got %d", got)
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
