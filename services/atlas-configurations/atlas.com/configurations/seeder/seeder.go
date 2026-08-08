package seeder

import (
	"atlas-configurations/services"
	"atlas-configurations/templates"
	"atlas-configurations/tenants"
	"context"
	"errors"
	"os"
	"path/filepath"

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

	enabled := os.Getenv("SEED_ENABLED") != "false"

	return Config{
		SeedPath: seedPath,
		Enabled:  enabled,
	}
}

// SeedResult tracks the outcome of seeding operations
type SeedResult struct {
	Imported int
	Skipped  int
	Failed   int
}

// Seeder handles importing seed data into the database
type Seeder struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	// catalog is the shipped-template catalog, loaded once in main.go. The
	// seeder no longer reads or parses files itself: templates.LoadCatalog is
	// the single parse path (FR-1.7), so the drift comparison and the boot
	// import can never disagree about what a file contains.
	catalog templates.Catalog
	config  Config
}

// NewSeeder creates a new Seeder instance
func NewSeeder(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, config Config, catalog templates.Catalog) *Seeder {
	return &Seeder{
		l:       l,
		ctx:     ctx,
		db:      db,
		config:  config,
		catalog: catalog,
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

	// Backfill the transactional outbox from existing service+tenant rows
	// so a fresh-cluster boot (or a cluster recovering from a wiped Kafka
	// topic) has a complete snapshot to publish. Idempotent on (topic, key).
	if n, err := services.Backfill(s.db); err != nil {
		s.l.WithError(err).Warn("outbox service backfill failed")
	} else if n > 0 {
		s.l.WithField("count", n).Info("seeder.backfill.services")
	}
	if n, err := tenants.Backfill(s.db); err != nil {
		s.l.WithError(err).Warn("outbox tenant backfill failed")
	} else if n > 0 {
		s.l.WithField("count", n).Info("seeder.backfill.tenants")
	}

	return nil
}

// seedTemplates imports every shipped template that does not already exist.
// The outcome strings and SeedResult counters are unchanged.
func (s *Seeder) seedTemplates() SeedResult {
	result := SeedResult{}

	entries := s.catalog.Entries()
	if len(entries) == 0 {
		s.l.WithField("path", filepath.Join(s.config.SeedPath, "templates")).Debug("No template seed files found")
		return result
	}

	s.l.WithField("count", len(entries)).Info("Discovered template seed files")

	for _, entry := range entries {
		switch s.importTemplate(entry) {
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

// importTemplate creates a template if no row exists on its key.
//
// CREATE-IF-ABSENT, DELIBERATELY (FR-4.1). It returns "skipped" for an
// existing key regardless of whether the file's content differs from the
// stored row. Reconciling on boot would silently discard operator edits on
// every redeploy; correcting drift is an explicit operator action
// (POST /configurations/templates/{id}/reseed), never a startup side effect.
func (s *Seeder) importTemplate(entry templates.CatalogEntry) string {
	l := s.l.WithFields(logrus.Fields{
		"file":         entry.FileName,
		"region":       entry.Model.Region,
		"majorVersion": entry.Model.MajorVersion,
		"minorVersion": entry.Model.MinorVersion,
	})

	exists, err := s.templateExists(entry.Model.Region, entry.Model.MajorVersion, entry.Model.MinorVersion)
	if err != nil {
		l.WithError(err).Error("Failed to check template existence")
		return "failed"
	}
	if exists {
		l.Debug("Template already exists, skipping")
		return "skipped"
	}

	id, err := templates.NewProcessor(s.l, s.ctx, s.db).Create(entry.Model)
	if err != nil {
		l.WithError(err).Error("Failed to create template")
		return "failed"
	}

	l.WithField("id", id.String()).Info("Template imported successfully")
	return "imported"
}
