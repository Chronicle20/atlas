package handler

import (
	"atlas-channel/session"
	"atlas-channel/test"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// rpsOperations mirrors the operations table Task 20's seed template will
// define for RPSActionHandle. The names MUST match RPSActionMode* consts:
// START, SELECT, UPDATE, CONTINUE, EXIT, RETRY.
var rpsOperations = map[string]interface{}{
	"START":    float64(0),
	"SELECT":   float64(1),
	"UPDATE":   float64(2),
	"CONTINUE": float64(3),
	"EXIT":     float64(4),
	"RETRY":    float64(5),
}

func rpsTestSession() session.Model {
	return session.NewSession(uuid.New(), test.CreateDefaultMockTenant(), 0, nil)
}

func rpsTestOptions() map[string]interface{} {
	return map[string]interface{}{"operations": rpsOperations}
}

// rpsEmitCall records a single emit-seam invocation.
type rpsEmitCall struct {
	kind  string
	throw byte
}

// installRPSEmitSeams swaps the four emit funcs to capture invocations
// instead of hitting a real Kafka producer, and returns the recorded calls
// plus a restore func. Mirrors the door handler's doorsByOwnerFunc/
// partyMemberIdsFunc seam pattern (mystic_door_enter.go).
func installRPSEmitSeams(t *testing.T) (*[]rpsEmitCall, func()) {
	t.Helper()
	calls := &[]rpsEmitCall{}

	origBegin := emitRPSBeginFunc
	origSelect := emitRPSSelectFunc
	origContinue := emitRPSContinueFunc
	origRetry := emitRPSRetryFunc
	origCollect := emitRPSCollectFunc

	emitRPSBeginFunc = func(_ logrus.FieldLogger, _ context.Context, _ uint32, _ world.Id, _ channel.Id) error {
		*calls = append(*calls, rpsEmitCall{kind: "BEGIN"})
		return nil
	}
	emitRPSRetryFunc = func(_ logrus.FieldLogger, _ context.Context, _ uint32, _ world.Id, _ channel.Id) error {
		*calls = append(*calls, rpsEmitCall{kind: "RETRY"})
		return nil
	}
	emitRPSSelectFunc = func(_ logrus.FieldLogger, _ context.Context, _ uint32, _ world.Id, _ channel.Id, throw byte) error {
		*calls = append(*calls, rpsEmitCall{kind: "SELECT", throw: throw})
		return nil
	}
	emitRPSContinueFunc = func(_ logrus.FieldLogger, _ context.Context, _ uint32, _ world.Id, _ channel.Id) error {
		*calls = append(*calls, rpsEmitCall{kind: "CONTINUE"})
		return nil
	}
	emitRPSCollectFunc = func(_ logrus.FieldLogger, _ context.Context, _ uint32, _ world.Id, _ channel.Id) error {
		*calls = append(*calls, rpsEmitCall{kind: "COLLECT"})
		return nil
	}

	return calls, func() {
		emitRPSBeginFunc = origBegin
		emitRPSSelectFunc = origSelect
		emitRPSContinueFunc = origContinue
		emitRPSRetryFunc = origRetry
		emitRPSCollectFunc = origCollect
	}
}

func runRPSAction(t *testing.T, raw []byte) *[]rpsEmitCall {
	t.Helper()
	calls, restore := installRPSEmitSeams(t)
	defer restore()

	l, _ := testlog.NewNullLogger()
	s := rpsTestSession()
	req := request.Request(raw)
	reader := request.NewRequestReader(&req, 0)

	fn := RPSActionHandleFunc(l, context.Background(), nil)
	fn(s, &reader, rpsTestOptions())
	return calls
}

func TestRPSActionSelectEmitsSelectCommandWithThrow(t *testing.T) {
	for _, throw := range []byte{0, 1, 2} {
		// mode=1 (SELECT) + raw throw byte, unremapped.
		calls := runRPSAction(t, []byte{1, throw})
		if len(*calls) != 1 {
			t.Fatalf("throw %d: want 1 emitted call, got %d", throw, len(*calls))
		}
		got := (*calls)[0]
		if got.kind != "SELECT" {
			t.Fatalf("throw %d: want kind SELECT, got %s", throw, got.kind)
		}
		if got.throw != throw {
			t.Fatalf("throw %d: want throw byte %d unmodified, got %d", throw, throw, got.throw)
		}
	}
}

func TestRPSActionContinueEmitsContinueCommand(t *testing.T) {
	calls := runRPSAction(t, []byte{3})
	if len(*calls) != 1 || (*calls)[0].kind != "CONTINUE" {
		t.Fatalf("want single CONTINUE call, got %+v", *calls)
	}
}

func TestRPSActionExitEmitsCollectCommand(t *testing.T) {
	calls := runRPSAction(t, []byte{4})
	if len(*calls) != 1 || (*calls)[0].kind != "COLLECT" {
		t.Fatalf("want single COLLECT call (EXIT->Collect per corrected mapping), got %+v", *calls)
	}
}

func TestRPSActionStartEmitsBeginCommand(t *testing.T) {
	calls := runRPSAction(t, []byte{0})
	if len(*calls) != 1 || (*calls)[0].kind != "BEGIN" {
		t.Fatalf("START must emit a single BEGIN command (opens the first round), got %+v", *calls)
	}
}

func TestRPSActionUpdateIsDropped(t *testing.T) {
	calls := runRPSAction(t, []byte{2})
	if len(*calls) != 0 {
		t.Fatalf("UPDATE/timeout must be a no-op, got %+v", *calls)
	}
}

func TestRPSActionRetryEmitsRetryCommand(t *testing.T) {
	calls := runRPSAction(t, []byte{5})
	if len(*calls) != 1 || (*calls)[0].kind != "RETRY" {
		t.Fatalf("RETRY must emit a single RETRY command (restart after a loss), got %+v", *calls)
	}
}

func TestRPSActionUnknownModeLogsWarningAndDrops(t *testing.T) {
	l, hook := testlog.NewNullLogger()
	calls, restore := installRPSEmitSeams(t)
	defer restore()

	s := rpsTestSession()
	raw := []byte{99}
	req := request.Request(raw)
	reader := request.NewRequestReader(&req, 0)

	fn := RPSActionHandleFunc(l, context.Background(), nil)
	fn(s, &reader, rpsTestOptions())

	if len(*calls) != 0 {
		t.Fatalf("unknown mode must not emit any command, got %+v", *calls)
	}
	found := false
	for _, e := range hook.AllEntries() {
		if e.Level == logrus.WarnLevel {
			found = true
		}
	}
	if !found {
		t.Fatal("unknown mode must log a warning")
	}
}

func TestRPSActionHandleConstAliasesServerbound(t *testing.T) {
	if RPSActionHandle != "RPSActionHandle" {
		t.Fatalf("RPSActionHandle const drifted: got %q", RPSActionHandle)
	}
}
