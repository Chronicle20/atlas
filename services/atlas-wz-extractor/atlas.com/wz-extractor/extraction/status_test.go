package extraction

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/google/uuid"
)

type statusEnvelopeJSON struct {
	Data struct {
		Type       string `json:"type"`
		Id         string `json:"id"`
		Attributes struct {
			FileCount  int     `json:"fileCount"`
			TotalBytes int64   `json:"totalBytes"`
			UpdatedAt  *string `json:"updatedAt"`
		} `json:"attributes"`
	} `json:"data"`
}

func statusRequest(method, path string, tenantId uuid.UUID) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("TENANT_ID", tenantId.String())
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")
	return req
}

func TestInputStatus_EmptyDir(t *testing.T) {
	inputDir := t.TempDir()
	mock := newMockProcessor()
	wg := &sync.WaitGroup{}
	router := setupRouterWithDirs(mock, wg, Dirs{InputDir: inputDir})

	tenantId := uuid.New()
	w := httptest.NewRecorder()
	router.ServeHTTP(w, statusRequest(http.MethodGet, "/wz/input", tenantId))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var env statusEnvelopeJSON
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if env.Data.Type != "wzInputStatus" {
		t.Errorf("type = %q, want wzInputStatus", env.Data.Type)
	}
	if env.Data.Attributes.FileCount != 0 {
		t.Errorf("fileCount = %d, want 0", env.Data.Attributes.FileCount)
	}
	if env.Data.Attributes.TotalBytes != 0 {
		t.Errorf("totalBytes = %d, want 0", env.Data.Attributes.TotalBytes)
	}
	if env.Data.Attributes.UpdatedAt != nil {
		t.Errorf("updatedAt should be null for empty dir, got %v", *env.Data.Attributes.UpdatedAt)
	}
}

func TestInputStatus_Populated(t *testing.T) {
	inputDir := t.TempDir()
	mock := newMockProcessor()
	wg := &sync.WaitGroup{}
	router := setupRouterWithDirs(mock, wg, Dirs{InputDir: inputDir})

	tenantId := uuid.New()
	dst := filepath.Join(inputDir, tenantPathFor(tenantId))
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dst, "Map.wz"), []byte("abc"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dst, "String.wz"), []byte("abcdefg"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	// Should NOT be counted: non-wz at top level
	if err := os.WriteFile(filepath.Join(dst, "notes.txt"), []byte("ignore"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, statusRequest(http.MethodGet, "/wz/input", tenantId))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var env statusEnvelopeJSON
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if env.Data.Attributes.FileCount != 2 {
		t.Errorf("fileCount = %d, want 2", env.Data.Attributes.FileCount)
	}
	if env.Data.Attributes.TotalBytes != 10 {
		t.Errorf("totalBytes = %d, want 10", env.Data.Attributes.TotalBytes)
	}
	if env.Data.Attributes.UpdatedAt == nil {
		t.Errorf("updatedAt should be set for populated dir")
	}
}

func TestExtractionStatus_RecursiveXml(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := t.TempDir()
	mock := newMockProcessor()
	wg := &sync.WaitGroup{}
	router := setupRouterWithDirs(mock, wg, Dirs{InputDir: inputDir, OutputXmlDir: outputDir})

	tenantId := uuid.New()
	dst := filepath.Join(outputDir, tenantPathFor(tenantId))
	nested := filepath.Join(dst, "String.wz")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nested, "Map.img.xml"), []byte("xyz"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dst, "top.xml"), []byte("top"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dst, "ignore.bin"), []byte("nope"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, statusRequest(http.MethodGet, "/wz/extractions", tenantId))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var env statusEnvelopeJSON
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if env.Data.Type != "wzExtractionStatus" {
		t.Errorf("type = %q, want wzExtractionStatus", env.Data.Type)
	}
	if env.Data.Attributes.FileCount != 2 {
		t.Errorf("fileCount = %d, want 2 (top.xml + Map.img.xml)", env.Data.Attributes.FileCount)
	}
	if env.Data.Attributes.TotalBytes != 6 {
		t.Errorf("totalBytes = %d, want 6", env.Data.Attributes.TotalBytes)
	}
}

func TestExtractionStatus_EmptyDir(t *testing.T) {
	outputDir := t.TempDir()
	mock := newMockProcessor()
	wg := &sync.WaitGroup{}
	router := setupRouterWithDirs(mock, wg, Dirs{OutputXmlDir: outputDir})

	tenantId := uuid.New()
	w := httptest.NewRecorder()
	router.ServeHTTP(w, statusRequest(http.MethodGet, "/wz/extractions", tenantId))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var env statusEnvelopeJSON
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if env.Data.Attributes.FileCount != 0 {
		t.Errorf("fileCount = %d, want 0", env.Data.Attributes.FileCount)
	}
	if env.Data.Attributes.UpdatedAt != nil {
		t.Errorf("updatedAt should be null for empty output dir")
	}
}
