package tenant_test

import (
	"atlas-tenants/kafka/message"
	"atlas-tenants/scope"
	"atlas-tenants/tenant"
	"atlas-tenants/test"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"gorm.io/gorm"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// seedTenantWithEnvironment creates a tenant owned by environment directly
// at the entity layer, bypassing ProcessorImpl.Create (and its Kafka
// producer, unavailable in this unit test) - matches the existing
// testProcessor.create() / TestGetTenantsPaginates seeding precedent in
// this package.
func seedTenantWithEnvironment(t *testing.T, db *gorm.DB, environment string) tenant.Model {
	t.Helper()
	m, err := tenant.NewModelBuilder().
		SetName("Test Tenant " + environment).
		SetRegion("GMS").
		SetMajorVersion(83).
		SetMinorVersion(1).
		SetEnvironment(environment).
		Build()
	if err != nil {
		t.Fatalf("seedTenantWithEnvironment: Build() unexpected error: %v", err)
	}
	if err := tenant.CreateTenant(db, tenant.FromModel(m)); err != nil {
		t.Fatalf("seedTenantWithEnvironment: CreateTenant: %v", err)
	}
	return m
}

func TestGetAllReturnsOnlyTheCallersEnvironmentsTenants(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	seedTenantWithEnvironment(t, db, "main")
	seedTenantWithEnvironment(t, db, "pr-123")

	logger, _ := logtest.NewNullLogger()
	ctx := env.WithContext(context.Background(), env.Id("pr-123"))
	processor := tenant.NewProcessor(logger, ctx, db)
	paged, err := processor.AllProvider(model.Page{Number: 1, Size: 250})()
	if err != nil {
		t.Fatalf("AllProvider: %v", err)
	}
	if len(paged.Items) != 1 || paged.Items[0].Environment() != "pr-123" {
		t.Fatalf("got %d tenants (%v), want pr-123's only", len(paged.Items), paged.Items)
	}
}

func TestUpdateTenantRejectsCrossEnvironmentWrite(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	m := seedTenantWithEnvironment(t, db, "main")

	caller := env.WithContext(context.Background(), env.Id("pr-123"))
	e := tenant.FromModel(m)
	err := tenant.UpdateTenant(caller, db, e)
	if !errors.Is(err, scope.ErrCrossEnvironmentWrite) {
		t.Fatalf("UpdateTenant() error = %v, want scope.ErrCrossEnvironmentWrite", err)
	}
}

func TestDeleteTenantRejectsCrossEnvironmentWrite(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	m := seedTenantWithEnvironment(t, db, "main")

	caller := env.WithContext(context.Background(), env.Id("pr-123"))
	err := tenant.DeleteTenant(caller, db, m.Id())
	if !errors.Is(err, scope.ErrCrossEnvironmentWrite) {
		t.Fatalf("DeleteTenant() error = %v, want scope.ErrCrossEnvironmentWrite", err)
	}
}

func TestGetAllWithLegacyEnvironmentReturnsEverything(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	seedTenantWithEnvironment(t, db, "main")
	seedTenantWithEnvironment(t, db, "pr-123")

	logger, _ := logtest.NewNullLogger()
	processor := tenant.NewProcessor(logger, context.Background(), db)
	paged, err := processor.AllProvider(model.Page{Number: 1, Size: 250})()
	if err != nil {
		t.Fatalf("AllProvider: %v", err)
	}
	if len(paged.Items) != 2 {
		t.Fatalf("legacy GetAll returned %d, want 2 (FR-1.8)", len(paged.Items))
	}
}

// TestProcessorUpdateRejectsCrossEnvironmentWrite closes the gap the fix-1
// review reproduced: ProcessorImpl.Update's own pre-lookup used to be
// scope.Strict-filtered (GetByIdProvider), so a cross-environment target
// never reached UpdateTenant's scope.AuthorizeWrite at all and returned a
// bare "tenant not found" that errors.Is could never match. Driving the
// processor directly (not tenant.UpdateTenant at the administrator layer,
// which TestUpdateTenantRejectsCrossEnvironmentWrite already covers) is what
// would have caught that hole.
func TestProcessorUpdateRejectsCrossEnvironmentWrite(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	m := seedTenantWithEnvironment(t, db, "main")

	logger, _ := logtest.NewNullLogger()
	caller := env.WithContext(context.Background(), env.Id("pr-123"))
	processor := tenant.NewProcessor(logger, caller, db)
	mb := message.NewBuffer()
	_, err := processor.Update(mb)(m.Id(), "renamed", "GMS", 83, 1)
	if !errors.Is(err, scope.ErrCrossEnvironmentWrite) {
		t.Fatalf("Processor.Update cross-env error = %v, want scope.ErrCrossEnvironmentWrite", err)
	}
}

// TestProcessorDeleteRejectsCrossEnvironmentWrite is Delete's counterpart to
// TestProcessorUpdateRejectsCrossEnvironmentWrite above.
func TestProcessorDeleteRejectsCrossEnvironmentWrite(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	m := seedTenantWithEnvironment(t, db, "main")

	logger, _ := logtest.NewNullLogger()
	caller := env.WithContext(context.Background(), env.Id("pr-123"))
	processor := tenant.NewProcessor(logger, caller, db)
	mb := message.NewBuffer()
	err := processor.Delete(mb)(m.Id())
	if !errors.Is(err, scope.ErrCrossEnvironmentWrite) {
		t.Fatalf("Processor.Delete cross-env error = %v, want scope.ErrCrossEnvironmentWrite", err)
	}
}

// TestProcessorUpdateSameEnvironmentSucceeds is a regression guard: making
// the pre-lookup unscoped must not stop a same-environment write from
// working.
func TestProcessorUpdateSameEnvironmentSucceeds(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	m := seedTenantWithEnvironment(t, db, "pr-123")

	logger, _ := logtest.NewNullLogger()
	caller := env.WithContext(context.Background(), env.Id("pr-123"))
	processor := tenant.NewProcessor(logger, caller, db)
	mb := message.NewBuffer()
	updated, err := processor.Update(mb)(m.Id(), "renamed", "GMS", 83, 1)
	if err != nil {
		t.Fatalf("Processor.Update same-environment: unexpected error: %v", err)
	}
	if updated.Name() != "renamed" {
		t.Fatalf("updated.Name() = %q, want %q", updated.Name(), "renamed")
	}
}

// TestProcessorUpdateLegacyEnvironmentSucceeds is a regression guard for the
// no-environment-in-context caller (FR-1.8): it must still be able to
// update a tenant that itself carries the legacy (empty) environment,
// matching pre-task-232 behaviour. scope.AuthorizeWrite requires
// caller == target exactly (scope.go), and the legacy caller is always ""
// - so the seeded row's environment must be "" too, not a named one like
// "main".
func TestProcessorUpdateLegacyEnvironmentSucceeds(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	m := seedTenantWithEnvironment(t, db, "")

	logger, _ := logtest.NewNullLogger()
	processor := tenant.NewProcessor(logger, context.Background(), db)
	mb := message.NewBuffer()
	updated, err := processor.Update(mb)(m.Id(), "renamed", "GMS", 83, 1)
	if err != nil {
		t.Fatalf("Processor.Update legacy environment: unexpected error: %v", err)
	}
	if updated.Name() != "renamed" {
		t.Fatalf("updated.Name() = %q, want %q", updated.Name(), "renamed")
	}
}

// TestProcessorDeleteSameEnvironmentSucceeds mirrors
// TestProcessorUpdateSameEnvironmentSucceeds for Delete.
func TestProcessorDeleteSameEnvironmentSucceeds(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	m := seedTenantWithEnvironment(t, db, "pr-123")

	logger, _ := logtest.NewNullLogger()
	caller := env.WithContext(context.Background(), env.Id("pr-123"))
	processor := tenant.NewProcessor(logger, caller, db)
	mb := message.NewBuffer()
	if err := processor.Delete(mb)(m.Id()); err != nil {
		t.Fatalf("Processor.Delete same-environment: unexpected error: %v", err)
	}
}

// TestProcessorDeleteLegacyEnvironmentSucceeds mirrors
// TestProcessorUpdateLegacyEnvironmentSucceeds for Delete.
func TestProcessorDeleteLegacyEnvironmentSucceeds(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	m := seedTenantWithEnvironment(t, db, "")

	logger, _ := logtest.NewNullLogger()
	processor := tenant.NewProcessor(logger, context.Background(), db)
	mb := message.NewBuffer()
	if err := processor.Delete(mb)(m.Id()); err != nil {
		t.Fatalf("Processor.Delete legacy environment: unexpected error: %v", err)
	}
}

// tenantUpdateBody builds the JSON:API request body ParseInput expects for
// PATCH /tenants/{tenantId}.
func tenantUpdateBody(t *testing.T, id, name, environment string) []byte {
	t.Helper()
	doc := map[string]any{
		"data": map[string]any{
			"type": "tenants",
			"id":   id,
			"attributes": map[string]any{
				"name":         name,
				"region":       "GMS",
				"majorVersion": 83,
				"minorVersion": 1,
				"environment":  environment,
			},
		},
	}
	body, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("tenantUpdateBody: marshal: %v", err)
	}
	return body
}

