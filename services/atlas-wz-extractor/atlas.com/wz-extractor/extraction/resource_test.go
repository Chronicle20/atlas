package extraction

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

type serverInfo struct{}

func (s serverInfo) GetBaseURL() string { return "" }
func (s serverInfo) GetPrefix() string  { return "/api/" }

type mockProcessor struct {
	extractCalled bool
	extractErr    error
	lastXmlOnly   bool
	lastImgOnly   bool
	mu            sync.Mutex
	done          chan struct{}
}

func newMockProcessor() *mockProcessor {
	return &mockProcessor{done: make(chan struct{})}
}

func (m *mockProcessor) Extract(l logrus.FieldLogger, ctx context.Context, xmlOnly, imagesOnly bool) error {
	m.mu.Lock()
	m.extractCalled = true
	m.lastXmlOnly = xmlOnly
	m.lastImgOnly = imagesOnly
	m.mu.Unlock()
	close(m.done)
	return m.extractErr
}

func (m *mockProcessor) waitForExtract(t *testing.T) {
	t.Helper()
	select {
	case <-m.done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for Extract call")
	}
}

func setupRouter(p Processor, wg *sync.WaitGroup) *mux.Router {
	router := mux.NewRouter()
	l, _ := test.NewNullLogger()
	initFn := InitResource(p, wg)
	routeInit := initFn(serverInfo{})
	routeInit(router, l)
	return router
}

func newTenantRequest(method, path string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("TENANT_ID", uuid.New().String())
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")
	return req
}

func TestHandleExtract_Returns202(t *testing.T) {
	mock := newMockProcessor()
	wg := &sync.WaitGroup{}
	router := setupRouter(mock, wg)

	req := newTenantRequest(http.MethodPost, "/wz/extractions")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected status %d, got %d", http.StatusAccepted, w.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("unable to decode response body: %v", err)
	}
	if body["status"] != "started" {
		t.Errorf("expected status 'started', got %q", body["status"])
	}

	mock.waitForExtract(t)
	wg.Wait()
}

func TestHandleExtract_CallsProcessor(t *testing.T) {
	mock := newMockProcessor()
	wg := &sync.WaitGroup{}
	router := setupRouter(mock, wg)

	req := newTenantRequest(http.MethodPost, "/wz/extractions")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	mock.waitForExtract(t)
	wg.Wait()

	mock.mu.Lock()
	defer mock.mu.Unlock()
	if !mock.extractCalled {
		t.Error("expected Extract to be called")
	}
}

func TestHandleExtract_XmlOnly(t *testing.T) {
	mock := newMockProcessor()
	wg := &sync.WaitGroup{}
	router := setupRouter(mock, wg)

	req := newTenantRequest(http.MethodPost, "/wz/extractions?xmlOnly=true")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	mock.waitForExtract(t)
	wg.Wait()

	mock.mu.Lock()
	defer mock.mu.Unlock()
	if !mock.lastXmlOnly {
		t.Error("expected xmlOnly=true")
	}
	if mock.lastImgOnly {
		t.Error("expected imagesOnly=false")
	}
}

func TestHandleExtract_ImagesOnly(t *testing.T) {
	mock := newMockProcessor()
	wg := &sync.WaitGroup{}
	router := setupRouter(mock, wg)

	req := newTenantRequest(http.MethodPost, "/wz/extractions?imagesOnly=true")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	mock.waitForExtract(t)
	wg.Wait()

	mock.mu.Lock()
	defer mock.mu.Unlock()
	if mock.lastXmlOnly {
		t.Error("expected xmlOnly=false")
	}
	if !mock.lastImgOnly {
		t.Error("expected imagesOnly=true")
	}
}

func TestHandleExtract_DefaultBothModes(t *testing.T) {
	mock := newMockProcessor()
	wg := &sync.WaitGroup{}
	router := setupRouter(mock, wg)

	req := newTenantRequest(http.MethodPost, "/wz/extractions")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	mock.waitForExtract(t)
	wg.Wait()

	mock.mu.Lock()
	defer mock.mu.Unlock()
	if mock.lastXmlOnly {
		t.Error("expected xmlOnly=false by default")
	}
	if mock.lastImgOnly {
		t.Error("expected imagesOnly=false by default")
	}
}

func TestHandleExtract_MissingTenantHeader_Returns400(t *testing.T) {
	mock := newMockProcessor()
	wg := &sync.WaitGroup{}
	router := setupRouter(mock, wg)

	req := httptest.NewRequest(http.MethodPost, "/wz/extractions", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleExtract_TracksGoroutineInWaitGroup(t *testing.T) {
	mock := newMockProcessor()
	wg := &sync.WaitGroup{}
	router := setupRouter(mock, wg)

	req := newTenantRequest(http.MethodPost, "/wz/extractions")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	mock.waitForExtract(t)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("WaitGroup did not complete — goroutine not tracked")
	}
}

func TestHandleExtract_PropagatesTenantToProcessor(t *testing.T) {
	tenantId := uuid.New()
	var capturedCtx context.Context

	p := &contextCapturingProcessor{done: make(chan struct{})}
	p.extractFunc = func(l logrus.FieldLogger, ctx context.Context, xmlOnly, imagesOnly bool) error {
		capturedCtx = ctx
		close(p.done)
		return nil
	}

	wg := &sync.WaitGroup{}
	router := setupRouter(p, wg)

	req := httptest.NewRequest(http.MethodPost, "/wz/extractions", nil)
	req.Header.Set("TENANT_ID", tenantId.String())
	req.Header.Set("REGION", "KMS")
	req.Header.Set("MAJOR_VERSION", "92")
	req.Header.Set("MINOR_VERSION", "3")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	select {
	case <-p.done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for Extract call")
	}
	wg.Wait()

	if capturedCtx == nil {
		t.Fatal("expected context to be captured")
	}

	tt := tenant.MustFromContext(capturedCtx)
	if tt.Id() != tenantId {
		t.Errorf("expected tenant ID %s, got %s", tenantId, tt.Id())
	}
	if tt.Region() != "KMS" {
		t.Errorf("expected region KMS, got %s", tt.Region())
	}
	if tt.MajorVersion() != 92 {
		t.Errorf("expected major version 92, got %d", tt.MajorVersion())
	}
	if tt.MinorVersion() != 3 {
		t.Errorf("expected minor version 3, got %d", tt.MinorVersion())
	}
}

type contextCapturingProcessor struct {
	extractFunc func(logrus.FieldLogger, context.Context, bool, bool) error
	done        chan struct{}
}

func (p *contextCapturingProcessor) Extract(l logrus.FieldLogger, ctx context.Context, xmlOnly, imagesOnly bool) error {
	return p.extractFunc(l, ctx, xmlOnly, imagesOnly)
}
