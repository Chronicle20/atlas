package drop

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadContinentDropFiles(t *testing.T) {
	tests := []struct {
		name           string
		setupPath      func() string
		expectedCount  int
		expectedErrors int
	}{
		{
			name: "valid JSON file",
			setupPath: func() string {
				return "testdata"
			},
			expectedCount:  2, // valid_drops.json has 2 entries
			expectedErrors: 1, // invalid.json should produce 1 error
		},
		{
			name: "non-existent directory",
			setupPath: func() string {
				return "testdata/nonexistent"
			},
			expectedCount:  0,
			expectedErrors: 1, // Should have error for missing directory
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("CONTINENT_DROPS_PATH", tt.setupPath())
			defer os.Unsetenv("CONTINENT_DROPS_PATH")

			models, errs := LoadContinentDropFiles()

			if len(models) != tt.expectedCount {
				t.Errorf("Expected %d models, got %d", tt.expectedCount, len(models))
			}

			if len(errs) != tt.expectedErrors {
				t.Errorf("Expected %d errors, got %d: %v", tt.expectedErrors, len(errs), errs)
			}
		})
	}
}

func TestLoadContinentDropFilesValidData(t *testing.T) {
	// Create a temp directory with only valid data
	tempDir, err := os.MkdirTemp("", "continent_drops_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Write valid test data
	validJSON := `[
		{"continentId": -1, "itemId": 200, "minimumQuantity": 1, "maximumQuantity": 5, "chance": 10000},
		{"continentId": 0, "itemId": 201, "minimumQuantity": 2, "maximumQuantity": 3, "chance": 50000}
	]`
	err = os.WriteFile(filepath.Join(tempDir, "test.json"), []byte(validJSON), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	os.Setenv("CONTINENT_DROPS_PATH", tempDir)
	defer os.Unsetenv("CONTINENT_DROPS_PATH")

	models, errs := LoadContinentDropFiles()

	if len(errs) != 0 {
		t.Errorf("Expected no errors, got %d: %v", len(errs), errs)
	}

	if len(models) != 2 {
		t.Errorf("Expected 2 models, got %d", len(models))
	}

	// Verify first model
	if len(models) > 0 {
		if models[0].ContinentId != -1 {
			t.Errorf("Expected ContinentId -1, got %d", models[0].ContinentId)
		}
		if models[0].ItemId != 200 {
			t.Errorf("Expected ItemId 200, got %d", models[0].ItemId)
		}
		if models[0].MinimumQuantity != 1 {
			t.Errorf("Expected MinimumQuantity 1, got %d", models[0].MinimumQuantity)
		}
		if models[0].MaximumQuantity != 5 {
			t.Errorf("Expected MaximumQuantity 5, got %d", models[0].MaximumQuantity)
		}
		if models[0].Chance != 10000 {
			t.Errorf("Expected Chance 10000, got %d", models[0].Chance)
		}
	}

	// Verify second model
	if len(models) > 1 {
		if models[1].ContinentId != 0 {
			t.Errorf("Expected ContinentId 0, got %d", models[1].ContinentId)
		}
		if models[1].Chance != 50000 {
			t.Errorf("Expected Chance 50000, got %d", models[1].Chance)
		}
	}
}

func TestLoadContinentDropFilesEmptyDirectory(t *testing.T) {
	// Create an empty temp directory
	tempDir, err := os.MkdirTemp("", "continent_drops_empty_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	os.Setenv("CONTINENT_DROPS_PATH", tempDir)
	defer os.Unsetenv("CONTINENT_DROPS_PATH")

	models, errs := LoadContinentDropFiles()

	if len(errs) != 0 {
		t.Errorf("Expected no errors for empty directory, got %d: %v", len(errs), errs)
	}

	if len(models) != 0 {
		t.Errorf("Expected 0 models for empty directory, got %d", len(models))
	}
}

func TestLoadContinentDropFilesIgnoresNonJSON(t *testing.T) {
	// Create a temp directory with non-JSON files
	tempDir, err := os.MkdirTemp("", "continent_drops_nonjson_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Write a non-JSON file
	err = os.WriteFile(filepath.Join(tempDir, "readme.txt"), []byte("This is not JSON"), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Write a valid JSON file
	validJSON := `[{"continentId": -1, "itemId": 200, "minimumQuantity": 1, "maximumQuantity": 1, "chance": 10000}]`
	err = os.WriteFile(filepath.Join(tempDir, "drops.json"), []byte(validJSON), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	os.Setenv("CONTINENT_DROPS_PATH", tempDir)
	defer os.Unsetenv("CONTINENT_DROPS_PATH")

	models, errs := LoadContinentDropFiles()

	if len(errs) != 0 {
		t.Errorf("Expected no errors, got %d: %v", len(errs), errs)
	}

	// Should only have 1 model from the JSON file, ignoring the .txt file
	if len(models) != 1 {
		t.Errorf("Expected 1 model (ignoring non-JSON), got %d", len(models))
	}
}

func TestLoadContinentDropFilesIgnoresDirectories(t *testing.T) {
	// Create a temp directory with a subdirectory
	tempDir, err := os.MkdirTemp("", "continent_drops_subdir_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a subdirectory
	err = os.Mkdir(filepath.Join(tempDir, "subdir"), 0755)
	if err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	// Write a JSON file in subdirectory (should be ignored)
	subJSON := `[{"continentId": 99, "itemId": 999, "minimumQuantity": 1, "maximumQuantity": 1, "chance": 10000}]`
	err = os.WriteFile(filepath.Join(tempDir, "subdir", "drops.json"), []byte(subJSON), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Write a valid JSON file in root
	validJSON := `[{"continentId": -1, "itemId": 200, "minimumQuantity": 1, "maximumQuantity": 1, "chance": 10000}]`
	err = os.WriteFile(filepath.Join(tempDir, "drops.json"), []byte(validJSON), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	os.Setenv("CONTINENT_DROPS_PATH", tempDir)
	defer os.Unsetenv("CONTINENT_DROPS_PATH")

	models, errs := LoadContinentDropFiles()

	if len(errs) != 0 {
		t.Errorf("Expected no errors, got %d: %v", len(errs), errs)
	}

	// Should only have 1 model from root, subdirectory should be ignored
	if len(models) != 1 {
		t.Errorf("Expected 1 model (ignoring subdirectory), got %d", len(models))
	}

	if len(models) > 0 && models[0].ContinentId != -1 {
		t.Errorf("Expected ContinentId -1 from root file, got %d", models[0].ContinentId)
	}
}

func TestGetContinentDropsPathDefault(t *testing.T) {
	// Ensure env var is not set
	os.Unsetenv("CONTINENT_DROPS_PATH")

	path := GetContinentDropsPath()
	if path != defaultContinentDropsPath {
		t.Errorf("Expected default path %s, got %s", defaultContinentDropsPath, path)
	}
}

func TestGetContinentDropsPathFromEnv(t *testing.T) {
	customPath := "/custom/path/to/drops"
	os.Setenv("CONTINENT_DROPS_PATH", customPath)
	defer os.Unsetenv("CONTINENT_DROPS_PATH")

	path := GetContinentDropsPath()
	if path != customPath {
		t.Errorf("Expected custom path %s, got %s", customPath, path)
	}
}
