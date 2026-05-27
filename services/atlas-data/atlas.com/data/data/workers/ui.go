package workers

import (
	"context"
	"fmt"
	"strings"

	"github.com/Chronicle20/atlas/libs/atlas-wz/canvas"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz/property"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	minio "atlas-data/storage/minio"
)

type UI struct{}

func (UI) Name() string        { return "UI" }
func (UI) ArchiveName() string { return "UI.wz" }

// Run extracts world-icon canvases from UI.wz Login.img/ViewAllChar/WorldIcons
// and uploads them to MinIO. UI.wz has no Postgres-side documents; the
// archive's role in atlas-data is purely to feed MinIO assets (world icons,
// monster gauge metadata, etc.). Monster gauge data is consumed inline by the
// Mob worker via monster.InitGauge — not by this worker.
//
// STATED LIMITATION: The full UI.wz extraction (UIWindow gauges, login
// decorations, character creation chrome) is significantly larger than world
// icons alone. This worker emits only the world-icon subset, which is the
// asset class actually consumed by the cross-cluster channel selector.
func (UI) Run(ctx context.Context, l logrus.FieldLogger, db *gorm.DB, mc *minio.Client, file *wz.File, p Params) error {
	ctx, _, err := withTenant(ctx, p)
	if err != nil {
		return err
	}
	root := file.Root()
	if root == nil {
		return nil
	}
	var loginProps []property.Property
	for _, img := range root.Images() {
		if strings.EqualFold(img.Name(), "Login") {
			props, err := img.Properties()
			if err != nil {
				return fmt.Errorf("ui worker: parse Login.img: %w", err)
			}
			loginProps = props
			break
		}
	}
	if loginProps == nil {
		l.Debugf("UI.wz has no Login.img; skipping world icon extraction")
		return nil
	}
	worldIcons := findWorldIconCanvases(loginProps)
	if len(worldIcons) == 0 {
		l.Debugf("UI.wz Login.img/ViewAllChar/WorldIcons absent or empty")
		return nil
	}
	prefix := minioAssetPrefix(p)
	for worldId, cp := range worldIcons {
		data, err := file.ReadCanvasData(cp.DataOffset(), cp.DataSize())
		if err != nil {
			l.WithError(err).Warnf("read world icon canvas %s", worldId)
			continue
		}
		img, err := canvas.Decompress(data, cp.Width(), cp.Height(), cp.Format(), file.CanvasEncryptionKey())
		if err != nil {
			l.WithError(err).Warnf("decompress world icon %s", worldId)
			continue
		}
		key := fmt.Sprintf("%s/world/%s/icon.png", prefix, worldId)
		if err := putPNG(ctx, mc, key, img); err != nil {
			l.WithError(err).Warnf("upload world icon %s", worldId)
		}
	}
	return nil
}

// findWorldIconCanvases walks Login.img/ViewAllChar/WorldIcons and returns
// each child canvas keyed by its normalized name (world id).
func findWorldIconCanvases(loginProps []property.Property) map[string]*property.CanvasProperty {
	viewAllChar := findSub(loginProps, "ViewAllChar")
	if viewAllChar == nil {
		return nil
	}
	worldIcons := findSub(viewAllChar.Children(), "WorldIcons")
	if worldIcons == nil {
		return nil
	}
	out := make(map[string]*property.CanvasProperty)
	for _, child := range worldIcons.Children() {
		cp, ok := child.(*property.CanvasProperty)
		if !ok {
			continue
		}
		out[cp.Name()] = cp
	}
	return out
}

