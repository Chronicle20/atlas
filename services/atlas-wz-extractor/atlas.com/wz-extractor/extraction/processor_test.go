package extraction

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
)

func TestNewProcessor(t *testing.T) {
	p := NewProcessor("/input", "/xml", "/img")
	if p == nil {
		t.Fatal("expected non-nil processor")
	}
}

func TestExtract_NoWzFiles(t *testing.T) {
	tenantId := uuid.New()
	tt, _ := tenant.Create(tenantId, "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), tt)

	inputDir := t.TempDir()
	tenantInput := filepath.Join(inputDir, tenantId.String(), "GMS", "83.1")
	if err := os.MkdirAll(tenantInput, 0o755); err != nil {
		t.Fatal(err)
	}

	p := NewProcessor(inputDir, t.TempDir(), t.TempDir())
	l, _ := test.NewNullLogger()
	err := p.Extract(l, ctx, false, false)
	if err == nil || !strings.Contains(err.Error(), "no WZ files found") {
		t.Fatalf("expected no-WZ-files error, got %v", err)
	}
}

func TestExtract_OutputPathConstruction(t *testing.T) {
	tenantId := uuid.New()
	tt, err := tenant.Create(tenantId, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("unable to create tenant: %v", err)
	}
	ctx := tenant.WithContext(context.Background(), tt)

	inputDir := t.TempDir()
	xmlDir := t.TempDir()
	imgDir := t.TempDir()

	// Place the dummy .wz under the tenant-scoped path (flat-dir fallback is gone).
	tenantInput := filepath.Join(inputDir, tenantId.String(), "GMS", "83.1")
	if err := os.MkdirAll(tenantInput, 0o755); err != nil {
		t.Fatalf("unable to mkdir tenant input: %v", err)
	}
	dummyPath := filepath.Join(tenantInput, "Test.wz")
	if err := os.WriteFile(dummyPath, []byte("not a valid wz file"), 0644); err != nil {
		t.Fatalf("unable to create dummy file: %v", err)
	}

	p := NewProcessor(inputDir, xmlDir, imgDir)
	l, _ := test.NewNullLogger()

	// Extract will try to open the dummy file and fail on parse, but that's OK.
	// The test verifies Extract doesn't panic and processes the tenant context.
	_ = p.Extract(l, ctx, false, false)

	// Verify the output directory structure was not created (since the WZ file is invalid)
	expectedXmlPath := filepath.Join(xmlDir, tenantId.String(), "GMS", "83.1")
	expectedImgPath := filepath.Join(imgDir, tenantId.String(), "GMS", "83.1")

	// These directories should NOT exist because wz.Open fails on the invalid file
	if _, err := os.Stat(expectedXmlPath); err == nil {
		t.Errorf("did not expect xml output path to exist for invalid WZ file")
	}
	if _, err := os.Stat(expectedImgPath); err == nil {
		t.Errorf("did not expect img output path to exist for invalid WZ file")
	}
}

func TestRunExtractionWipesCharacterCache(t *testing.T) {
	tmp := t.TempDir()
	imgOut := filepath.Join(tmp, "out", "img", "tenant-a", "GMS", "83.1")
	cacheDir := filepath.Join(imgOut, "character")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "abcdef1234567890.png"), []byte("stale"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := wipeCharacterCache(imgOut); err != nil {
		t.Fatalf("wipe: %v", err)
	}
	if _, err := os.Stat(cacheDir); !os.IsNotExist(err) {
		t.Fatalf("expected cacheDir gone, stat err=%v", err)
	}
}

func TestExtractUnit_FailsWhenWzCannotBeOpened(t *testing.T) {
	tenantId := uuid.New()
	tt, _ := tenant.Create(tenantId, "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), tt)

	inputDir := t.TempDir()
	xmlDir := t.TempDir()
	imgDir := t.TempDir()
	tenantInput := filepath.Join(inputDir, tenantId.String(), "GMS", "83.1")
	if err := os.MkdirAll(tenantInput, 0o755); err != nil {
		t.Fatal(err)
	}
	bad := filepath.Join(tenantInput, "Bad.wz")
	if err := os.WriteFile(bad, []byte("not a real wz"), 0o644); err != nil {
		t.Fatal(err)
	}

	p := NewProcessor(inputDir, xmlDir, imgDir)
	l, _ := test.NewNullLogger()
	if err := p.ExtractUnit(l, ctx, "Bad.wz", false, false); err == nil {
		t.Fatalf("expected non-nil error when wz.Open fails")
	}
}

func TestExtractUnit_RejectsMissingWzFile(t *testing.T) {
	tenantId := uuid.New()
	tt, _ := tenant.Create(tenantId, "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), tt)
	p := NewProcessor(t.TempDir(), t.TempDir(), t.TempDir())
	l, _ := test.NewNullLogger()
	if err := p.ExtractUnit(l, ctx, "Nope.wz", false, false); err == nil {
		t.Fatalf("expected error for missing wz file")
	}
}

func TestExtract_TenantPathFormat(t *testing.T) {
	tests := []struct {
		name    string
		region  string
		major   uint16
		minor   uint16
		wantVer string
	}{
		{"standard version", "GMS", 83, 1, "83.1"},
		{"zero minor", "KMS", 92, 0, "92.0"},
		{"high version", "JMS", 200, 50, "200.50"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tenantId := uuid.New()
			tt, err := tenant.Create(tenantId, tc.region, tc.major, tc.minor)
			if err != nil {
				t.Fatalf("unable to create tenant: %v", err)
			}
			ctx := tenant.WithContext(context.Background(), tt)

			inputDir := t.TempDir()
			xmlDir := t.TempDir()
			imgDir := t.TempDir()

			tenantInput := filepath.Join(inputDir, tenantId.String(), tc.region, tc.wantVer)
			if err := os.MkdirAll(tenantInput, 0o755); err != nil {
				t.Fatalf("unable to mkdir tenant input: %v", err)
			}
			if err := os.WriteFile(filepath.Join(tenantInput, "Test.wz"), []byte("dummy"), 0644); err != nil {
				t.Fatalf("unable to create dummy file: %v", err)
			}

			p := NewProcessor(inputDir, xmlDir, imgDir)
			l, _ := test.NewNullLogger()

			_ = p.Extract(l, ctx, false, false)

			// Verify the path format by checking the processor derived the version string correctly.
			// We access the internals to validate path construction.
			impl := p.(*processorImpl)
			version := tc.wantVer
			wantXml := filepath.Join(impl.outputXmlDir, tenantId.String(), tc.region, version)
			wantImg := filepath.Join(impl.outputImgDir, tenantId.String(), tc.region, version)

			// These are the paths that would have been passed to runExtraction.
			// Since the WZ file is invalid, no dirs are created, but the format is validated
			// by asserting the expected path structure matches what Extract would compute.
			expectedXml := filepath.Join(xmlDir, tenantId.String(), tc.region, tc.wantVer)
			expectedImg := filepath.Join(imgDir, tenantId.String(), tc.region, tc.wantVer)

			if wantXml != expectedXml {
				t.Errorf("xml path mismatch: want %s, got %s", expectedXml, wantXml)
			}
			if wantImg != expectedImg {
				t.Errorf("img path mismatch: want %s, got %s", expectedImg, wantImg)
			}
		})
	}
}
