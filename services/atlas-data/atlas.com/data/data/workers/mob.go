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

func (Mob) Run(ctx context.Context, l logrus.FieldLogger, db *gorm.DB, mc *minio.Client, file *wz.File, p Params) error {
	return nil
}