// TestUpdateTenantHandlerCrossEnvironmentReturns403 is the end-to-end
// assertion the fix-1 review required: a correct-looking
// scope.ErrCrossEnvironmentWrite value is not proof the HTTP response is
// right. Before this fix, ProcessorImpl.Update's scoped pre-lookup
// short-circuited to a plain "tenant not found" that rest.WriteErrorResponse
// could not recognise, so the actual response was 500, not 403.
func TestUpdateTenantHandlerCrossEnvironmentReturns403(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)
	logger, _ := logtest.NewNullLogger()

	m := seedTenantWithEnvironment(t, db, "main")

	router := mux.NewRouter()
	tenant.RegisterRoutes(db)(testServerInformation{})(router, logger)

	body := tenantUpdateBody(t, m.Id().String(), "renamed", "main")
	req, err := http.NewRequest(http.MethodPatch, "/tenants/"+m.Id().String(), bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set(env.Key, "pr-123")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403, body=%s", rr.Code, rr.Body.String())
	}
}

// TestDeleteTenantHandlerCrossEnvironmentReturns403 is Delete's counterpart
// to TestUpdateTenantHandlerCrossEnvironmentReturns403 above.
func TestDeleteTenantHandlerCrossEnvironmentReturns403(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)
	logger, _ := logtest.NewNullLogger()

	m := seedTenantWithEnvironment(t, db, "main")

	router := mux.NewRouter()
	tenant.RegisterRoutes(db)(testServerInformation{})(router, logger)

	req, err := http.NewRequest(http.MethodDelete, "/tenants/"+m.Id().String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set(env.Key, "pr-123")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403, body=%s", rr.Code, rr.Body.String())
	}
}

