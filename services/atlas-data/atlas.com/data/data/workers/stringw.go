package workers

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	minio "atlas-data/storage/minio"
)

type String struct{}

func (String) Name() string        { return "STRING" }
func (String) ArchiveName() string { return "String.wz" }

func (String) Run(ctx context.Context, l logrus.FieldLogger, db *gorm.DB, mc *minio.Client, file *wz.File, p Params) error {
	return nil
}
