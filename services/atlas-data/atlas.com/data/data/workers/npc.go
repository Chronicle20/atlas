package workers

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	minio "atlas-data/storage/minio"
)

type Npc struct{}

func (Npc) Name() string        { return "NPC" }
func (Npc) ArchiveName() string { return "Npc.wz" }

func (Npc) Run(ctx context.Context, l logrus.FieldLogger, db *gorm.DB, mc *minio.Client, img *wz.Image, p Params) error {
	l.Infof("TODO Task 8: implement NPC worker (archive=%s scope=%s region=%s version=%d.%d)", "Npc.wz", p.ScopeKey, p.Region, p.MajorVersion, p.MinorVersion)
	return nil
}
