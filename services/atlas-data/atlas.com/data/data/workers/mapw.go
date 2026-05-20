package workers

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	minio "atlas-data/storage/minio"
)

type Map struct{}

func (Map) Name() string        { return "MAP" }
func (Map) ArchiveName() string { return "Map.wz" }

func (Map) Run(ctx context.Context, l logrus.FieldLogger, db *gorm.DB, mc *minio.Client, img *wz.Image, p Params) error {
	l.Infof("TODO Task 8: implement MAP worker (archive=%s scope=%s region=%s version=%d.%d)", "Map.wz", p.ScopeKey, p.Region, p.MajorVersion, p.MinorVersion)
	return nil
}
