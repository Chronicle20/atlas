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
	"atlas-data/monster"
)

type Mob struct{}

func (Mob) Name() string        { return "MOB" }
func (Mob) ArchiveName() string { return "Mob.wz" }

func (Mob) Run(ctx context.Context, l logrus.FieldLogger, db *gorm.DB, mc *minio.Client, file *wz.File, p Params) error {
	ctx, t, err := withTenant(ctx, p)
	if err != nil {
		return err
	}
	// Serialize Mob.wz to scratch so the existing per-img readers can run.
	root, err := serializeArchive(l, p, file)
	if err != nil {
		return fmt.Errorf("serialize Mob.wz: %w", err)
	}
	// String.wz Mob.img / UI.wz UIWindow.img are referenced by monster.InitString
	// and monster.InitGauge. Fetch them so registry lookups succeed.
	if _, err := fetchAndSerializeArchive(ctx, l, mc, p, "String.wz"); err != nil {
		l.WithError(err).Warnf("String.wz unavailable; monster names will be empty")
	} else {
		if err := monster.InitString(t, filepath.Join(root, "String.wz", "Mob.img.xml")); err != nil {
			l.WithError(err).Warnf("monster.InitString failed; names will be empty")
		}
		defer func() { _ = monster.GetMonsterStringRegistry().Clear(t) }()
	}
	if _, err := fetchAndSerializeArchive(ctx, l, mc, p, "UI.wz"); err != nil {
		l.WithError(err).Warnf("UI.wz unavailable; monster gauges will be empty")
	} else {
		if err := monster.InitGauge(t, filepath.Join(root, "UI.wz", "UIWindow.img.xml")); err != nil {
			l.WithError(err).Warnf("monster.InitGauge failed; gauges will be empty")
		}
		defer func() { _ = monster.GetMonsterGaugeRegistry().Clear(t) }()
	}

	// Register every mob image.
	if err := registerAllInDirectory(l, ctx, filepath.Join(root, "Mob.wz"), monster.RegisterMonster(db)); err != nil {
		return err
	}

	// Emit per-mob icons to MinIO (best-effort; missing icons are not fatal).
	prefix := minioAssetPrefix(p)
	for _, img := range file.Root().Images() {
		id, ok := imgID(img.Name())
		if !ok {
			continue
		}
		icon, err := icons.ExtractMobIcon(file, id)
		if err != nil || icon == nil {
			continue
		}
		key := fmt.Sprintf("%s/mob/%d/icon.png", prefix, id)
		if err := putPNG(ctx, mc, key, icon); err != nil {
			l.WithError(err).Warnf("upload mob icon %d", id)
		}
	}
	return nil
}
