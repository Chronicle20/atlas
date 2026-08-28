package message

import (
	"atlas-messages/character"
	"atlas-messages/chat"
	"atlas-messages/command"
	"context"
	"errors"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestCaptureLineSwallowsRedisOutage proves that captureLine never surfaces a
// Redis failure to its caller: chat capture is best-effort and must not
// break message delivery. It simulates an outage by closing the backing
// miniredis instance out from under an already-initialized registry, then
// calls captureLine directly and asserts (a) it returns nothing the caller
// could observe as failure (its signature has no error return) and (b) the
// failure was logged at Warn so it isn't silently lost either.
func TestCaptureLineSwallowsRedisOutage(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run: %v", err)
	}
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	chat.InitRegistry(client)
	mr.Close() // simulate a Redis outage for all subsequent commands

	l, hook := test.NewNullLogger()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	ctx := tenant.WithContext(context.Background(), tm)
	p := &ProcessorImpl{l: l, ctx: ctx}
	f := field.NewBuilder(0, 1, 100000000).Build()

	// If capture failure were not swallowed, this would panic or the test
	// harness would otherwise surface it; completing normally is the point.
	p.captureLine(f, 1, "Alice", "GENERAL", "hello")

	foundWarn := false
	for _, e := range hook.AllEntries() {
		if e.Level == logrus.WarnLevel {
			foundWarn = true
		}
	}
	if !foundWarn {
		t.Errorf("expected a Warn-level log entry on redis outage, got: %+v", hook.AllEntries())
	}
}

// stubCharacterProcessor is a minimal character.Processor substitute for
// tests that exercise message.ProcessorImpl's Handle* methods without a live
// REST dependency. It is not a test-only constructor or a *_testhelpers.go
// file — it's an ordinary type declared in this _test.go file, injected via
// the production NewProcessorWithClients seam.
type stubCharacterProcessor struct {
	byId map[uint32]character.Model
}

var _ character.Processor = (*stubCharacterProcessor)(nil)

func (s *stubCharacterProcessor) GetById(_ ...model.Decorator[character.Model]) func(characterId uint32) (character.Model, error) {
	return func(characterId uint32) (character.Model, error) {
		if m, ok := s.byId[characterId]; ok {
			return m, nil
		}
		return character.Model{}, errors.New("character not found")
	}
}

func (s *stubCharacterProcessor) ByNameProvider(_ ...model.Decorator[character.Model]) func(name string) model.Provider[[]character.Model] {
	return func(name string) model.Provider[[]character.Model] {
		return func() ([]character.Model, error) { return nil, nil }
	}
}

func (s *stubCharacterProcessor) GetByName(_ ...model.Decorator[character.Model]) func(name string) (character.Model, error) {
	return func(name string) (character.Model, error) {
		return character.Model{}, errors.New("character not found")
	}
}

func (s *stubCharacterProcessor) IdByNameProvider(_ string) model.Provider[uint32] {
	return func() (uint32, error) { return 0, errors.New("not implemented") }
}

func (s *stubCharacterProcessor) SkillModelDecorator(m character.Model) character.Model {
	return m
}

// setupChatBuffer wires a fresh miniredis-backed chat registry for the
// duration of the test.
func setupChatBuffer(t *testing.T) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run: %v", err)
	}
	t.Cleanup(mr.Close)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	chat.InitRegistry(client)
}

func testTenantContext(t *testing.T) context.Context {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tenant.WithContext(context.Background(), tm)
}

// TestHandleGeneralCapturesLine is the positive control: a captured chat
// type (GENERAL) must actually land in the buffer when relayed through the
// real Handle path. Without this, the negative-capture tests below could
// pass trivially if capture were broken outright.
func TestHandleGeneralCapturesLine(t *testing.T) {
	setupChatBuffer(t)

	l, _ := test.NewNullLogger()
	ctx := testTenantContext(t)
	alice, err := character.NewBuilder().SetId(1).SetName("Alice").SetWorldId(0).Build()
	if err != nil {
		t.Fatalf("failed to build test character: %v", err)
	}
	cp := &stubCharacterProcessor{byId: map[uint32]character.Model{1: alice}}
	p := NewProcessorWithClients(l, ctx, cp)
	f := field.NewBuilder(0, 1, 100000000).Build()

	if err := p.HandleGeneral(f, 1, "hello everyone", false); err != nil {
		t.Fatalf("HandleGeneral: %v", err)
	}

	lines, err := chat.NewProcessor(l, ctx).RecentInvolving([]uint32{1})
	if err != nil {
		t.Fatalf("RecentInvolving: %v", err)
	}
	if len(lines) != 1 {
		t.Fatalf("expected 1 captured line, got %d: %+v", len(lines), lines)
	}
	if lines[0].Text != "hello everyone" || lines[0].ChatType != "GENERAL" || lines[0].SenderName != "Alice" {
		t.Errorf("captured line mismatch: %+v", lines[0])
	}
}

