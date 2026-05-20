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

func (Npc) Run(ctx context.Context, l logrus.FieldLogger, db *gorm.DB, mc *minio.Client, file *wz.File, p Params) error {
	return nil
}
