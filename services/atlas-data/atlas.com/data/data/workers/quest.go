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

func (Quest) Run(ctx context.Context, l logrus.FieldLogger, db *gorm.DB, mc *minio.Client, img *wz.Image, p Params) error {
	l.Infof("TODO Task 8: implement QUEST worker (archive=%s scope=%s region=%s version=%d.%d)", "Quest.wz", p.ScopeKey, p.Region, p.MajorVersion, p.MinorVersion)
	return nil
}
