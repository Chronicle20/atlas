package family

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type testServerInformation struct{}

func (t *testServerInformation) GetBaseURL() string { return "http://localhost:8080" }
func (t *testServerInformation) GetPrefix() string  { return "/api/" }

var _ jsonapi.ServerInformation = &testServerInformation{}

func setupFamilyRouter(db *gorm.DB) *mux.Router {
	r := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	ri := InitResource(&testServerInformation{})(db)
	ri(r, l)
	return r
}

func requestWithTenant(method, url string, tenantId uuid.UUID) *http.Request {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("TENANT_ID", tenantId.String())
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")
	return req
}

func seedFamilyMember(t *testing.T, db *gorm.DB, tenantId uuid.UUID, id, characterId uint32, seniorId *uint32, juniorIds []uint32) {
	t.Helper()
	now := time.Now()
	require.NoError(t, db.Create(&Entity{
		ID: id, TenantId: tenantId, CharacterId: characterId, SeniorId: seniorId,
		JuniorIds: juniorIds, Level: 10, World: 0,
		CreatedAt: now, UpdatedAt: now,
	}).Error)
}

// TestGetFamilyTreePaginates drives GET /families/tree/{characterId} through
// the real resource router (InitResource) against an in-memory tenant-scoped
// DB. The tree for character 20 is: self(20) + senior(10) + juniors of 20
// (30, 40) + sibling (21, another junior of 10) = 5 members, deterministically
// ordered by CharacterId (10, 20, 21, 30, 40) regardless of DB fetch order.
func TestGetFamilyTreePaginates(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	tenantId := uuid.New()

	ten := uint32(10)
	twenty := uint32(20)
	seedFamilyMember(t, db, tenantId, 1, 10, nil, []uint32{20, 21})
	seedFamilyMember(t, db, tenantId, 2, 20, &ten, []uint32{30, 40})
	seedFamilyMember(t, db, tenantId, 3, 21, &ten, nil)
	seedFamilyMember(t, db, tenantId, 4, 30, &twenty, nil)
	seedFamilyMember(t, db, tenantId, 5, 40, &twenty, nil)

	srv := httptest.NewServer(setupFamilyRouter(db))
	defer srv.Close()

	t.Run("FirstPageOfTwo", func(t *testing.T) {
		url := fmt.Sprintf("%s/families/tree/20?page[number]=1&page[size]=2", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

		require.NotNil(t, doc.Data)
		require.Len(t, doc.Data.DataArray, 2)

		// stable-sorted by CharacterId: page 1 must be {10, 20}, not
		// whatever order the graph traversal (self, senior, juniors,
		// siblings) happened to build the slice in.
		var firstAttrs, secondAttrs struct {
			CharacterId uint32 `json:"characterId"`
		}
		require.NoError(t, json.Unmarshal(doc.Data.DataArray[0].Attributes, &firstAttrs))
		require.NoError(t, json.Unmarshal(doc.Data.DataArray[1].Attributes, &secondAttrs))
		assert.EqualValues(t, 10, firstAttrs.CharacterId)
		assert.EqualValues(t, 20, secondAttrs.CharacterId)

		require.NotNil(t, doc.Meta)
		assert.EqualValues(t, 5, doc.Meta["total"])
		page := doc.Meta["page"].(map[string]interface{})
		assert.EqualValues(t, 3, page["last"])

		require.NotNil(t, doc.Links)
		assert.Contains(t, doc.Links, "next")
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/families/tree/20?page[size]=0", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("LegacyLimitParamIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/families/tree/20?limit=5", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("PastLastPageReturnsEmptyWithPrevAtLast", func(t *testing.T) {
		url := fmt.Sprintf("%s/families/tree/20?page[number]=99&page[size]=2", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

		require.NotNil(t, doc.Data)
		assert.Len(t, doc.Data.DataArray, 0)

		require.NotNil(t, doc.Links)
		require.Contains(t, doc.Links, "prev")
		assert.Contains(t, doc.Links["prev"].Href, "page%5Bnumber%5D=3")
		assert.NotContains(t, doc.Links, "next")
	})
}

// TestBreakLinkHandlerBadPageParamsIsBadRequest verifies DELETE
// /families/links/{characterId} validates page[number]/page[size] before
// doing any work — same param-parsing gate getFamilyTreeHandler uses. This
// path returns before the processor's BreakLinkAndEmit call, so it also
// requires no live Kafka broker.
func TestBreakLinkHandlerBadPageParamsIsBadRequest(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	tenantId := uuid.New()

	srv := httptest.NewServer(setupFamilyRouter(db))
	defer srv.Close()

	url := fmt.Sprintf("%s/families/links/20?page[size]=0", srv.URL)
	req := requestWithTenant(http.MethodDelete, url, tenantId)

	resp, err := (&http.Client{}).Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// TestBreakLinkHandlerLegacyLimitParamIsBadRequest mirrors the tree
// handler's rejection of the legacy ?limit= param.
func TestBreakLinkHandlerLegacyLimitParamIsBadRequest(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	tenantId := uuid.New()

	srv := httptest.NewServer(setupFamilyRouter(db))
	defer srv.Close()

	url := fmt.Sprintf("%s/families/links/20?limit=5", srv.URL)
	req := requestWithTenant(http.MethodDelete, url, tenantId)

	resp, err := (&http.Client{}).Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// TestBreakLinkHandlerMemberNotFoundIs404 and
// TestBreakLinkHandlerNoLinkToBreakIsConflict drive the two error branches
// that return before BreakLink ever touches the Kafka emission buffer
// (ErrMemberNotFound / ErrNoLinkToBreak both return out of
// ProcessorImpl.BreakLink before its `buf.Put` calls), so — like the bad
// page-param case above — they need no live broker.
func TestBreakLinkHandlerMemberNotFoundIs404(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	tenantId := uuid.New()

	srv := httptest.NewServer(setupFamilyRouter(db))
	defer srv.Close()

	url := fmt.Sprintf("%s/families/links/999", srv.URL)
	req := requestWithTenant(http.MethodDelete, url, tenantId)

	resp, err := (&http.Client{}).Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestBreakLinkHandlerNoLinkToBreakIsConflict(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	tenantId := uuid.New()
	seedFamilyMember(t, db, tenantId, 20, 20, nil, nil)

	srv := httptest.NewServer(setupFamilyRouter(db))
	defer srv.Close()

	url := fmt.Sprintf("%s/families/links/20", srv.URL)
	req := requestWithTenant(http.MethodDelete, url, tenantId)

	resp, err := (&http.Client{}).Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusConflict, resp.StatusCode)
}

// TestBreakLinkPaginatesUpdatedMembers exercises the exact sort +
// paginate.Slice + paginate.EnvelopeFor pipeline breakLinkHandler applies
// to BreakLinkAndEmit's result (resource.go breakLinkHandler, task-117 Task
// 25). It drives the shared logic through Processor.BreakLink(nil) rather
// than the full HTTP route / BreakLinkAndEmit, because BreakLinkAndEmit is
// a thin message.EmitWithResult wrapper around this identical BreakLink
// call (see processor.go) that additionally emits a Kafka event on success
// — and every AndEmit handler in this service (add_junior included)
// requires a live, reachable Kafka broker to complete without error; none
// is available in this unit-test environment, and this service has no
// injectable-producer test seam (unlike atlas-doors' `emit` field or
// atlas-marriages' WithProducer). BreakLink already treats buf==nil as
// "skip event emission" (see the `if buf != nil` guard around every
// `buf.Put` call), so calling it directly exercises the identical DB
// mutation and result-set construction BreakLinkAndEmit would have
// produced, letting this test verify the *pagination* logic — the actual
// change under test — against real DB-backed data without a broker
// dependency that predates and is orthogonal to this task.
//
// The break target (10) has juniors only (no senior) so the
// updatedMembers set stays duplicate-free: BreakLink's dedup-append
// fallback (processor.go, "Update the member in the result" loop, ~line
// 383) only scans for the member's own CharacterId among *already
// appended* entries and otherwise unconditionally appends it again keyed
// off whether the *last* entry happens to be the member — that check is
// wrong whenever a member has BOTH a senior and juniors (the member gets
// appended once by the HasSenior branch, then juniors get appended after
// it, so the "last entry" check misses the earlier in-place replace and
// double-appends the member). Confirmed empirically: breaking link 20
// (which has both a senior and juniors) yields a 5-entry slice with 20
// appearing twice, not the expected 4. That's a pre-existing bug in
// BreakLink's result-set construction, orthogonal to this task's response-
// envelope fix — not touched here.
func TestBreakLinkPaginatesUpdatedMembers(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	tenantId := uuid.New()

	// entity.ID must equal CharacterId here: Processor.GetByCharacterId
	// (used internally by BreakLink for the member/senior/junior lookups)
	// resolves by primary key id via provider.GetByIdProvider, not by the
	// character_id column — a pre-existing quirk in this service unrelated
	// to this task; seedFamilyMember's (id, characterId) pair must match
	// for BreakLink's internal lookups to succeed.
	seedFamilyMember(t, db, tenantId, 10, 10, nil, []uint32{20, 21})
	seedFamilyMember(t, db, tenantId, 20, 20, uint32Ptr(10), nil)
	seedFamilyMember(t, db, tenantId, 21, 21, uint32Ptr(10), nil)

	tm, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tm)

	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)

	updatedMembers, err := NewProcessor(l, ctx, db).BreakLink(nil)(10, "test")()
	require.NoError(t, err)
	// junior(20, senior cleared) + junior(21, senior cleared) + member(10,
	// juniors cleared) = 3 members.
	require.Len(t, updatedMembers, 3)

	sorted := make([]FamilyMember, len(updatedMembers))
	copy(sorted, updatedMembers)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].CharacterId() < sorted[j].CharacterId()
	})

	page, err := paginate.ParseParams(url.Values{"page[number]": {"1"}, "page[size]": {"2"}}, paginate.MaxPageSize, paginate.MaxPageSize)
	require.NoError(t, err)

	paged := paginate.Slice(sorted, page)
	require.Len(t, paged.Items, 2)
	assert.EqualValues(t, 10, paged.Items[0].CharacterId())
	assert.EqualValues(t, 20, paged.Items[1].CharacterId())

	env := paginate.EnvelopeFor(paged)
	assert.Equal(t, 3, env.Total)
	assert.Equal(t, 1, env.PageNumber)
	assert.Equal(t, 2, env.LastPage())
}

// uint32Ptr returns a pointer to the given uint32, for building SeniorId
// fields in test fixtures.
func uint32Ptr(v uint32) *uint32 {
	return &v
}
