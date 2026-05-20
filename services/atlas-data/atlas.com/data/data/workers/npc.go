package workers

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/Chronicle20/atlas/libs/atlas-wz/icons"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	minio "atlas-data/storage/minio"
	"atlas-data/npc"
)

type Npc struct{}

func (Npc) Name() string        { return "NPC" }
func (Npc) ArchiveName() string { return "Npc.wz" }

func (Npc) Run(ctx context.Context, l logrus.FieldLogger, db *gorm.DB, mc *minio.Client, file *wz.File, p Params) error {
	ctx, t, err := withTenant(ctx, p)
	if err != nil {
		return err
	}
	root, err := serializeArchive(l, p, file)
	if err != nil {
		return fmt.Errorf("serialize Npc.wz: %w", err)
	}
	if _, err := fetchAndSerializeArchive(ctx, l, mc, p, "String.wz"); err != nil {
		l.WithError(err).Warnf("String.wz unavailable; NPC names will be empty")
	} else {
		if err := npc.InitString(t, filepath.Join(root, "String.wz", "Npc.img.xml")); err != nil {
			l.WithError(err).Warnf("npc.InitString failed")
		}
		// Note: leave NPC string registry populated; Map worker may need it.
	}
	if err := registerAllInDirectory(l, ctx, filepath.Join(root, "Npc.wz"), npc.RegisterNpc(db)); err != nil {
		return err
	}
	prefix := minioAssetPrefix(p)
	for _, img := range file.Root().Images() {
		id, ok := imgID(img.Name())
		if !ok {
			continue
		}
		icon, err := icons.ExtractNpcIcon(file, id)
		if err != nil || icon == nil {
			continue
		}
		key := fmt.Sprintf("%s/npc/%d/icon.png", prefix, id)
		if err := putPNG(ctx, mc, key, icon); err != nil {
			l.WithError(err).Warnf("upload npc icon %d", id)
		}
	}
	return nil
}
