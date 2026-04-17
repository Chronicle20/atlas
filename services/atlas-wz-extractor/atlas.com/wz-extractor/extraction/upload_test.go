package extraction

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/google/uuid"
)

type zipEntry struct {
	name string
	data []byte
}

func buildZip(t *testing.T, entries []zipEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, e := range entries {
		w, err := zw.Create(e.name)
		if err != nil {
			t.Fatalf("zip create: %v", err)
		}
		if _, err := w.Write(e.data); err != nil {
			t.Fatalf("zip write: %v", err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}
	return buf.Bytes()
}

func buildMultipart(t *testing.T, partName, fileName string, body []byte) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, err := mw.CreateFormFile(partName, fileName)
	if err != nil {
		t.Fatalf("multipart form: %v", err)
	}
	if _, err := io.Copy(part, bytes.NewReader(body)); err != nil {
		t.Fatalf("copy: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("mw close: %v", err)
	}
	return &buf, mw.FormDataContentType()
}

func uploadRequest(t *testing.T, zipBytes []byte, tenantId uuid.UUID) *http.Request {
	t.Helper()
	body, ct := buildMultipart(t, "zip_file", "fixture.zip", zipBytes)
	req := httptest.NewRequest(http.MethodPatch, "/wz/input", body)
	req.Header.Set("Content-Type", ct)
	req.Header.Set("TENANT_ID", tenantId.String())
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")
	return req
}

func tenantPathFor(tenantId uuid.UUID) string {
	return filepath.Join(tenantId.String(), "GMS", "83.1")
}

func TestUpload_Flat202(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := t.TempDir()
	mock := newMockProcessor()
	wg := &sync.WaitGroup{}
	router := setupRouterWithDirs(mock, wg, Dirs{InputDir: inputDir, OutputXmlDir: outputDir})

	tenantId := uuid.New()
	zipBytes := buildZip(t, []zipEntry{
		{"Map.wz", []byte("map-bytes")},
		{"String.wz", []byte("string-bytes-longer")},
	})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, uploadRequest(t, zipBytes, tenantId))

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}

	dst := filepath.Join(inputDir, tenantPathFor(tenantId))
	for _, name := range []string{"Map.wz", "String.wz"} {
		p := filepath.Join(dst, name)
		info, err := os.Stat(p)
		if err != nil {
			t.Fatalf("expected file %s: %v", p, err)
		}
		if info.Size() == 0 {
			t.Errorf("expected non-empty file %s", p)
		}
	}
}

func TestUpload_NestedPath400(t *testing.T) {
	inputDir := t.TempDir()
	mock := newMockProcessor()
	wg := &sync.WaitGroup{}
	router := setupRouterWithDirs(mock, wg, Dirs{InputDir: inputDir})

	tenantId := uuid.New()
	// pre-populate destination with a sentinel file to verify it is NOT wiped on 400
	dst := filepath.Join(inputDir, tenantPathFor(tenantId))
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	sentinel := filepath.Join(dst, "pre-existing.wz")
	if err := os.WriteFile(sentinel, []byte("keep me"), 0644); err != nil {
		t.Fatalf("sentinel: %v", err)
	}

	zipBytes := buildZip(t, []zipEntry{
		{"String.wz/Map.img.xml", []byte("nope")},
	})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, uploadRequest(t, zipBytes, tenantId))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	if _, err := os.Stat(sentinel); err != nil {
		t.Errorf("destination was modified on 400: sentinel missing: %v", err)
	}
}

func TestUpload_DotDot400(t *testing.T) {
	inputDir := t.TempDir()
	mock := newMockProcessor()
	wg := &sync.WaitGroup{}
	router := setupRouterWithDirs(mock, wg, Dirs{InputDir: inputDir})

	tenantId := uuid.New()
	zipBytes := buildZip(t, []zipEntry{
		{"..evil.wz", []byte("no")},
	})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, uploadRequest(t, zipBytes, tenantId))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpload_AbsolutePath400(t *testing.T) {
	inputDir := t.TempDir()
	mock := newMockProcessor()
	wg := &sync.WaitGroup{}
	router := setupRouterWithDirs(mock, wg, Dirs{InputDir: inputDir})

	tenantId := uuid.New()
	// absolute path triggers the path-separator check first
	zipBytes := buildZip(t, []zipEntry{
		{"/etc/passwd.wz", []byte("no")},
	})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, uploadRequest(t, zipBytes, tenantId))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpload_NonWzEntry400(t *testing.T) {
	inputDir := t.TempDir()
	mock := newMockProcessor()
	wg := &sync.WaitGroup{}
	router := setupRouterWithDirs(mock, wg, Dirs{InputDir: inputDir})

	tenantId := uuid.New()
	zipBytes := buildZip(t, []zipEntry{
		{"Map.wz", []byte("ok")},
		{"readme.txt", []byte("nope")},
	})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, uploadRequest(t, zipBytes, tenantId))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err == nil {
		if body["error"] == "" {
			t.Errorf("expected error field in body, got %v", body)
		}
	}
}

func TestUpload_MutexBusy409(t *testing.T) {
	inputDir := t.TempDir()
	mock := newMockProcessor()
	wg := &sync.WaitGroup{}
	router := setupRouterWithDirs(mock, wg, Dirs{InputDir: inputDir})

	tenantId := uuid.New()

	// hand-craft a tenant to derive the same key used by handleUpload
	key := tenantId.String() + ":GMS:83.1"
	held := Acquire(key)
	defer Release(held)

	zipBytes := buildZip(t, []zipEntry{
		{"Map.wz", []byte("ok")},
	})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, uploadRequest(t, zipBytes, tenantId))

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpload_ReuploadReplaces(t *testing.T) {
	inputDir := t.TempDir()
	mock := newMockProcessor()
	wg := &sync.WaitGroup{}
	router := setupRouterWithDirs(mock, wg, Dirs{InputDir: inputDir})

	tenantId := uuid.New()

	first := buildZip(t, []zipEntry{
		{"Old.wz", []byte("old")},
	})
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, uploadRequest(t, first, tenantId))
	if w1.Code != http.StatusAccepted {
		t.Fatalf("first upload failed: %d %s", w1.Code, w1.Body.String())
	}

	second := buildZip(t, []zipEntry{
		{"New.wz", []byte("new")},
	})
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, uploadRequest(t, second, tenantId))
	if w2.Code != http.StatusAccepted {
		t.Fatalf("second upload failed: %d %s", w2.Code, w2.Body.String())
	}

	dst := filepath.Join(inputDir, tenantPathFor(tenantId))
	if _, err := os.Stat(filepath.Join(dst, "Old.wz")); !os.IsNotExist(err) {
		t.Errorf("expected Old.wz to be removed after re-upload")
	}
	if _, err := os.Stat(filepath.Join(dst, "New.wz")); err != nil {
		t.Errorf("expected New.wz to exist after re-upload: %v", err)
	}
}