// TestHandlePetDoesNotCaptureLine proves by execution that pet chat (which
// echoes owner input under the pet's actor id) is never written to the chat
// capture buffer.
func TestHandlePetDoesNotCaptureLine(t *testing.T) {
	setupChatBuffer(t)

	l, _ := test.NewNullLogger()
	ctx := testTenantContext(t)
	cp := &stubCharacterProcessor{byId: map[uint32]character.Model{}}
	p := NewProcessorWithClients(l, ctx, cp)
	f := field.NewBuilder(0, 1, 100000000).Build()

	const petId = uint32(42)
	if err := p.HandlePet(f, petId, "pet says hi", 1, 0, 0, 0, false); err != nil {
		t.Fatalf("HandlePet: %v", err)
	}

	lines, err := chat.NewProcessor(l, ctx).RecentInvolving([]uint32{petId, 1})
	if err != nil {
		t.Fatalf("RecentInvolving: %v", err)
	}
	if len(lines) != 0 {
		t.Fatalf("expected pet chat to NOT be captured, got %d lines: %+v", len(lines), lines)
	}
}

// TestIssuePinkTextDoesNotCaptureLine proves by execution that system-issued
// pink text is never written to the chat capture buffer.
func TestIssuePinkTextDoesNotCaptureLine(t *testing.T) {
	setupChatBuffer(t)

	l, _ := test.NewNullLogger()
	ctx := testTenantContext(t)
	cp := &stubCharacterProcessor{byId: map[uint32]character.Model{}}
	p := NewProcessorWithClients(l, ctx, cp)
	f := field.NewBuilder(0, 1, 100000000).Build()

	const actorId = uint32(7)
	if err := p.IssuePinkText(f, actorId, "system notice", []uint32{1, 2}); err != nil {
		t.Fatalf("IssuePinkText: %v", err)
	}

	lines, err := chat.NewProcessor(l, ctx).RecentInvolving([]uint32{actorId, 1, 2})
	if err != nil {
		t.Fatalf("RecentInvolving: %v", err)
	}
	if len(lines) != 0 {
		t.Fatalf("expected pink text to NOT be captured, got %d lines: %+v", len(lines), lines)
	}
}

const slashCommandTestMessage = "@atlas-messages-capture-test-slash"

// slashCommandTestProducer is a command.Producer that only matches
// slashCommandTestMessage, registered once for
// TestSlashCommandShortCircuitsBeforeCapture. command.Registry() is a
// process-wide singleton, so the trigger text is namespaced to this test to
// avoid colliding with any other command or test in this binary.
func slashCommandTestProducer(_ logrus.FieldLogger) func(ctx context.Context) func(f field.Model, c character.Model, m string) (command.Executor, bool) {
	return func(_ context.Context) func(f field.Model, c character.Model, m string) (command.Executor, bool) {
		return func(_ field.Model, _ character.Model, m string) (command.Executor, bool) {
			if m != slashCommandTestMessage {
				return nil, false
			}
			return func(_ logrus.FieldLogger) func(ctx context.Context) error {
				return func(_ context.Context) error { return nil }
			}, true
		}
	}
}

// TestSlashCommandShortCircuitsBeforeCapture proves by execution that a
// registered slash command returns before captureLine is ever reached — the
// command-registry check in HandleGeneral runs first, and command execution
// short-circuits the function before it gets anywhere near the emit/capture
// lines.
func TestSlashCommandShortCircuitsBeforeCapture(t *testing.T) {
	command.Registry().Add(slashCommandTestProducer)

	setupChatBuffer(t)

	l, _ := test.NewNullLogger()
	ctx := testTenantContext(t)
	alice, err := character.NewBuilder().SetId(1).SetName("Alice").SetWorldId(0).Build()
	if err != nil {
		t.Fatalf("failed to build test character: %v", err)
	}
	cp := &stubCharacterProcessor{byId: map[uint32]character.Model{1: alice}}
	p := NewProcessorWithClients(l, ctx, cp)
	f := field.NewBuilder(0, 1, 100000000).Build()

	if err := p.HandleGeneral(f, 1, slashCommandTestMessage, false); err != nil {
		t.Fatalf("HandleGeneral: %v", err)
	}

	lines, err := chat.NewProcessor(l, ctx).RecentInvolving([]uint32{1})
	if err != nil {
		t.Fatalf("RecentInvolving: %v", err)
	}
	if len(lines) != 0 {
		t.Fatalf("expected a recognized slash command to short-circuit before capture, got %d lines: %+v", len(lines), lines)
	}
}
