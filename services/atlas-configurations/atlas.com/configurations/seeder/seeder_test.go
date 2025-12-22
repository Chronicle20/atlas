package seeder

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestDefaultConfig(t *testing.T) {
	// Test with no environment variables set
	os.Unsetenv("SEED_DATA_PATH")
	os.Unsetenv("SEED_ENABLED")

	config := DefaultConfig()

	if config.SeedPath != "/seed-data" {
		t.Errorf("Expected default SeedPath to be '/seed-data', got '%s'", config.SeedPath)
	}

	if !config.Enabled {
		t.Error("Expected default Enabled to be true")
	}
}

func TestDefaultConfigWithEnvVars(t *testing.T) {
	// Test with environment variables set
	os.Setenv("SEED_DATA_PATH", "/custom/path")
	os.Setenv("SEED_ENABLED", "false")
	defer func() {
		os.Unsetenv("SEED_DATA_PATH")
		os.Unsetenv("SEED_ENABLED")
	}()

	config := DefaultConfig()

	if config.SeedPath != "/custom/path" {
		t.Errorf("Expected SeedPath to be '/custom/path', got '%s'", config.SeedPath)
	}

	if config.Enabled {
		t.Error("Expected Enabled to be false when SEED_ENABLED=false")
	}
}

func TestDiscoverFiles(t *testing.T) {
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel) // Suppress logs during tests

	s := &Seeder{
		l:   l,
		ctx: context.Background(),
		config: Config{
			SeedPath: "testdata",
			Enabled:  true,
		},
	}

	tests := []struct {
		name          string
		dir           string
		expectedCount int
		expectError   bool
	}{
		{
			name:          "templates directory with json files",
			dir:           "testdata/templates",
			expectedCount: 3, // valid_template.json, invalid_json.json, missing_region.json
			expectError:   false,
		},
		{
			name:          "non-existent directory",
			dir:           "testdata/nonexistent",
			expectedCount: 0,
			expectError:   false, // Should not error, just return empty
		},
		{
			name:          "directory with non-json files",
			dir:           "testdata",
			expectedCount: 0, // not_json.txt should be ignored, subdirs are not files
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := s.discoverFiles(tt.dir)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if len(files) != tt.expectedCount {
				t.Errorf("Expected %d files, got %d: %v", tt.expectedCount, len(files), files)
			}
		})
	}
}

func TestDiscoverFilesSorting(t *testing.T) {
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)

	s := &Seeder{
		l:   l,
		ctx: context.Background(),
		config: Config{
			SeedPath: "testdata",
			Enabled:  true,
		},
	}

	files, err := s.discoverFiles("testdata/templates")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify files are sorted
	for i := 1; i < len(files); i++ {
		if files[i-1] > files[i] {
			t.Errorf("Files not sorted: %s should come before %s", files[i], files[i-1])
		}
	}
}

func TestExtractMetadata(t *testing.T) {
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)

	s := &Seeder{
		l:   l,
		ctx: context.Background(),
	}

	tests := []struct {
		name        string
		filePath    string
		expectError bool
		region      string
		major       uint16
		minor       uint16
	}{
		{
			name:        "valid template",
			filePath:    "testdata/templates/valid_template.json",
			expectError: false,
			region:      "TEST",
			major:       1,
			minor:       0,
		},
		{
			name:        "invalid json",
			filePath:    "testdata/templates/invalid_json.json",
			expectError: true,
		},
		{
			name:        "missing region",
			filePath:    "testdata/templates/missing_region.json",
			expectError: true,
		},
		{
			name:        "non-existent file",
			filePath:    "testdata/templates/does_not_exist.json",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta, err := s.extractMetadata(tt.filePath)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if meta.Region != tt.region {
				t.Errorf("Expected region '%s', got '%s'", tt.region, meta.Region)
			}

			if meta.MajorVersion != tt.major {
				t.Errorf("Expected majorVersion %d, got %d", tt.major, meta.MajorVersion)
			}

			if meta.MinorVersion != tt.minor {
				t.Errorf("Expected minorVersion %d, got %d", tt.minor, meta.MinorVersion)
			}
		})
	}
}

func TestRunWithSeedingDisabled(t *testing.T) {
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)

	s := &Seeder{
		l:   l,
		ctx: context.Background(),
		db:  nil, // DB not needed when disabled
		config: Config{
			SeedPath: "testdata",
			Enabled:  false,
		},
	}

	err := s.Run()
	if err != nil {
		t.Errorf("Expected no error when seeding disabled, got: %v", err)
	}
}

func TestDiscoverFilesOnlyJson(t *testing.T) {
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)

	s := &Seeder{
		l:   l,
		ctx: context.Background(),
	}

	files, err := s.discoverFiles("testdata/templates")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	for _, f := range files {
		if filepath.Ext(f) != ".json" {
			t.Errorf("Non-JSON file discovered: %s", f)
		}
	}
}
