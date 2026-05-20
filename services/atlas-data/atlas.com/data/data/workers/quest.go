package workers

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	minio "atlas-data/storage/minio"
)

type Quest struct{}

func (Quest) Name() string        { return "QUEST" }
func (Quest) ArchiveName() string { return "Quest.wz" }

func (Quest) Run(ctx context.Context, l logrus.FieldLogger, db *gorm.DB, mc *minio.Client, file *wz.File, p Params) error {
	return nil
}
