package workers

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"atlas-data/quest"
	minio "atlas-data/storage/minio"
)

type Quest struct{}

func (Quest) Name() string        { return "QUEST" }
func (Quest) ArchiveName() string { return "Quest.wz" }

func (Quest) Run(ctx context.Context, l logrus.FieldLogger, db *gorm.DB, mc *minio.Client, file *wz.File, p Params) error {
	ctx, _, err := withTenant(ctx, p)
	if err != nil {
		return err
	}
	root, err := serializeArchive(l, p, file)
	if err != nil {
		return fmt.Errorf("serialize Quest.wz: %w", err)
	}
	// quest.RegisterQuest reads QuestInfo.img.xml + Check.img.xml + Act.img.xml
	// out of the given directory.
	return quest.RegisterQuest(db)(l)(ctx)(filepath.Join(root, "Quest.wz"))
}
