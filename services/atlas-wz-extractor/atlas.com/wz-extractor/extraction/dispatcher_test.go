package extraction

import (
	"atlas-wz-extractor/extraction/job"
	"atlas-wz-extractor/extraction/lock"
	mext "atlas-wz-extractor/kafka/message/extraction"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	goredis "github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus/hooks/test"
)

func newRedisD(t *testing.T) *goredis.Client {
	t.Helper()
	mr := miniredis.RunT(t)
	return goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
}

type fakeEmitter struct {
	mu       sync.Mutex
	messages []mext.Command[mext.StartExtractionUnitBody]
	failOn   int
}

// Emit adapts to producer.MessageProducer signature: func(model.Provider[[]kafka.Message]) error
func (f *fakeEmitter) Emit(token string) producer.MessageProducer {
	return func(mp model.Provider[[]kafka.Message]) error {
		ms, err := mp()
		if err != nil {
			return err
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		for _, m := range ms {
			var c mext.Command[mext.StartExtractionUnitBody]
			_ = json.Unmarshal(m.Value, &c)
			f.messages = append(f.messages, c)
			if f.failOn > 0 && len(f.messages) >= f.failOn {
				return os.ErrClosed // arbitrary non-nil
			}
		}
		return nil
	}
}

func (f *fakeEmitter) provider() producerProvider {
	return func(_ context.Context) func(token string) producer.MessageProducer {
		return func(token string) producer.MessageProducer {
			return f.Emit(token)
		}
	}
}

func setupDispatcher(t *testing.T, fe *fakeEmitter, c *goredis.Client) (*mux.Router, *uuid.UUID, string) {
	t.Helper()
	tenantId := uuid.New()
	inputDir := t.TempDir()
	tenantInput := filepath.Join(inputDir, tenantId.String(), "GMS", "83.1")
	if err := os.MkdirAll(tenantInput, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Map.wz", "Mob.wz"} {
		if err := os.WriteFile(filepath.Join(tenantInput, name), []byte("dummy"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	store := job.NewStore(c)
	tl := lock.NewTenantLock(c, time.Minute)
	dirs := Dirs{InputDir: inputDir, OutputXmlDir: t.TempDir(), OutputImgDir: t.TempDir()}

	router := mux.NewRouter()
	l, _ := test.NewNullLogger()
	initFn := InitResource(NewProcessor(inputDir, dirs.OutputXmlDir, dirs.OutputImgDir), store, tl, fe.provider(), &sync.WaitGroup{}, dirs)
	routeInit := initFn(serverInfo{})
	routeInit(router, l)

	return router, &tenantId, inputDir
}

func TestDispatcher_HappyPath202(t *testing.T) {
	c := newRedisD(t)
	fe := &fakeEmitter{}
	router, tid, _ := setupDispatcher(t, fe, c)

	req := httptest.NewRequest(http.MethodPost, "/wz/extractions", nil)
	req.Header.Set("TENANT_ID", tid.String())
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("status: %d body=%s", w.Code, w.Body.String())
	}
	var body struct {
		JobId      string `json:"jobId"`
		UnitsTotal int    `json:"unitsTotal"`
		Status     string `json:"status"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.JobId == "" || body.UnitsTotal != 2 || body.Status != "running" {
		t.Fatalf("body: %+v", body)
	}

	fe.mu.Lock()
	defer fe.mu.Unlock()
	if len(fe.messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(fe.messages))
	}
}

func TestDispatcher_EmptyInput400(t *testing.T) {
	c := newRedisD(t)
	fe := &fakeEmitter{}
	tenantId := uuid.New()
	inputDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(inputDir, tenantId.String(), "GMS", "83.1"), 0o755); err != nil {
		t.Fatal(err)
	}
	store := job.NewStore(c)
	tl := lock.NewTenantLock(c, time.Minute)

	router := mux.NewRouter()
	l, _ := test.NewNullLogger()
	initFn := InitResource(NewProcessor(inputDir, t.TempDir(), t.TempDir()), store, tl, fe.provider(), &sync.WaitGroup{}, Dirs{InputDir: inputDir, OutputXmlDir: t.TempDir(), OutputImgDir: t.TempDir()})
	initFn(serverInfo{})(router, l)

	req := httptest.NewRequest(http.MethodPost, "/wz/extractions", nil)
	req.Header.Set("TENANT_ID", tenantId.String())
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: %d body=%s", w.Code, w.Body.String())
	}
}

func TestDispatcher_LockConflict409(t *testing.T) {
	c := newRedisD(t)
	fe := &fakeEmitter{}
	router, tid, _ := setupDispatcher(t, fe, c)

	req := func() *http.Request {
		r := httptest.NewRequest(http.MethodPost, "/wz/extractions", nil)
		r.Header.Set("TENANT_ID", tid.String())
		r.Header.Set("REGION", "GMS")
		r.Header.Set("MAJOR_VERSION", "83")
		r.Header.Set("MINOR_VERSION", "1")
		return r
	}

	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req())
	if w1.Code != http.StatusAccepted {
		t.Fatalf("first call: %d body=%s", w1.Code, w1.Body.String())
	}

	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req())
	if w2.Code != http.StatusConflict {
		t.Fatalf("second call: %d body=%s", w2.Code, w2.Body.String())
	}
}

func TestDispatcher_PassesRequestContextToProvider(t *testing.T) {
	c := newRedisD(t)
	fe := &fakeEmitter{}

	// Capture the ctx the provider was invoked with.
	var capturedCtx context.Context
	captureProvider := func(ctx context.Context) func(string) producer.MessageProducer {
		capturedCtx = ctx
		return func(token string) producer.MessageProducer {
			return fe.Emit(token)
		}
	}

	tenantId := uuid.New()
	inputDir := t.TempDir()
	tenantInput := filepath.Join(inputDir, tenantId.String(), "GMS", "83.1")
	if err := os.MkdirAll(tenantInput, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tenantInput, "Map.wz"), []byte("dummy"), 0o644); err != nil {
		t.Fatal(err)
	}

	store := job.NewStore(c)
	tl := lock.NewTenantLock(c, time.Minute)
	dirs := Dirs{InputDir: inputDir, OutputXmlDir: t.TempDir(), OutputImgDir: t.TempDir()}

	router := mux.NewRouter()
	l, _ := test.NewNullLogger()
	initFn := InitResource(NewProcessor(inputDir, dirs.OutputXmlDir, dirs.OutputImgDir), store, tl, captureProvider, &sync.WaitGroup{}, dirs)
	initFn(serverInfo{})(router, l)

	req := httptest.NewRequest(http.MethodPost, "/wz/extractions", nil)
	req.Header.Set("TENANT_ID", tenantId.String())
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("status: %d body=%s", w.Code, w.Body.String())
	}
	if capturedCtx == nil {
		t.Fatal("provider was never invoked with a context")
	}
	tt, err := tenant.FromContext(capturedCtx)()
	if err != nil {
		t.Fatalf("expected tenant in producer ctx, got error: %v", err)
	}
	if tt.Id() != tenantId {
		t.Fatalf("expected tenant %s, got %s", tenantId, tt.Id())
	}
	if tt.Region() != "GMS" || tt.MajorVersion() != 83 || tt.MinorVersion() != 1 {
		t.Fatalf("tenant fields wrong: %+v", tt)
	}
}
