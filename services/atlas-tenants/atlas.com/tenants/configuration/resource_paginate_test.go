package configuration_test

import (
	"atlas-tenants/configuration"
	"atlas-tenants/test"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"gorm.io/gorm"
)

type testServerInformation struct{}

func (t testServerInformation) GetBaseURL() string {
	return "http://localhost:8080"
}

func (t testServerInformation) GetPrefix() string {
	return ""
}

func doGetConfig(t *testing.T, router *mux.Router, path string) *httptest.ResponseRecorder {
	t.Helper()
	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

type pageDoc struct {
	Data []struct {
		Id string `json:"id"`
	} `json:"data"`
	Meta struct {
		Total int `json:"total"`
		Page  struct {
			Last int `json:"last"`
		} `json:"page"`
	} `json:"meta"`
	Links struct {
		Next *string `json:"next"`
	} `json:"links"`
}

func decodePageDoc(t *testing.T, rr *httptest.ResponseRecorder) pageDoc {
	t.Helper()
	var doc pageDoc
	if err := json.Unmarshal(rr.Body.Bytes(), &doc); err != nil {
		t.Fatalf("unmarshal: %v, body=%s", err, rr.Body.String())
	}
	return doc
}

// seedRoutesBlob stores 3 routes with ids "300", "100", "200" - in that
// literal array order - as a single configurations row, matching the real
// JSONB "data" array shape GetAllRoutesProvider decodes. The array order
// deliberately does not match ascending id order, so the handler's explicit
// sort is what makes the paged response deterministic.
func seedRoutesBlob(t *testing.T, db *gorm.DB, tenantId uuid.UUID) {
	t.Helper()
	routes := make([]map[string]interface{}, 0, 3)
	for _, id := range []string{"300", "100", "200"} {
		routes = append(routes, map[string]interface{}{
			"type": "routes",
			"id":   id,
			"attributes": map[string]interface{}{
				"name": fmt.Sprintf("Route-%s", id),
			},
		})
	}
	resourceData, err := configuration.CreateRouteJsonData(routes)
	if err != nil {
		t.Fatalf("seed marshal failed: %v", err)
	}
	entity := configuration.NewEntityBuilder().
		SetID(uuid.New()).
		SetTenantId(tenantId).
		SetResourceName("routes").
		SetResourceData(resourceData).
		Build()
	if err := configuration.CreateConfiguration(db, entity); err != nil {
		t.Fatalf("seed create failed: %v", err)
	}
}

func seedVesselsBlob(t *testing.T, db *gorm.DB, tenantId uuid.UUID) {
	t.Helper()
	vessels := make([]map[string]interface{}, 0, 3)
	for _, id := range []string{"300", "100", "200"} {
		vessels = append(vessels, map[string]interface{}{
			"type": "vessels",
			"id":   id,
			"attributes": map[string]interface{}{
				"name": fmt.Sprintf("Vessel-%s", id),
			},
		})
	}
	resourceData, err := configuration.CreateVesselJsonData(vessels)
	if err != nil {
		t.Fatalf("seed marshal failed: %v", err)
	}
	entity := configuration.NewEntityBuilder().
		SetID(uuid.New()).
		SetTenantId(tenantId).
		SetResourceName("vessels").
		SetResourceData(resourceData).
		Build()
	if err := configuration.CreateConfiguration(db, entity); err != nil {
		t.Fatalf("seed create failed: %v", err)
	}
}

func seedInstanceRoutesBlob(t *testing.T, db *gorm.DB, tenantId uuid.UUID) {
	t.Helper()
	routes := make([]map[string]interface{}, 0, 3)
	for _, id := range []string{"300", "100", "200"} {
		routes = append(routes, map[string]interface{}{
			"type": "instance-routes",
			"id":   id,
			"attributes": map[string]interface{}{
				"name": fmt.Sprintf("InstanceRoute-%s", id),
			},
		})
	}
	resourceData, err := configuration.CreateInstanceRouteJsonData(routes)
	if err != nil {
		t.Fatalf("seed marshal failed: %v", err)
	}
	entity := configuration.NewEntityBuilder().
		SetID(uuid.New()).
		SetTenantId(tenantId).
		SetResourceName("instance-routes").
		SetResourceData(resourceData).
		Build()
	if err := configuration.CreateConfiguration(db, entity); err != nil {
		t.Fatalf("seed create failed: %v", err)
	}
}

func assertFirstPageOfTwoAscending(t *testing.T, router *mux.Router, path string) {
	t.Helper()
	rr := doGetConfig(t, router, path+"?page[number]=1&page[size]=2")
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rr.Code, rr.Body.String())
	}
	doc := decodePageDoc(t, rr)
	if len(doc.Data) != 2 {
		t.Fatalf("len(data) = %d, want 2, body=%s", len(doc.Data), rr.Body.String())
	}
	if doc.Data[0].Id != "100" || doc.Data[1].Id != "200" {
		t.Fatalf("got ids [%s, %s], want [100, 200]", doc.Data[0].Id, doc.Data[1].Id)
	}
	if doc.Meta.Total != 3 {
		t.Fatalf("meta.total = %d, want 3", doc.Meta.Total)
	}
	if doc.Meta.Page.Last != 2 {
		t.Fatalf("meta.page.last = %d, want 2", doc.Meta.Page.Last)
	}
	if doc.Links.Next == nil {
		t.Fatal("expected links.next to be present")
	}

	badRR := doGetConfig(t, router, path+"?page[size]=0")
	if badRR.Code != http.StatusBadRequest {
		t.Fatalf("page[size]=0 status = %d, want 400", badRR.Code)
	}

	limitRR := doGetConfig(t, router, path+"?limit=5")
	if limitRR.Code != http.StatusBadRequest {
		t.Fatalf("?limit=5 status = %d, want 400", limitRR.Code)
	}

	pastRR := doGetConfig(t, router, path+"?page[number]=99&page[size]=2")
	if pastRR.Code != http.StatusOK {
		t.Fatalf("past-last-page status = %d, want 200, body=%s", pastRR.Code, pastRR.Body.String())
	}
	pastDoc := decodePageDoc(t, pastRR)
	if len(pastDoc.Data) != 0 {
		t.Fatalf("past-last-page len(data) = %d, want 0", len(pastDoc.Data))
	}
}

func TestGetAllRoutesPaginates(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)
	logger, _ := logtest.NewNullLogger()
	tenantId := uuid.New()
	seedRoutesBlob(t, db, tenantId)

	router := mux.NewRouter()
	configuration.RegisterRoutes(db)(testServerInformation{})(router, logger)

	assertFirstPageOfTwoAscending(t, router, fmt.Sprintf("/tenants/%s/configurations/routes", tenantId))
}

func TestGetAllVesselsPaginates(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)
	logger, _ := logtest.NewNullLogger()
	tenantId := uuid.New()
	seedVesselsBlob(t, db, tenantId)

	router := mux.NewRouter()
	configuration.RegisterRoutes(db)(testServerInformation{})(router, logger)

	assertFirstPageOfTwoAscending(t, router, fmt.Sprintf("/tenants/%s/configurations/vessels", tenantId))
}

func TestGetAllInstanceRoutesPaginates(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)
	logger, _ := logtest.NewNullLogger()
	tenantId := uuid.New()
	seedInstanceRoutesBlob(t, db, tenantId)

	router := mux.NewRouter()
	configuration.RegisterRoutes(db)(testServerInformation{})(router, logger)

	assertFirstPageOfTwoAscending(t, router, fmt.Sprintf("/tenants/%s/configurations/instance-routes", tenantId))
}
