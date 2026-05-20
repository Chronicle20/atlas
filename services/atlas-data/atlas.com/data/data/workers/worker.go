package workers

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	minio "atlas-data/storage/minio"
)

// Params is the shared input for every Worker.Run invocation.
type Params struct {
	ScopeKey     string // "shared" or "tenants/<tenantId>"
	Region       string
	MajorVersion uint32
	MinorVersion uint32
	ScratchDir   string
}

// Worker is the per-archive ingest step. ArchiveName is the key suffix under
// MINIO_BUCKET_WZ at "<scope>/regions/<region>/versions/<major>.<minor>/<archive>".
type Worker interface {
	Name() string
	ArchiveName() string
	Run(ctx context.Context, l logrus.FieldLogger, db *gorm.DB, mc *minio.Client, img *wz.Image, p Params) error
}
