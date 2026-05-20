package workers

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	minio "atlas-data/storage/minio"
)

type Item struct{}

func (Item) Name() string        { return "ITEM" }
func (Item) ArchiveName() string { return "Item.wz" }

func (Item) Run(ctx context.Context, l logrus.FieldLogger, db *gorm.DB, mc *minio.Client, img *wz.Image, p Params) error {
	l.Infof("TODO Task 8: implement ITEM worker (archive=%s scope=%s region=%s version=%d.%d)", "Item.wz", p.ScopeKey, p.Region, p.MajorVersion, p.MinorVersion)
	return nil
}
