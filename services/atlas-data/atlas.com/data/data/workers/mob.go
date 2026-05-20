package workers

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	minio "atlas-data/storage/minio"
)

type Mob struct{}

func (Mob) Name() string        { return "MOB" }
func (Mob) ArchiveName() string { return "Mob.wz" }

func (Mob) Run(ctx context.Context, l logrus.FieldLogger, db *gorm.DB, mc *minio.Client, img *wz.Image, p Params) error {
	l.Infof("TODO Task 8: implement MOB worker (archive=%s scope=%s region=%s version=%d.%d)", "Mob.wz", p.ScopeKey, p.Region, p.MajorVersion, p.MinorVersion)
	return nil
}
