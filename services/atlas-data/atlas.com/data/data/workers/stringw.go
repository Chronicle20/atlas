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

func (String) Run(ctx context.Context, l logrus.FieldLogger, db *gorm.DB, mc *minio.Client, img *wz.Image, p Params) error {
	l.Infof("TODO Task 8: implement STRING worker (archive=%s scope=%s region=%s version=%d.%d)", "String.wz", p.ScopeKey, p.Region, p.MajorVersion, p.MinorVersion)
	return nil
}
