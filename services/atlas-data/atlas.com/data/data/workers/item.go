package workers

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-wz/icons"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz/property"
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

	// Emit per-item icons. Item.wz items are stored two ways depending on
	// category. For Pet/, each item is a single-item .img file whose name
	// is the item id (e.g. Pet/5000028.img). For Consume/Cash/Etc/Install,
	// each .img file packs many items as numbered SUB-properties — Consume/
	// 02000000.img contains items 02000001..02000999 (or similar). The
	// initial implementation iterated only .img names, which missed every
	// sub-property item (~24k items → 54 icons emitted on PR-544).
	//
	// Strategy: enumerate every numeric id reachable via either path and
	// dedupe before calling ExtractItemIcon. ExtractItemIcon already handles
	// both layouts when given the correct id (single-item match-by-name or
	// multi-item walk of sub-properties).
	prefix := minioAssetPrefix(p)
	var scanned, extracted, uploaded int
	seen := make(map[uint32]struct{})
	for _, sub := range file.Root().Directories() {
		for _, img := range sub.Images() {
			// Single-item: .img name IS the id (Pet/ + occasional others).
			if id, ok := imgID(img.Name()); ok {
				seen[id] = struct{}{}
			}
			// Multi-item: top-level numeric SUB-properties name the ids.
			props, err := img.Properties()
			if err != nil {
				return fmt.Errorf("item worker: parse %s: %w", img.Name(), err)
			}
			for _, prop := range props {
				sp, ok := prop.(*property.SubProperty)
				if !ok {
					continue
				}
				idU, err := strconv.ParseUint(sp.Name(), 10, 32)
				if err != nil {
					continue
				}
				seen[uint32(idU)] = struct{}{}
			}
		}
	}
	for id := range seen {
		scanned++
		icon, err := icons.ExtractItemIcon(file, id)
		if err != nil || icon == nil {
			continue
		}
		extracted++
		key := fmt.Sprintf("%s/item/%d/icon.png", prefix, id)
		if err := putPNG(ctx, mc, key, icon); err != nil {
			l.WithError(err).Warnf("upload item icon %d", id)
			continue
		}
		uploaded++
	}
	l.Infof("item icons: scanned=%d extracted=%d uploaded=%d", scanned, extracted, uploaded)
	return nil
}
