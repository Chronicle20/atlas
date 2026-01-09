package shops

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadShopFilesValidData(t *testing.T) {
	// Set up test directory
	originalPath := ShopsPath
	ShopsPath = "testdata"
	defer func() { ShopsPath = originalPath }()

	models, errors := LoadShopFiles()

	// Should have one valid shop (valid_shop.json)
	// invalid.json should cause an error
	// not_json.txt should be ignored
	// schema.json should be skipped

	if len(models) != 1 {
		t.Errorf("Expected 1 model, got %d", len(models))
	}

	if len(errors) != 1 {
		t.Errorf("Expected 1 error (from invalid.json), got %d", len(errors))
	}

	if len(models) > 0 {
		shop := models[0]
		if shop.NpcId != 99999 {
			t.Errorf("Expected NpcId 99999, got %d", shop.NpcId)
		}
		if !shop.Recharger {
			t.Error("Expected Recharger to be true")
		}
		if len(shop.Commodities) != 2 {
			t.Errorf("Expected 2 commodities, got %d", len(shop.Commodities))
		}
	}
}

func TestLoadShopFilesEmptyDirectory(t *testing.T) {
	// Create a temporary empty directory
	tempDir, err := os.MkdirTemp("", "empty_shops_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalPath := ShopsPath
	ShopsPath = tempDir
	defer func() { ShopsPath = originalPath }()

	models, errors := LoadShopFiles()

	if len(models) != 0 {
		t.Errorf("Expected 0 models from empty directory, got %d", len(models))
	}

	if len(errors) != 0 {
		t.Errorf("Expected 0 errors from empty directory, got %d", len(errors))
	}
}

func TestLoadShopFilesIgnoresNonJSON(t *testing.T) {
	// Create a temporary directory with only non-JSON files
	tempDir, err := os.MkdirTemp("", "non_json_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a non-JSON file
	err = os.WriteFile(filepath.Join(tempDir, "readme.txt"), []byte("not json"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	originalPath := ShopsPath
	ShopsPath = tempDir
	defer func() { ShopsPath = originalPath }()

	models, errors := LoadShopFiles()

	if len(models) != 0 {
		t.Errorf("Expected 0 models, got %d", len(models))
	}

	if len(errors) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(errors))
	}
}

func TestLoadShopFilesIgnoresDirectories(t *testing.T) {
	// Create a temporary directory with a subdirectory
	tempDir, err := os.MkdirTemp("", "subdir_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a subdirectory
	err = os.Mkdir(filepath.Join(tempDir, "subdir"), 0755)
	if err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	originalPath := ShopsPath
	ShopsPath = tempDir
	defer func() { ShopsPath = originalPath }()

	models, errors := LoadShopFiles()

	if len(models) != 0 {
		t.Errorf("Expected 0 models, got %d", len(models))
	}

	if len(errors) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(errors))
	}
}

func TestLoadShopFilesSkipsSchema(t *testing.T) {
	// Create a temporary directory with only schema.json
	tempDir, err := os.MkdirTemp("", "schema_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create schema.json
	schemaContent := `{"$schema": "test"}`
	err = os.WriteFile(filepath.Join(tempDir, "schema.json"), []byte(schemaContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create schema file: %v", err)
	}

	originalPath := ShopsPath
	ShopsPath = tempDir
	defer func() { ShopsPath = originalPath }()

	models, errors := LoadShopFiles()

	if len(models) != 0 {
		t.Errorf("Expected 0 models (schema should be skipped), got %d", len(models))
	}

	if len(errors) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(errors))
	}
}

func TestLoadShopFilesHandlesInvalidJSON(t *testing.T) {
	// Create a temporary directory with invalid JSON
	tempDir, err := os.MkdirTemp("", "invalid_json_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create invalid JSON file
	err = os.WriteFile(filepath.Join(tempDir, "bad.json"), []byte("{invalid}"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	originalPath := ShopsPath
	ShopsPath = tempDir
	defer func() { ShopsPath = originalPath }()

	models, errors := LoadShopFiles()

	if len(models) != 0 {
		t.Errorf("Expected 0 models from invalid JSON, got %d", len(models))
	}

	if len(errors) != 1 {
		t.Errorf("Expected 1 error from invalid JSON, got %d", len(errors))
	}
}

func TestLoadShopFilesNonExistentDirectory(t *testing.T) {
	originalPath := ShopsPath
	ShopsPath = "/nonexistent/path/that/does/not/exist"
	defer func() { ShopsPath = originalPath }()

	models, errors := LoadShopFiles()

	if models != nil {
		t.Error("Expected nil models from nonexistent directory")
	}

	if len(errors) != 1 {
		t.Errorf("Expected 1 error from nonexistent directory, got %d", len(errors))
	}
}
