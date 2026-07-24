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

// An untagged (legacy) entry still applies to every service, so an unregistered
// handler for it warns exactly as before — nothing regresses for old configs.
func TestBuildHandlerMap_WarnsOnUnknownHandler(t *testing.T) {
	l, h := test.NewNullLogger()

	handlers := []HandlerConfig{
		{OpCode: "0x45", Validator: "NoOp", Handler: "MissingHandle"},
	}
	validatorMap := map[string]interface{}{"NoOp": struct{}{}}
	handlerMap := map[string]interface{}{}

	BuildHandlerMap(l, ServiceLogin, handlers, validatorMap, handlerMap, nilAdapter)

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

	BuildHandlerMap(l, ServiceLogin, handlers, validatorMap, handlerMap, nilAdapter)

	if got := warnCount(h); got != 0 {
		t.Fatalf("expected zero warnings, got %d", got)
	}
}

// An entry owned by another service is skipped entirely: not mapped, and crucially
// not warned about — this is the cross-service noise the scoping removes.
func TestBuildHandlerMap_SkipsOtherServiceEntry(t *testing.T) {
	l, h := test.NewNullLogger()

	handlers := []HandlerConfig{
		{OpCode: "0x01", Validator: "NoOp", Handler: "LoginHandle", Services: []string{ServiceLogin}},
		{OpCode: "0x45", Validator: "NoOp", Handler: "ChannelItemUseHandle", Services: []string{ServiceChannel}},
	}
	validatorMap := map[string]interface{}{"NoOp": struct{}{}}
	handlerMap := map[string]interface{}{"LoginHandle": struct{}{}} // login only registers login handlers

	result := BuildHandlerMap(l, ServiceLogin, handlers, validatorMap, handlerMap, nilAdapter)

	if _, ok := result[0x01]; !ok {
		t.Fatalf("expected own login handler (0x01) to be mapped")
	}
	if _, ok := result[0x45]; ok {
		t.Fatalf("did not expect the channel-scoped handler (0x45) to be mapped")
	}
	if got := warnCount(h); got != 0 {
		t.Fatalf("expected zero warnings (channel entry skipped, not warned), got %d", got)
	}
}

// A shared entry (e.g. Pong) is tagged for both services and processed by each.
func TestBuildHandlerMap_ProcessesSharedEntry(t *testing.T) {
	l, _ := test.NewNullLogger()

	handlers := []HandlerConfig{
		{OpCode: "0x18", Validator: "NoOp", Handler: "PongHandle", Services: []string{ServiceLogin, ServiceChannel}},
	}
	validatorMap := map[string]interface{}{"NoOp": struct{}{}}
	handlerMap := map[string]interface{}{"PongHandle": struct{}{}}

	for _, svc := range []string{ServiceLogin, ServiceChannel} {
		result := BuildHandlerMap(l, svc, handlers, validatorMap, handlerMap, nilAdapter)
		if _, ok := result[0x18]; !ok {
			t.Fatalf("expected shared PongHandle (0x18) mapped for service %q", svc)
		}
	}
}

// A tagged-for-this-service entry whose handler isn't registered is now a REAL
// signal (a genuine per-service gap), not benign cross-service noise.
func TestBuildHandlerMap_WarnsWhenOwnServiceHandlerMissing(t *testing.T) {
	l, h := test.NewNullLogger()

	handlers := []HandlerConfig{
		{OpCode: "0x01", Validator: "NoOp", Handler: "LoginHandle", Services: []string{ServiceLogin}},
	}
	validatorMap := map[string]interface{}{"NoOp": struct{}{}}
	handlerMap := map[string]interface{}{} // login-owned handler NOT registered

	BuildHandlerMap(l, ServiceLogin, handlers, validatorMap, handlerMap, nilAdapter)

	if !warnContaining(h, "LoginHandle", "0x01") {
		t.Fatalf("expected a real warning for a login-owned handler that is not registered")
	}
}

func TestBuildWriterProducer_SkipsOtherServiceWriter(t *testing.T) {
	l, h := test.NewNullLogger()

	writers := []WriterConfig{
		{OpCode: "0x1A", Writer: "SelectWorld", Services: []string{ServiceLogin}},
		{OpCode: "0x5C", Writer: "NpcTalk", Services: []string{ServiceChannel}},
	}
	// login only declares (and maps) SelectWorld; NpcTalk is a channel writer.
	available := []string{"SelectWorld"}

	BuildWriterProducer(l, ServiceLogin, writers, available, stubOpWriter{})

	// The channel-scoped writer is skipped; SelectWorld is configured, so no warns.
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

	BuildWriterProducer(l, ServiceLogin, writers, available, stubOpWriter{})

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

	BuildWriterProducer(l, ServiceLogin, writers, available, stubOpWriter{})

	if got := warnCount(h); got != 0 {
		t.Fatalf("expected zero warnings, got %d", got)
	}
}
