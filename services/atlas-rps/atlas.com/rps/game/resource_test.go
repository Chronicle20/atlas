package game_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"atlas-rps/game"
	"atlas-rps/rest"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newResourceRouter wires game.InitResource against a fresh mux.Router, with
// newProcessor building a real NewProcessorWithLadder-backed Processor for
// every request (never the ErrLadderNotConfigured shell), so GET's
// current-prize resolution works exactly like production wiring.
func newResourceRouter(ladder game.Ladder) *mux.Router {
	router := mux.NewRouter()
	newProcessor := func(l logrus.FieldLogger, ctx context.Context) game.Processor {
		return game.NewProcessorWithLadder(l, ctx, game.DefaultThrowSource, ladderProviderFor(ladder))
	}
	game.InitResource(rest.GetServer(), newProcessor)(router, testLogger())
	return router
}

func addTenantHeaders(req *http.Request, ten tenant.Model) {
	req.Header.Set(tenant.ID, ten.Id().String())
	req.Header.Set(tenant.Region, ten.Region())
	req.Header.Set(tenant.MajorVersion, strconv.Itoa(int(ten.MajorVersion())))
	req.Header.Set(tenant.MinorVersion, strconv.Itoa(int(ten.MinorVersion())))
}

func postStartGame(t *testing.T, router *mux.Router, ten tenant.Model, characterId uint32) *httptest.ResponseRecorder {
	t.Helper()
	body := []byte(fmt.Sprintf(
		`{"data":{"type":"rps-games","attributes":{"characterId":%d,"worldId":0,"channelId":1,"npcId":9020000}}}`,
		characterId,
	))
	req := httptest.NewRequest(http.MethodPost, "/rps/games", bytes.NewReader(body))
	addTenantHeaders(req, ten)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func getGame(t *testing.T, router *mux.Router, ten tenant.Model, characterId uint32) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/rps/games/%d", characterId), nil)
	addTenantHeaders(req, ten)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

// TestResource_Post_CreatesSession verifies POST /rps/games opens a fresh
// rung-0 session and returns it as a JSON:API rps-games resource.
func TestResource_Post_CreatesSession(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	router := newResourceRouter(oneRungLadder())

	rr := postStartGame(t, router, ten, 1000)

	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	var got game.RestModel
	require.NoError(t, jsonapi.Unmarshal(rr.Body.Bytes(), &got))
	assert.Equal(t, "OPEN", got.Status)
	assert.Equal(t, 0, got.Rung)
	assert.Equal(t, uint32(1000), got.CharacterId)
	assert.Nil(t, got.Prize, "a freshly opened session has no prize yet")

	ctx := testCtx(ten)
	stored, found := game.GetRegistry().Get(ctx, 1000)
	require.True(t, found)
	assert.Equal(t, game.StatusOpen, stored.Status())
}

// TestResource_Post_SecondCallDisposesAndRecreates verifies FR-1.4: a second
// POST for a character with an already-active session disposes the stale
// one and opens a brand new rung-0 session rather than erroring.
func TestResource_Post_SecondCallDisposesAndRecreates(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	router := newResourceRouter(oneRungLadder())

	first := postStartGame(t, router, ten, 2000)
	require.Equal(t, http.StatusOK, first.Code)

	// Advance the session past rung 0 so we can tell a fresh Start actually
	// replaced it, rather than merely leaving the prior state untouched.
	ctx := testCtx(ten)
	advanced, err := game.CloneModelBuilder(mustGet(t, ctx, 2000)).SetRung(1).SetStatus(game.StatusAwaitingDecision).Build()
	require.NoError(t, err)
	game.GetRegistry().Put(ctx, advanced)

	second := postStartGame(t, router, ten, 2000)
	require.Equal(t, http.StatusOK, second.Code, second.Body.String())

	var got game.RestModel
	require.NoError(t, jsonapi.Unmarshal(second.Body.Bytes(), &got))
	assert.Equal(t, "OPEN", got.Status)
	assert.Equal(t, 0, got.Rung, "second POST must dispose the stale rung-1 session and open a fresh rung-0 one")

	stored, found := game.GetRegistry().Get(ctx, 2000)
	require.True(t, found)
	assert.Equal(t, game.StatusOpen, stored.Status())
	assert.Equal(t, 0, stored.Rung())
}

func mustGet(t *testing.T, ctx context.Context, characterId uint32) game.Model {
	t.Helper()
	m, found := game.GetRegistry().Get(ctx, characterId)
	require.True(t, found)
	return m
}

// TestResource_Get_UnknownCharacterReturns404 verifies GET returns 404 for a
// character with no active session.
func TestResource_Get_UnknownCharacterReturns404(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	router := newResourceRouter(oneRungLadder())

	rr := getGame(t, router, ten, 9999)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

// TestResource_Get_ReturnsSessionWithResolvedPrize verifies GET resolves the
// current prize from the real (non-shell) ladder for an in-progress session.
func TestResource_Get_ReturnsSessionWithResolvedPrize(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	router := newResourceRouter(oneRungLadder())
	ctx := testCtx(ten)

	m := game.NewModelBuilder(ten).
		SetCharacterId(3000).
		SetWorldId(0).
		SetChannelId(1).
		SetNpcId(9020000).
		SetRung(1).
		SetStatus(game.StatusAwaitingDecision).
		MustBuild()
	game.GetRegistry().Put(ctx, m)

	rr := getGame(t, router, ten, 3000)

	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	var got game.RestModel
	require.NoError(t, jsonapi.Unmarshal(rr.Body.Bytes(), &got))
	assert.Equal(t, "AWAITING_DECISION", got.Status)
	assert.Equal(t, 1, got.Rung)
	require.NotNil(t, got.Prize, "rung 1 has a configured prize on oneRungLadder")
	assert.EqualValues(t, 4000000, got.Prize.ItemId)
}
