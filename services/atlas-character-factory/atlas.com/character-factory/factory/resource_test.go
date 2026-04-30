package factory

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
)

// stubServerInfo satisfies jsonapi.ServerInformation for testing.
type stubServerInfo struct{}

func (s stubServerInfo) GetBaseURL() string    { return "http://localhost" }
func (s stubServerInfo) GetPrefix() string     { return "" }

func newTestDeps(t *testing.T) (*server.HandlerDependency, *server.HandlerContext) {
	t.Helper()
	ctx, _ := createMockContext(t)
	l := logrus.New()
	l.SetLevel(logrus.DebugLevel)
	dep := server.NewHandlerDependency(l, ctx)
	hc := server.NewHandlerContext(jsonapi.ServerInformation(stubServerInfo{}))
	return &dep, &hc
}

func TestHandleCreateFromPreset_BadJSON(t *testing.T) {
	d, c := newTestDeps(t)
	handler := handleCreateFromPreset(d, c)

	req := httptest.NewRequest(http.MethodPost, "/characters/from-preset", strings.NewReader("{invalid json"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}
}

func TestHandleCreateFromPreset_MissingPresetId(t *testing.T) {
	d, c := newTestDeps(t)
	handler := handleCreateFromPreset(d, c)

	body := `{"accountId":1,"worldId":0,"name":"TestChar"}`
	req := httptest.NewRequest(http.MethodPost, "/characters/from-preset", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}
}

func TestHandleCreateFromPreset_MissingName(t *testing.T) {
	d, c := newTestDeps(t)
	handler := handleCreateFromPreset(d, c)

	body := `{"presetId":"550e8400-e29b-41d4-a716-446655440000","accountId":1,"worldId":0}`
	req := httptest.NewRequest(http.MethodPost, "/characters/from-preset", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}
}

func TestHandleNameValidity_MissingWorldId(t *testing.T) {
	d, c := newTestDeps(t)
	handler := handleNameValidity(d, c)

	req := httptest.NewRequest(http.MethodGet, "/characters/name-validity?name=TestChar", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}
}

func TestHandleNameValidity_MissingName(t *testing.T) {
	d, c := newTestDeps(t)
	handler := handleNameValidity(d, c)

	req := httptest.NewRequest(http.MethodGet, "/characters/name-validity?worldId=0", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}
}

func TestHandleNameValidity_InvalidWorldId(t *testing.T) {
	d, c := newTestDeps(t)
	handler := handleNameValidity(d, c)

	req := httptest.NewRequest(http.MethodGet, "/characters/name-validity?name=TestChar&worldId=notanumber", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}
}

func TestCategorizePresetError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"nil error", nil, http.StatusOK},
		{"ErrInvalidPresetId", ErrInvalidPresetId, http.StatusBadRequest},
		{"ErrPresetNotFound", ErrPresetNotFound, http.StatusNotFound},
		{"ErrNameDuplicate", ErrNameDuplicate, http.StatusConflict},
		{"ErrAtlasDataUnreachable", ErrAtlasDataUnreachable, http.StatusBadGateway},
		{"ErrPresetValidation", ErrPresetValidation, http.StatusBadRequest},
		{"NameInvalidError", &NameInvalidError{Reason: "blocked", Detail: "name is blocked"}, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				// categorizePresetError is not called with nil in production; skip
				return
			}
			got := categorizePresetError(tt.err)
			if got != tt.wantStatus {
				t.Errorf("categorizePresetError(%v) = %d, want %d", tt.err, got, tt.wantStatus)
			}
		})
	}
}
