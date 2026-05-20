package workers

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/Chronicle20/atlas/libs/atlas-wz/icons"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"atlas-data/reactor"
	minio "atlas-data/storage/minio"
)

type Reactor struct{}

func (Reactor) Name() string        { return "REACTOR" }
func (Reactor) ArchiveName() string { return "Reactor.wz" }

func (Reactor) Run(ctx context.Context, l logrus.FieldLogger, db *gorm.DB, mc *minio.Client, file *wz.File, p Params) error {
	ctx, _, err := withTenant(ctx, p)
	if err != nil {
		return err
	}
	root, err := serializeArchive(l, p, file)
	if err != nil {
		return fmt.Errorf("serialize Reactor.wz: %w", err)
	}
	if err := registerAllInDirectory(l, ctx, filepath.Join(root, "Reactor.wz"), reactor.RegisterReactor(db)); err != nil {
		return err
	}
	prefix := minioAssetPrefix(p)
	for _, img := range file.Root().Images() {
		id, ok := imgID(img.Name())
		if !ok {
			continue
		}
		icon, err := icons.ExtractReactorIcon(file, id)
		if err != nil || icon == nil {
			continue
		}
		key := fmt.Sprintf("%s/reactor/%d/icon.png", prefix, id)
		if err := putPNG(ctx, mc, key, icon); err != nil {
			l.WithError(err).Warnf("upload reactor icon %d", id)
		}
	}
	return nil
}
