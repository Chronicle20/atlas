package workers

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/Chronicle20/atlas/libs/atlas-wz/icons"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"atlas-data/cash"
	"atlas-data/consumable"
	"atlas-data/etc"
	"atlas-data/pet"
	"atlas-data/setup"
	minio "atlas-data/storage/minio"
)

type Item struct{}

func (Item) Name() string        { return "ITEM" }
func (Item) ArchiveName() string { return "Item.wz" }

func (Item) Run(ctx context.Context, l logrus.FieldLogger, db *gorm.DB, mc *minio.Client, file *wz.File, p Params) error {
	ctx, _, err := withTenant(ctx, p)
	if err != nil {
		return err
	}
	root, err := serializeArchive(l, p, file)
	if err != nil {
		return fmt.Errorf("serialize Item.wz: %w", err)
	}
	base := filepath.Join(root, "Item.wz")
	categories := []struct {
		subdir string
		rf     RegisterFunc
	}{
		{"Consume", consumable.RegisterConsumable(db)},
		{"Cash", cash.RegisterCash(db)},
		{"Etc", etc.RegisterEtc(db)},
		{"Install", setup.RegisterSetup(db)},
		{"Pet", pet.RegisterPet(db)},
	}
	for _, c := range categories {
		dir := filepath.Join(base, c.subdir)
		if err := registerAllInDirectory(l, ctx, dir, c.rf); err != nil {
			l.WithError(err).Warnf("walk %s", dir)
		}
	}

	// Emit per-item icons (Consume/Cash/Etc/Install/Pet share Item.wz layout).
	prefix := minioAssetPrefix(p)
	for _, sub := range file.Root().Directories() {
		for _, img := range sub.Images() {
			id, ok := imgID(img.Name())
			if !ok {
				continue
			}
			icon, err := icons.ExtractItemIcon(file, id)
			if err != nil || icon == nil {
				continue
			}
			key := fmt.Sprintf("%s/item/%d/icon.png", prefix, id)
			if err := putPNG(ctx, mc, key, icon); err != nil {
				l.WithError(err).Warnf("upload item icon %d", id)
			}
		}
	}
	return nil
}
