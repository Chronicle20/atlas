package workers

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"atlas-data/commodity"
	minio "atlas-data/storage/minio"
)

// Commodity ingests cash-shop commodity rows from Etc.wz/Commodity.img.xml.
// It exists as a dedicated worker (rather than living inside Character or a
// shared Etc worker) because Commodity is the only Postgres-side ingest under
// Etc.wz; the Character worker also fetches Etc.wz internally for
// MakeCharInfo, and the double-fetch cost is negligible relative to the WZ
// download. Splitting keeps each worker's responsibility legible.
type Commodity struct{}

func (Commodity) Name() string        { return "COMMODITY" }
func (Commodity) ArchiveName() string { return "Etc.wz" }

func (Commodity) Run(ctx context.Context, l logrus.FieldLogger, db *gorm.DB, mc *minio.Client, file *wz.File, p Params) error {
	if _, _, err := withTenant(ctx, p); err != nil {
		return err
	}
	root, err := serializeArchive(l, p, file)
	if err != nil {
		return fmt.Errorf("serialize Etc.wz: %w", err)
	}
	commodityPath := filepath.Join(root, "Etc.wz", "Commodity.img.xml")
	if err := commodity.RegisterCommodity(db)(l)(ctx)(commodityPath); err != nil {
		return fmt.Errorf("register commodities: %w", err)
	}
	return nil
}
