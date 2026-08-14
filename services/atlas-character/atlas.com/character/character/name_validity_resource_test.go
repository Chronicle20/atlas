package character_test

import (
	"atlas-character/character"
	"atlas-character/kafka/message"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func buildNameValidityServer(db *gorm.DB) *httptest.Server {
	r := mux.NewRouter()
	ri := character.InitResource(GetServer())(db)
	ri(r, testLogger())
	return httptest.NewServer(r)
}

func addTenantHeaders(req *http.Request, tm *tenant.Model) {
	req.Header.Set("TENANT_ID", tm.Id().String())
	req.Header.Set("REGION", tm.Region())
	req.Header.Set("MAJOR_VERSION", fmt.Sprintf("%d", tm.MajorVersion()))
	req.Header.Set("MINOR_VERSION", fmt.Sprintf("%d", tm.MinorVersion()))
}

func doNameValidityRequestWithTenant(t *testing.T, ts *httptest.Server, tm *tenant.Model, name string, worldId string) *http.Response {
	t.Helper()
	url := fmt.Sprintf("%s/characters/name-validity?name=%s&worldId=%s", ts.URL, name, worldId)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("Failed to build request: %v", err)
	}
	addTenantHeaders(req, tm)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	return resp
}

func parseNameValidityResponse(t *testing.T, resp *http.Response) character.NameValidityResponse {
	t.Helper()
	var r character.NameValidityResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	_ = resp.Body.Close()
	return r
}

func seedCharacterWithTenant(t *testing.T, db *gorm.DB, tm *tenant.Model, name string, wid world.Id) {
	t.Helper()
	tctx := tenant.WithContext(context.Background(), *tm)
	input := character.NewModelBuilder().SetAccountId(1).SetWorldId(wid).SetName(name).SetLevel(1).Build()
	_, err := character.NewProcessor(testLogger(), tctx, db).Create(message.NewBuffer())(uuid.New(), input, 0)
	if err != nil {
		t.Fatalf("Failed to seed character %s in world %d: %v", name, wid, err)
	}
}

// newFixedTenant creates a tenant with a fixed UUID so seed + HTTP request share the same tenant.
func newFixedTenant() *tenant.Model {
	id := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	tm, _ := tenant.Create(id, "GMS", 83, 1)
	return &tm
}

// TestNameValidity_ValidName — valid name, no duplicate → 200 {"valid":true}
func TestNameValidity_ValidName(t *testing.T) {
	db := testDatabase(t)
	tm := newFixedTenant()
	ts := buildNameValidityServer(db)
	defer ts.Close()

	resp := doNameValidityRequestWithTenant(t, ts, tm, "Hero", "1")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
	r := parseNameValidityResponse(t, resp)
	if !r.Valid {
		t.Fatalf("Expected valid=true, got false (reason=%s)", r.Reason)
	}
	if r.Reason != "" {
		t.Fatalf("Expected empty reason, got %q", r.Reason)
	}
}

// TestNameValidity_TooShort — name "ab" → 200 {"valid":false,"reason":"length"}
func TestNameValidity_TooShort(t *testing.T) {
	db := testDatabase(t)
	tm := newFixedTenant()
	ts := buildNameValidityServer(db)
	defer ts.Close()

	resp := doNameValidityRequestWithTenant(t, ts, tm, "ab", "1")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
	r := parseNameValidityResponse(t, resp)
	if r.Valid {
		t.Fatalf("Expected valid=false")
	}
	if r.Reason != "length" {
		t.Fatalf("Expected reason=length, got %q", r.Reason)
	}
}

// TestNameValidity_InvalidChars — name "ab!@" → 200 {"valid":false,"reason":"regex"}
func TestNameValidity_InvalidChars(t *testing.T) {
	db := testDatabase(t)
	tm := newFixedTenant()
	ts := buildNameValidityServer(db)
	defer ts.Close()

	resp := doNameValidityRequestWithTenant(t, ts, tm, "ab!@", "1")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
	r := parseNameValidityResponse(t, resp)
	if r.Valid {
		t.Fatalf("Expected valid=false")
	}
	if r.Reason != "regex" {
		t.Fatalf("Expected reason=regex, got %q", r.Reason)
	}
}

// TestNameValidity_DuplicateInTargetWorld — duplicate in world 1 → invalid
func TestNameValidity_DuplicateInTargetWorld(t *testing.T) {
	db := testDatabase(t)
	tm := newFixedTenant()
	seedCharacterWithTenant(t, db, tm, "Hero", world.Id(1))
	ts := buildNameValidityServer(db)
	defer ts.Close()

	resp := doNameValidityRequestWithTenant(t, ts, tm, "Hero", "1")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
	r := parseNameValidityResponse(t, resp)
	if r.Valid {
		t.Fatalf("Expected valid=false for duplicate in same world")
	}
	if r.Reason != "duplicate" {
		t.Fatalf("Expected reason=duplicate, got %q", r.Reason)
	}
}

