package configuration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func testCtx(t *testing.T) context.Context {
	t.Helper()
	tm, err := tenant.Register(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tenant.WithContext(context.Background(), tm)
}

func TestGetRecomputeIntervalConfigured(t *testing.T) {
	tenantId := uuid.New()
	wantPath := "/tenants/" + tenantId.String() + "/configurations/rankings"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != wantPath {
			t.Errorf("unexpected path %s, want %s", r.URL.Path, wantPath)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"type":"rankings","id":"` + tenantId.String() + `","attributes":{"recomputeIntervalMinutes":15}}}`))
	}))
	defer srv.Close()
	t.Setenv("TENANTS_SERVICE_URL", srv.URL+"/")

	got := GetRecomputeInterval(logrus.New(), testCtx(t))(tenantId)
	if got != 15*time.Minute {
		t.Fatalf("interval = %v, want 15m", got)
	}
}

func TestGetRecomputeIntervalDefaultsOn404(t *testing.T) {
	tenantId := uuid.New()
	wantPath := "/tenants/" + tenantId.String() + "/configurations/rankings"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != wantPath {
			t.Errorf("unexpected path %s, want %s", r.URL.Path, wantPath)
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	t.Setenv("TENANTS_SERVICE_URL", srv.URL+"/")

	got := GetRecomputeInterval(logrus.New(), testCtx(t))(tenantId)
	if got != DefaultRecomputeInterval {
		t.Fatalf("interval = %v, want default %v", got, DefaultRecomputeInterval)
	}
}

func TestGetRecomputeIntervalDefaultsOnZero(t *testing.T) {
	tenantId := uuid.New()
	wantPath := "/tenants/" + tenantId.String() + "/configurations/rankings"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != wantPath {
			t.Errorf("unexpected path %s, want %s", r.URL.Path, wantPath)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"type":"rankings","id":"x","attributes":{"recomputeIntervalMinutes":0}}}`))
	}))
	defer srv.Close()
	t.Setenv("TENANTS_SERVICE_URL", srv.URL+"/")

	got := GetRecomputeInterval(logrus.New(), testCtx(t))(tenantId)
	if got != DefaultRecomputeInterval {
		t.Fatalf("interval = %v, want default %v", got, DefaultRecomputeInterval)
	}
}
