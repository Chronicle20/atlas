package script

import (
	"atlas-portal-actions/dedupe"
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// fakeProcessor is a script.Processor whose Process return value the test
// controls, and which records that it was called at all.
type fakeProcessor struct {
	Processor
	calls  int
	result ProcessResult
}

func (f *fakeProcessor) Process(_ field.Model, _ uint32, _ string, _ uint32) ProcessResult {
	f.calls++
	return f.result
}

// consumerTestHarness swaps the package seams and restores them on cleanup,
// returning a pointer to the unlock counter and the fake processor.
func consumerTestHarness(t *testing.T, result ProcessResult) (*int, *fakeProcessor, logrus.FieldLogger, context.Context) {
	t.Helper()
	unlocks := 0
	fp := &fakeProcessor{result: result}

	prevProc := newScriptProcessorFn
	prevUnlock := enableActionsFn
	newScriptProcessorFn = func(_ logrus.FieldLogger, _ context.Context, _ *gorm.DB) Processor { return fp }
	enableActionsFn = func(_ logrus.FieldLogger) func(context.Context) func(channel.Model, uint32) {
		return func(_ context.Context) func(channel.Model, uint32) {
			return func(_ channel.Model, _ uint32) { unlocks++ }
		}
	}
	t.Cleanup(func() {
		newScriptProcessorFn = prevProc
		enableActionsFn = prevUnlock
	})

	logger, _ := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return &unlocks, fp, logger, tenant.WithContext(context.Background(), tm)
}

func testEnterCommand() commandEvent[enterBody] {
	return commandEvent[enterBody]{
		WorldId:   0,
		ChannelId: 1,
		MapId:     200090510,
		Instance:  uuid.Nil,
		PortalId:  3,
		Type:      "ENTER",
		Body:      enterBody{CharacterId: 100, PortalName: "undodraco"},
	}
}

// FR-4.1: an outcome that moved the character must NOT unlock the client.
// The SET_FIELD produced by the warp clears m_bExclRequestSent; unlocking here
// while the warp is in flight is what makes the client re-fire.
func TestHandleEnterCommand_MovingOutcomeDoesNotUnlock(t *testing.T) {
	unlocks, _, l, ctx := consumerTestHarness(t, ProcessResult{
		Allow: true, MatchedRule: "r1", CharacterMoved: true,
	})
	handleEnterCommand(l, ctx, nil, testEnterCommand())
	assert.Equal(t, 0, *unlocks, "a moving outcome must not emit EnableActions")
}

// FR-4.1: an outcome that did not move the character still unlocks.
func TestHandleEnterCommand_StaticOutcomeUnlocks(t *testing.T) {
	unlocks, _, l, ctx := consumerTestHarness(t, ProcessResult{
		Allow: true, MatchedRule: "r1", CharacterMoved: false,
	})
	handleEnterCommand(l, ctx, nil, testEnterCommand())
	assert.Equal(t, 1, *unlocks)
}

// FR-2.4: every non-moving outcome keeps unlocking exactly as before.
func TestHandleEnterCommand_NonMovingOutcomesAllUnlock(t *testing.T) {
	for name, result := range map[string]ProcessResult{
		"denied":         {Allow: false, MatchedRule: "r1"},
		"no_script":      {Allow: true, MatchedRule: "no_script"},
		"no_match":       {Allow: false, MatchedRule: "no_match"},
		"allowed_no_ops": {Allow: true, MatchedRule: "r1"},
	} {
		t.Run(name, func(t *testing.T) {
			unlocks, _, l, ctx := consumerTestHarness(t, result)
			handleEnterCommand(l, ctx, nil, testEnterCommand())
			assert.Equal(t, 1, *unlocks)
		})
	}
}

// FR-2.3 strengthened: a rule error where nothing was dispatched still unlocks,
// so the player is never frozen with no saga to release them.
func TestHandleEnterCommand_ErrorWithoutMoveUnlocks(t *testing.T) {
	unlocks, _, l, ctx := consumerTestHarness(t, ProcessResult{
		Allow: false, MatchedRule: "r1", CharacterMoved: false,
		Error: errors.New("rule evaluation failed"),
	})
	handleEnterCommand(l, ctx, nil, testEnterCommand())
	assert.Equal(t, 1, *unlocks)
}

// FR-2.3 strengthened: a dispatched warp followed by an unrelated operation
// error must NOT double-unlock — the warp is in flight.
func TestHandleEnterCommand_ErrorAfterMoveDoesNotUnlock(t *testing.T) {
	unlocks, _, l, ctx := consumerTestHarness(t, ProcessResult{
		Allow: true, MatchedRule: "r1", CharacterMoved: true,
		Error: errors.New("operation execution failed"),
	})
	handleEnterCommand(l, ctx, nil, testEnterCommand())
	assert.Equal(t, 0, *unlocks)
}

// fakeGate returns a fixed decision and counts calls.
type fakeGate struct {
	allow bool
	calls int
}

func (g *fakeGate) Allow(_ logrus.FieldLogger, _ context.Context, _ dedupe.Key) bool {
	g.calls++
	return g.allow
}

func withGate(t *testing.T, g dedupe.Gate) {
	t.Helper()
	prev := gateFn
	gateFn = func() dedupe.Gate { return g }
	t.Cleanup(func() { gateFn = prev })
}

// FR-4.2/FR-3.1: a dropped duplicate performs NO rule evaluation, executes no
// operation, and does not unlock the client.
func TestHandleEnterCommand_DuplicateDroppedBeforeProcessing(t *testing.T) {
	unlocks, fp, l, ctx := consumerTestHarness(t, ProcessResult{
		Allow: true, MatchedRule: "r1", CharacterMoved: true,
	})
	g := &fakeGate{allow: false}
	withGate(t, g)

	handleEnterCommand(l, ctx, nil, testEnterCommand())

	assert.Equal(t, 1, g.calls, "the gate is consulted")
	assert.Equal(t, 0, fp.calls, "the script processor must never be invoked for a duplicate")
	assert.Equal(t, 0, *unlocks, "a dropped duplicate emits no EnableActions")
}

// A first ENTER passes the gate and is processed normally.
func TestHandleEnterCommand_AllowedByGateIsProcessed(t *testing.T) {
	unlocks, fp, l, ctx := consumerTestHarness(t, ProcessResult{
		Allow: true, MatchedRule: "r1", CharacterMoved: false,
	})
	g := &fakeGate{allow: true}
	withGate(t, g)

	handleEnterCommand(l, ctx, nil, testEnterCommand())

	assert.Equal(t, 1, g.calls)
	assert.Equal(t, 1, fp.calls)
	assert.Equal(t, 1, *unlocks)
}
