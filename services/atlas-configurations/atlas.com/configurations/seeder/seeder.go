package seeder

import (
	"atlas-configurations/templates"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Config holds the seeder configuration
type Config struct {
	SeedPath string
	Enabled  bool
}

// DefaultConfig returns the default seeder configuration
func DefaultConfig() Config {
	seedPath := os.Getenv("SEED_DATA_PATH")
	if seedPath == "" {
		seedPath = "/seed-data"
	}

	enabled := true
	if os.Getenv("SEED_ENABLED") == "false" {
		enabled = false
	}

	return Config{
		SeedPath: seedPath,
		Enabled:  enabled,
	}
}

// ConfigMetadata represents the minimal JSON structure needed to identify a configuration
type ConfigMetadata struct {
	Region       string `json:"region"`
	MajorVersion uint16 `json:"majorVersion"`
	MinorVersion uint16 `json:"minorVersion"`
}

// SeedResult tracks the outcome of seeding operations
type SeedResult struct {
	Imported int
	Skipped  int
	Failed   int
}

// Seeder handles importing seed data into the database
type Seeder struct {
	l      logrus.FieldLogger
	ctx    context.Context
	db     *gorm.DB
	config Config
}

// NewSeeder creates a new Seeder instance
func NewSeeder(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, config Config) *Seeder {
	return &Seeder{
		l:      l,
		ctx:    ctx,
		db:     db,
		config: config,
	}
}

// Run executes the seeding process
func (s *Seeder) Run() error {
	if !s.config.Enabled {
		s.l.Info("Seeding is disabled via SEED_ENABLED=false")
		return nil
	}

	s.l.WithField("path", s.config.SeedPath).Info("Starting seed import")

	// Seed templates
	result := s.seedTemplates()
	s.l.WithFields(logrus.Fields{
		"imported": result.Imported,
		"skipped":  result.Skipped,
		"failed":   result.Failed,
	}).Info("Template seeding complete")

	return nil
}

// seedTemplates imports all template seed files
func (s *Seeder) seedTemplates() SeedResult {
	result := SeedResult{}
	templatesPath := filepath.Join(s.config.SeedPath, "templates")

	files, err := s.discoverFiles(templatesPath)
	if err != nil {
		s.l.WithError(err).Warn("Failed to discover template files")
		return result
	}

	if len(files) == 0 {
		s.l.WithField("path", templatesPath).Debug("No template seed files found")
		return result
	}

	s.l.WithField("count", len(files)).Info("Discovered template seed files")

	for _, file := range files {
		outcome := s.importTemplate(file)
		switch outcome {
		case "imported":
			result.Imported++
		case "skipped":
			result.Skipped++
		case "failed":
			result.Failed++
		}
	}

	return result
}

// discoverFiles finds all JSON files in the specified directory
func (s *Seeder) discoverFiles(dir string) ([]string, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		s.l.WithField("directory", dir).Debug("Seed directory does not exist")
		return []string{}, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) == ".json" {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}

	// Sort for deterministic ordering
	sort.Strings(files)
	return files, nil
}

// extractMetadata reads a JSON file and extracts the configuration metadata
func (s *Seeder) extractMetadata(filePath string) (*ConfigMetadata, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var meta ConfigMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}

	if meta.Region == "" {
		return nil, errors.New("missing required field: region")
	}

	return &meta, nil
}

// templateExists checks if a template with the given identifiers already exists
func (s *Seeder) templateExists(region string, majorVersion uint16, minorVersion uint16) (bool, error) {
	processor := templates.NewProcessor(s.l, s.ctx, s.db)
	_, err := processor.GetByRegionAndVersion(region, majorVersion, minorVersion)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// importTemplate imports a single template file if it doesn't already exist
func (s *Seeder) importTemplate(filePath string) string {
	fileName := filepath.Base(filePath)
	l := s.l.WithField("file", fileName)

	// Extract metadata to check existence
	meta, err := s.extractMetadata(filePath)
	if err != nil {
		l.WithError(err).Error("Failed to extract template metadata")
		return "failed"
	}

	l = l.WithFields(logrus.Fields{
		"region":       meta.Region,
		"majorVersion": meta.MajorVersion,
		"minorVersion": meta.MinorVersion,
	})

	// Check if already exists
	exists, err := s.templateExists(meta.Region, meta.MajorVersion, meta.MinorVersion)
	if err != nil {
		l.WithError(err).Error("Failed to check template existence")
		return "failed"
	}

	if exists {
		l.Debug("Template already exists, skipping")
		return "skipped"
	}

	// Read full file content
	data, err := os.ReadFile(filePath)
	if err != nil {
		l.WithError(err).Error("Failed to read template file")
		return "failed"
	}

	// Parse into RestModel
	var model templates.RestModel
	if err := json.Unmarshal(data, &model); err != nil {
		l.WithError(err).Error("Failed to parse template JSON")
		return "failed"
	}

	// Create the template
	processor := templates.NewProcessor(s.l, s.ctx, s.db)
	id, err := processor.Create(model)
	if err != nil {
		l.WithError(err).Error("Failed to create template")
		return "failed"
	}

	l.WithField("id", id.String()).Info("Template imported successfully")
	return "imported"
}
