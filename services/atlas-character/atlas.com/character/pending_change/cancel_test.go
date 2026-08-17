package pending_change

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// TestCancelForCharacterAndType covers the processor-level primitive the new
// self-scoped cancel route (task-227 client-cancel addendum) delegates to.
func TestCancelForCharacterAndType(t *testing.T) {
	t.Run("cancels the pending name change and refunds exactly once, reason player_cancelled", func(t *testing.T) {
		db := newProcessorTestDB(t)
		characterId := seedCharacter(t, db, "Mike", world.Id(0))
		assetId := uint32(9100)
		p := NewProcessor(testLogger(t), testContext(t), db)
		m, err := p.CreateAndEmit(uuid.New(), characterId, TypeNameChange, "November", world.Id(0), &assetId)
		require.NoError(t, err)

		awardsBefore := countOutboxMessagesMatching(t, db, "award_asset")

		got, moved, err := p.CancelForCharacterAndType(characterId, TypeNameChange)
		require.NoError(t, err)
		require.True(t, moved)
		require.Equal(t, m.Id(), got.Id())
		require.Equal(t, StatusCancelled, got.Status())
		require.Equal(t, "player_cancelled", got.Reason())

		require.Equal(t, 1, countOutboxMessagesMatching(t, db, "award_asset")-awardsBefore)
	})

	// The discriminating test: it must fail against a version of the filter
	// that ignores type and cancels the first pending record it finds.
	t.Run("type filter: cancelling WORLD_TRANSFER does not touch a co-pending NAME_CHANGE", func(t *testing.T) {
		db := newProcessorTestDB(t)
		characterId := seedCharacter(t, db, "Oscar", world.Id(0))
		p := NewProcessor(testLogger(t), testContext(t), db).withTransferEligibilityGates(passingGateDeps())

		nameChange, err := p.CreateAndEmit(uuid.New(), characterId, TypeNameChange, "Papa", world.Id(0), nil)
		require.NoError(t, err)
		worldTransfer, err := p.CreateAndEmit(uuid.New(), characterId, TypeWorldTransfer, "", world.Id(2), nil)
		require.NoError(t, err)

		got, moved, err := p.CancelForCharacterAndType(characterId, TypeWorldTransfer)
		require.NoError(t, err)
		require.True(t, moved)
		require.Equal(t, worldTransfer.Id(), got.Id())
		require.Equal(t, StatusCancelled, got.Status())

		stillPending, err := p.GetById(nameChange.Id())
		require.NoError(t, err)
		require.Equal(t, StatusPending, stillPending.Status())
	})

	t.Run("nothing pending of that type: zero model, not moved, no error", func(t *testing.T) {
		db := newProcessorTestDB(t)
		characterId := seedCharacter(t, db, "Quebec", world.Id(0))
		p := NewProcessor(testLogger(t), testContext(t), db)

		got, moved, err := p.CancelForCharacterAndType(characterId, TypeNameChange)
		require.NoError(t, err)
		require.False(t, moved)
		require.Equal(t, uuid.Nil, got.Id())
	})
}

func cancelBody(t *testing.T, in CancelInputRestModel) []byte {
	t.Helper()
	b, err := jsonapi.Marshal(in)
	require.NoError(t, err)
	return b
}

func cancelForCharacterUrl(base string, characterId uint32) string {
	return fmt.Sprintf("%s/characters/%d/pending-changes/cancel", base, characterId)
}

func deletePendingChangeUrl(base string, characterId uint32, id uuid.UUID) string {
	return fmt.Sprintf("%s/characters/%d/pending-changes/%s", base, characterId, id)
}