// TestNameValidity_DuplicateInOtherWorld — duplicate only in world 2, query world 1 → valid
func TestNameValidity_DuplicateInOtherWorld(t *testing.T) {
	db := testDatabase(t)
	tm := newFixedTenant()
	seedCharacterWithTenant(t, db, tm, "Hero", world.Id(2))
	ts := buildNameValidityServer(db)
	defer ts.Close()

	resp := doNameValidityRequestWithTenant(t, ts, tm, "Hero", "1")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
	r := parseNameValidityResponse(t, resp)
	if !r.Valid {
		t.Fatalf("Expected valid=true when duplicate only in other world, got reason=%s", r.Reason)
	}
}

// TestNameValidity_MissingName — missing name param → 400
func TestNameValidity_MissingName(t *testing.T) {
	db := testDatabase(t)
	tm := newFixedTenant()
	ts := buildNameValidityServer(db)
	defer ts.Close()

	url := fmt.Sprintf("%s/characters/name-validity?worldId=1", ts.URL)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("Failed to build request: %v", err)
	}
	addTenantHeaders(req, tm)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected 400, got %d", resp.StatusCode)
	}
}

// TestNameValidity_MissingWorldId — missing worldId param → 400
func TestNameValidity_MissingWorldId(t *testing.T) {
	db := testDatabase(t)
	tm := newFixedTenant()
	ts := buildNameValidityServer(db)
	defer ts.Close()

	url := fmt.Sprintf("%s/characters/name-validity?name=Hero", ts.URL)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("Failed to build request: %v", err)
	}
	addTenantHeaders(req, tm)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected 400, got %d", resp.StatusCode)
	}
}

// TestNameValidity_MalformedWorldId — worldId="abc" → 400
func TestNameValidity_MalformedWorldId(t *testing.T) {
	db := testDatabase(t)
	tm := newFixedTenant()
	ts := buildNameValidityServer(db)
	defer ts.Close()

	resp := doNameValidityRequestWithTenant(t, ts, tm, "Hero", "abc")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected 400, got %d", resp.StatusCode)
	}
}

// TestNameValidity_InvalidScope — scope="BOGUS" → 400
func TestNameValidity_InvalidScope(t *testing.T) {
	db := testDatabase(t)
	tm := newFixedTenant()
	ts := buildNameValidityServer(db)
	defer ts.Close()

	url := fmt.Sprintf("%s/characters/name-validity?name=Hero&worldId=1&scope=BOGUS", ts.URL)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("Failed to build request: %v", err)
	}
	addTenantHeaders(req, tm)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected 400, got %d", resp.StatusCode)
	}
}

// TestNameValidity_TenantScopeRejectsOtherWorldDuplicate proves the scope
// parameter is not ignored: the same name in a different world is accepted
// under WORLD scope (today's creation behaviour) and rejected under TENANT
// scope (FR-3.2, used by name change so a later world transfer can never
// collide).
func TestNameValidity_TenantScopeRejectsOtherWorldDuplicate(t *testing.T) {
	db := testDatabase(t)
	tm := newFixedTenant()
	seedCharacterWithTenant(t, db, tm, "Foxtrot", world.Id(1))
	ts := buildNameValidityServer(db)
	defer ts.Close()

	// World scope: creation in world 0 may reuse a name held only in world 1.
	resp := doNameValidityRequestWithTenant(t, ts, tm, "Foxtrot", "0")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
	r := parseNameValidityResponse(t, resp)
	if !r.Valid {
		t.Fatalf("expected world-scoped check to allow a cross-world duplicate, got %s", r.Reason)
	}

	// Tenant scope: a name change may not.
	url := fmt.Sprintf("%s/characters/name-validity?name=Foxtrot&worldId=0&scope=TENANT", ts.URL)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("Failed to build request: %v", err)
	}
	addTenantHeaders(req, tm)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
	r = parseNameValidityResponse(t, resp)
	if r.Valid || r.Reason != "duplicate" {
		t.Fatalf("expected tenant-scoped duplicate, got valid=%v reason=%s", r.Valid, r.Reason)
	}
}

// TestNameValidity_RejectsAReservedNameInBothScopes proves a live pending
// name-change reservation blocks BOTH scopes (FR-3.3): character creation is
// the one behaviour change a name-change reservation is required to affect.
func TestNameValidity_RejectsAReservedNameInBothScopes(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	p := character.NewProcessor(testLogger(), tctx, testDatabase(t)).
		WithNameReserved(func(_ logrus.FieldLogger, _ context.Context, name string) (bool, error) {
			return strings.EqualFold(name, "Golf"), nil
		})

	for _, scope := range []character.NameScope{character.NameScopeWorld, character.NameScopeTenant} {
		res, err := p.CheckNameValidity("golf", world.Id(0), scope)
		if err != nil {
			t.Fatalf("scope %s: %v", scope, err)
		}
		if res.Valid || res.Reason != "reserved" {
			t.Fatalf("scope %s: expected reserved, got valid=%v reason=%s", scope, res.Valid, res.Reason)
		}
	}
}

// TestNameValidity_UnwiredProcessorDoesNotPanic proves an un-wired processor
// (nameReserved == nil, every pre-Task-6 NewProcessor call site) degrades
// safely instead of panicking.
func TestNameValidity_UnwiredProcessorDoesNotPanic(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	p := character.NewProcessor(testLogger(), tctx, testDatabase(t))

	res, err := p.CheckNameValidity("Hero", world.Id(0), character.NameScopeWorld)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Valid {
		t.Fatalf("expected valid=true, got reason=%s", res.Reason)
	}
}
