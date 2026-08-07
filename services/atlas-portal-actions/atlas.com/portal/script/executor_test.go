package script

import (
	"atlas-portal-actions/action"
	"context"
	"errors"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
	"github.com/Chronicle20/atlas/libs/atlas-script-core/operation"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestMain initializes the package-global action registry against an
// in-memory miniredis instance. executeWarp and executeWarpToSavedLocation
// call action.GetRegistry().AddWithTTL, which panics on a nil registry;
// production wiring initializes it at startup (main.go), but the test binary
// must do so itself since these tests exercise the real executeWarp path
// rather than a fake.
func TestMain(m *testing.M) {
	mr, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer mr.Close()
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	action.InitRegistry(client)
	m.Run()
}

// fakeSagaProcessor records the sagas an executor tries to create and can be
// told to fail.
type fakeSagaProcessor struct {
	created []sharedsaga.Saga
	err     error
}

func (f *fakeSagaProcessor) Create(s sharedsaga.Saga) error {
	if f.err != nil {
		return f.err
	}
	f.created = append(f.created, s)
	return nil
}

func executorTestCtx(t *testing.T) context.Context {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return tenant.WithContext(context.Background(), tm)
}

func testField() field.Model {
	return field.NewBuilder(0, 1, 200090510).SetInstance(uuid.Nil).Build()
}

func mustOp(t *testing.T, opType string, params map[string]string) operation.Model {
	t.Helper()
	b := operation.NewBuilder().SetType(opType)
	if params != nil {
		b = b.SetParams(params)
	}
	m, err := b.Build()
	require.NoError(t, err)
	return m
}

func newTestExecutor(t *testing.T, sp *fakeSagaProcessor) (*OperationExecutor, context.Context) {
	t.Helper()
	logger, _ := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	ctx := executorTestCtx(t)
	return newOperationExecutorWithSaga(logger, ctx, sp), ctx
}

// A dispatched warp reports movedCharacter == true.
func TestExecuteOperations_WarpReportsMoved(t *testing.T) {
	sp := &fakeSagaProcessor{}
	e, _ := newTestExecutor(t, sp)

	moved, err := e.ExecuteOperations(testField(), 100, 3, []operation.Model{
		mustOp(t, "warp", map[string]string{"mapId": "200090500", "portalId": "1"}),
	})
	require.NoError(t, err)
	assert.True(t, moved)
	require.Len(t, sp.created, 1)
}

// warp_to_saved_location likewise.
func TestExecuteOperations_WarpToSavedLocationReportsMoved(t *testing.T) {
	sp := &fakeSagaProcessor{}
	e, _ := newTestExecutor(t, sp)

	moved, err := e.ExecuteOperations(testField(), 100, 3, []operation.Model{
		mustOp(t, "warp_to_saved_location", map[string]string{"locationType": "FREE_MARKET"}),
	})
	require.NoError(t, err)
	assert.True(t, moved)
}

// A static-only outcome does not report a move.
func TestExecuteOperations_StaticOnlyReportsNotMoved(t *testing.T) {
	sp := &fakeSagaProcessor{}
	e, _ := newTestExecutor(t, sp)

	moved, err := e.ExecuteOperations(testField(), 100, 3, []operation.Model{
		mustOp(t, "play_portal_sound", nil),
	})
	require.NoError(t, err)
	assert.False(t, moved)
}

// FR-2.3 strengthened: a warp that fails BEFORE its saga is created reports
// movedCharacter == false, so the caller still unlocks the client. There is no
// saga in flight to fail and release them.
func TestExecuteOperations_WarpDispatchFailureReportsNotMoved(t *testing.T) {
	sp := &fakeSagaProcessor{err: errors.New("kafka unavailable")}
	e, _ := newTestExecutor(t, sp)

	moved, err := e.ExecuteOperations(testField(), 100, 3, []operation.Model{
		mustOp(t, "warp", map[string]string{"mapId": "200090500"}),
	})
	require.Error(t, err)
	assert.False(t, moved, "a warp that never dispatched has not moved the character")
}

// A warp validation error (missing mapId) also reports not-moved.
func TestExecuteOperations_WarpParamErrorReportsNotMoved(t *testing.T) {
	sp := &fakeSagaProcessor{}
	e, _ := newTestExecutor(t, sp)

	moved, err := e.ExecuteOperations(testField(), 100, 3, []operation.Model{
		mustOp(t, "warp", map[string]string{"portalId": "1"}),
	})
	require.Error(t, err)
	assert.False(t, moved)
}

// A successful warp followed by a failing static operation still reports moved:
// the warp is in flight and its SET_FIELD will unlock the client.
func TestExecuteOperations_MovedStickyAcrossLaterError(t *testing.T) {
	sp := &fakeSagaProcessor{}
	e, _ := newTestExecutor(t, sp)

	moved, err := e.ExecuteOperations(testField(), 100, 3, []operation.Model{
		mustOp(t, "warp", map[string]string{"mapId": "200090500"}),
		mustOp(t, "start_quest", nil), // missing required params -> error
	})
	require.Error(t, err)
	assert.True(t, moved, "the warp already dispatched; the SET_FIELD will unlock the client")
}