// TestCancelPendingChangeForCharacterRoute drives the new self-scoped POST
// .../pending-changes/cancel route through the real resource router.
func TestCancelPendingChangeForCharacterRoute(t *testing.T) {
	db := newProcessorTestDB(t)
	characterId := seedCharacter(t, db, "Romeo", world.Id(0))
	_, err := NewProcessor(testLogger(t), testContext(t), db).
		CreateAndEmit(uuid.New(), characterId, TypeNameChange, "Sierra", world.Id(0), nil)
	require.NoError(t, err)

	router := setupPendingChangeResourceRouter(db)
	srv := httptest.NewServer(router)
	defer srv.Close()

	url := cancelForCharacterUrl(srv.URL, characterId)

	// First cancel succeeds.
	first := pendingChangeRequestWithTenant(t, http.MethodPost, url, cancelBody(t, CancelInputRestModel{Type: TypeNameChange}))
	resp, err := (&http.Client{}).Do(first)
	require.NoError(t, err)
	_ = resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	got, err := NewProcessor(testLogger(t), testContext(t), db).GetByCharacterId(characterId)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, StatusCancelled, got[0].Status())
	require.Equal(t, "player_cancelled", got[0].Reason())

	// A repeat cancel finds nothing PENDING of that type any more -- 404, not
	// 409. This deliberately differs from the id-based DELETE route (see
	// TestDeletePendingChangeIsConflictOnceTerminal): DELETE resolves by a
	// known id regardless of status, so a repeat is a genuine conflict on a
	// known record; this route resolves by (characterId, type) through
	// getPendingByCharacterId, which only ever sees PENDING rows, so once
	// nothing is pending there is nothing to conflict with -- it is
	// indistinguishable from "never had one," and correctly reported as such.
	second := pendingChangeRequestWithTenant(t, http.MethodPost, url, cancelBody(t, CancelInputRestModel{Type: TypeNameChange}))
	resp2, err := (&http.Client{}).Do(second)
	require.NoError(t, err)
	_ = resp2.Body.Close()
	require.Equal(t, http.StatusNotFound, resp2.StatusCode)

	// Nothing pending of a type never requested at all is also 404.
	third := pendingChangeRequestWithTenant(t, http.MethodPost, url, cancelBody(t, CancelInputRestModel{Type: TypeWorldTransfer}))
	resp3, err := (&http.Client{}).Do(third)
	require.NoError(t, err)
	_ = resp3.Body.Close()
	require.Equal(t, http.StatusNotFound, resp3.StatusCode)
}

// TestCrossCharacterCannotCancelViaNewRoute proves ownership holds by
// construction on the new self-scoped route: character B's request is scoped
// to character B's own records via getPendingByCharacterId, so it can never
// even see character A's pending record, let alone cancel it.
func TestCrossCharacterCannotCancelViaNewRoute(t *testing.T) {
	db := newProcessorTestDB(t)
	characterA := seedCharacter(t, db, "Tango", world.Id(0))
	characterB := seedCharacter(t, db, "Uniform", world.Id(0))

	p := NewProcessor(testLogger(t), testContext(t), db)
	m, err := p.CreateAndEmit(uuid.New(), characterA, TypeNameChange, "Victor", world.Id(0), nil)
	require.NoError(t, err)

	router := setupPendingChangeResourceRouter(db)
	srv := httptest.NewServer(router)
	defer srv.Close()

	req := pendingChangeRequestWithTenant(t, http.MethodPost, cancelForCharacterUrl(srv.URL, characterB), cancelBody(t, CancelInputRestModel{Type: TypeNameChange}))
	resp, err := (&http.Client{}).Do(req)
	require.NoError(t, err)
	_ = resp.Body.Close()
	require.Equal(t, http.StatusNotFound, resp.StatusCode)

	stillPending, err := p.GetById(m.Id())
	require.NoError(t, err)
	require.Equal(t, StatusPending, stillPending.Status())
}

// TestCrossCharacterCannotCancelViaDeleteRoute is the red/green evidence for
// the ownership fix on the pre-existing id-based DELETE route
// (handleCancelPendingChange). Before task-227's client-cancel addendum, this
// route resolved purely by {id} with no check that the record belonged to
// the path's {characterId} -- harmless while the only caller was a trusted
// operator panel, not harmless once a second, client-driven cancel path
// exists beside it. This test must be seen FAILING against the pre-fix
// handler and passing after; both runs are recorded verbatim in the task
// report.
func TestCrossCharacterCannotCancelViaDeleteRoute(t *testing.T) {
	db := newProcessorTestDB(t)
	characterA := seedCharacter(t, db, "Whiskey", world.Id(0))
	characterB := seedCharacter(t, db, "Xray", world.Id(0))

	p := NewProcessor(testLogger(t), testContext(t), db)
	m, err := p.CreateAndEmit(uuid.New(), characterA, TypeNameChange, "Yankee", world.Id(0), nil)
	require.NoError(t, err)

	router := setupPendingChangeResourceRouter(db)
	srv := httptest.NewServer(router)
	defer srv.Close()

	// Character B issues a DELETE against character A's pending-change id,
	// scoped under B's own characterId path segment.
	req := pendingChangeRequestWithTenant(t, http.MethodDelete, deletePendingChangeUrl(srv.URL, characterB, m.Id()), nil)
	resp, err := (&http.Client{}).Do(req)
	require.NoError(t, err)
	_ = resp.Body.Close()
	require.Equal(t, http.StatusNotFound, resp.StatusCode, "character B must not be able to cancel character A's pending change")

	stillPending, err := p.GetById(m.Id())
	require.NoError(t, err)
	require.Equal(t, StatusPending, stillPending.Status(), "character A's record must be untouched by character B's cross-character DELETE attempt")
}
