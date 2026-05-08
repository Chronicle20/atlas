package extraction

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

type serverInfo struct{}

func (s serverInfo) GetBaseURL() string { return "" }
func (s serverInfo) GetPrefix() string  { return "/api/" }

// mockProcessor is a no-op Processor used by status_test.go and upload_test.go.
// It satisfies the Processor interface without doing any real work.
type mockProcessor struct{}

func newMockProcessor() *mockProcessor { return &mockProcessor{} }

func (m *mockProcessor) Extract(l logrus.FieldLogger, ctx context.Context, xmlOnly, imagesOnly bool) error {
	return nil
}

func (m *mockProcessor) ExtractUnit(l logrus.FieldLogger, ctx context.Context, wzFile string, xmlOnly, imagesOnly bool) error {
	return nil
}

// setupRouter wires the resource with nil redis-backed deps. Used by tests
// that exercise routing and the missing-tenant-header path; full dispatcher
// behaviour lives in dispatcher_test.go and job_handler_test.go.
func setupRouter(p Processor, wg *sync.WaitGroup) *mux.Router {
	return setupRouterWithDirs(p, wg, Dirs{})
}

func setupRouterWithDirs(p Processor, wg *sync.WaitGroup, dirs Dirs) *mux.Router {
	router := mux.NewRouter()
	l, _ := test.NewNullLogger()
	initFn := InitResource(p, nil, nil, nil, wg, dirs)
	routeInit := initFn(serverInfo{})
	routeInit(router, l)
	return router
}

func TestHandleExtract_MissingTenantHeader_Returns400(t *testing.T) {
	wg := &sync.WaitGroup{}
	router := setupRouter(NewProcessor(t.TempDir(), t.TempDir(), t.TempDir()), wg)

	req := httptest.NewRequest(http.MethodPost, "/wz/extractions", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}
