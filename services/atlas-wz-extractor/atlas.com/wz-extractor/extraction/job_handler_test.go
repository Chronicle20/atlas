package extraction

import (
	"atlas-wz-extractor/extraction/job"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gorilla/mux"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus/hooks/test"
)

func newRedisJ(t *testing.T) *goredis.Client {
	t.Helper()
	mr := miniredis.RunT(t)
	return goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
}

func TestJobHandler_404Unknown(t *testing.T) {
	c := newRedisJ(t)
	store := job.NewStore(c)
	router := mux.NewRouter()
	l, _ := test.NewNullLogger()
	dirs := Dirs{InputDir: t.TempDir(), OutputXmlDir: t.TempDir(), OutputImgDir: t.TempDir()}
	initFn := InitResource(NewProcessor(dirs.InputDir, dirs.OutputXmlDir, dirs.OutputImgDir), store, nil, nil, &sync.WaitGroup{}, dirs)
	initFn(serverInfo{})(router, l)

	req := httptest.NewRequest(http.MethodGet, "/wz/extractions/jobs/does-not-exist", nil)
	req.Header.Set("TENANT_ID", "00000000-0000-0000-0000-000000000001")
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status: %d", w.Code)
	}
}

func TestJobHandler_200Returns_wzExtractionJob(t *testing.T) {
	c := newRedisJ(t)
	store := job.NewStore(c)
	now := time.Now().UTC().Truncate(time.Second)
	j := job.NewJobBuilder().SetId("J").SetTenantId("T").SetRegion("GMS").
		SetMajorVersion(83).SetMinorVersion(1).
		SetStatus(job.JobRunning).SetUnitsTotal(2).SetUnitsCompleted(1).
		SetCreatedAt(now).SetUpdatedAt(now).Build()
	if err := store.Create(context.Background(), j, []job.Unit{
		job.NewUnitBuilder().SetWzFile("Map.wz").SetStatus(job.UnitSucceeded).Build(),
		job.NewUnitBuilder().SetWzFile("Mob.wz").SetStatus(job.UnitRunning).Build(),
	}, 3600); err != nil {
		t.Fatal(err)
	}

	router := mux.NewRouter()
	l, _ := test.NewNullLogger()
	dirs := Dirs{InputDir: t.TempDir(), OutputXmlDir: t.TempDir(), OutputImgDir: t.TempDir()}
	initFn := InitResource(NewProcessor(dirs.InputDir, dirs.OutputXmlDir, dirs.OutputImgDir), store, nil, nil, &sync.WaitGroup{}, dirs)
	initFn(serverInfo{})(router, l)

	req := httptest.NewRequest(http.MethodGet, "/wz/extractions/jobs/J", nil)
	req.Header.Set("TENANT_ID", "00000000-0000-0000-0000-000000000001")
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: %d body=%s", w.Code, w.Body.String())
	}
	// api2go serialises as {"data":{"type":"...","id":"...","attributes":{...}}}
	var env map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&env); err != nil {
		t.Fatal(err)
	}
	data, ok := env["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing data object in envelope: %v", env)
	}
	if data["type"] != "wzExtractionJob" {
		t.Fatalf("wrong type: %v", data["type"])
	}
	if data["id"] != "J" {
		t.Fatalf("wrong id: %v", data["id"])
	}
	attrs, ok := data["attributes"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing attributes in data: %v", data)
	}
	// unitsTotal is decoded as float64 by json.Unmarshal into interface{}
	if int(attrs["unitsTotal"].(float64)) != 2 {
		t.Fatalf("unitsTotal: %v", attrs["unitsTotal"])
	}
	units, ok := attrs["units"].([]interface{})
	if !ok || len(units) != 2 {
		t.Fatalf("units: %v", attrs["units"])
	}
}
