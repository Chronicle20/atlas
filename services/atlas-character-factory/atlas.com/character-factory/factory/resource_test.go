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

func (s stubServerInfo) GetBaseURL() string { return "http://localhost" }
func (s stubServerInfo) GetPrefix() string  { return "" }

func newTestDeps(t *testing.T) (*server.HandlerDependency, *server.HandlerContext) {
	t.Helper()
	ctx, _ := createMockContext(t)
	l := logrus.New()
	l.SetLevel(logrus.DebugLevel)
	dep := server.NewHandlerDependency(l, ctx)
	hc := server.NewHandlerContext(jsonapi.ServerInformation(stubServerInfo{}))
	return &dep, &hc
}

// postPreset issues a POST to the handleCreateFromPreset input handler, going through
// ParseInput so JSON:API unmarshalling happens exactly as in production.
func postPreset(t *testing.T, body string) *httptest.ResponseRecorder {
	t.Helper()
	d, c := newTestDeps(t)
	req := httptest.NewRequest(http.MethodPost, "/characters/from-preset", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/vnd.api+json")
	rr := httptest.NewRecorder()
	server.ParseInput[PresetCreateRestModel](d, c, handleCreateFromPreset)(rr, req)
	return rr
}

// TestHandleCreateFromPreset_BadJSON verifies that malformed JSON is rejected with 400.
func TestHandleCreateFromPreset_BadJSON(t *testing.T) {
	rr := postPreset(t, "{invalid json")
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}
}

// TestHandleCreateFromPreset_MissingPresetId verifies that a missing presetId results in 400.
// With JSON:API decoding, an empty presetId in attributes maps to the zero value "".
// The processor's uuid.Parse("") call returns ErrInvalidPresetId → 400.
func TestHandleCreateFromPreset_MissingPresetId(t *testing.T) {
	// Valid JSON:API body with no presetId attribute (omitted → empty string)
	body := `{"data":{"type":"preset-create","attributes":{"accountId":1,"worldId":0,"name":"TestChar"}}}`
	rr := postPreset(t, body)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}
}

// TestHandleCreateFromPreset_InvalidPresetIdFormat verifies that a malformed presetId
// (not a valid UUID) is rejected with 400 via ErrInvalidPresetId, without reaching the
// tenant registry or any downstream service.
func TestHandleCreateFromPreset_InvalidPresetIdFormat(t *testing.T) {
	body := `{"data":{"type":"preset-create","attributes":{"presetId":"not-a-valid-uuid","accountId":1,"worldId":0,"name":"TestChar"}}}`
	rr := postPreset(t, body)
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
