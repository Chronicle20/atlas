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
	MajorVersion uint16
	MinorVersion uint16
	ScratchDir   string
}

// Worker is the per-archive ingest step. ArchiveName is the key suffix under
// MINIO_BUCKET_WZ at "<scope>/regions/<region>/versions/<major>.<minor>/<archive>".
// The Run method receives an opened *wz.File — workers walk the file's root
// directory themselves (each .wz archive contains many .img entries).
type Worker interface {
	Name() string
	ArchiveName() string
	Run(ctx context.Context, l logrus.FieldLogger, db *gorm.DB, mc *minio.Client, file *wz.File, p Params) error
}