// TestUpdateTenantHandlerSameEnvironmentSucceeds is a regression guard: the
// 403 mapping above must not fire for a same-environment write.
func TestUpdateTenantHandlerSameEnvironmentSucceeds(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)
	logger, _ := logtest.NewNullLogger()

	m := seedTenantWithEnvironment(t, db, "pr-123")

	router := mux.NewRouter()
	tenant.RegisterRoutes(db)(testServerInformation{})(router, logger)

	body := tenantUpdateBody(t, m.Id().String(), "renamed", "pr-123")
	req, err := http.NewRequest(http.MethodPatch, "/tenants/"+m.Id().String(), bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set(env.Key, "pr-123")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rr.Code, rr.Body.String())
	}
}

// TestDeleteTenantHandlerSameEnvironmentSucceeds mirrors
// TestUpdateTenantHandlerSameEnvironmentSucceeds for Delete.
func TestDeleteTenantHandlerSameEnvironmentSucceeds(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)
	logger, _ := logtest.NewNullLogger()

	m := seedTenantWithEnvironment(t, db, "pr-123")

	router := mux.NewRouter()
	tenant.RegisterRoutes(db)(testServerInformation{})(router, logger)

	req, err := http.NewRequest(http.MethodDelete, "/tenants/"+m.Id().String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set(env.Key, "pr-123")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204, body=%s", rr.Code, rr.Body.String())
	}
}
