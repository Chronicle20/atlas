package workers

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	minio "atlas-data/storage/minio"
)

type Character struct{}

func (Character) Name() string        { return "CHARACTER" }
func (Character) ArchiveName() string { return "Character.wz" }

func (Character) Run(ctx context.Context, l logrus.FieldLogger, db *gorm.DB, mc *minio.Client, img *wz.Image, p Params) error {
	l.Infof("TODO Task 8: implement CHARACTER worker (archive=%s scope=%s region=%s version=%d.%d)", "Character.wz", p.ScopeKey, p.Region, p.MajorVersion, p.MinorVersion)
	return nil
}
