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

// Run extracts UI.wz canvases and uploads them to MinIO: the world-icon set
// (Login.img/ViewAllChar/WorldIcons) consumed by the cross-cluster channel
// selector, and the item-protector padlock (UIWindow.img/ItemProtector/Icon)
// reused by atlas-ui as the sealed-item badge. UI.wz has no Postgres-side
// documents; the archive's role in atlas-data is purely to feed MinIO assets.
// Monster gauge data is consumed inline by the Mob worker via monster.InitGauge
// — not by this worker.
//
// STATED LIMITATION: The full UI.wz extraction (UIWindow gauges, login
// decorations, character creation chrome) is significantly larger than the
// subset emitted here. This worker emits only the assets actually consumed
// downstream (world icons, the item-protector overlay).
func (UI) Run(ctx context.Context, l logrus.FieldLogger, db *gorm.DB, mc *minio.Client, file *wz.File, p Params) error {
	ctx, _, err := withTenant(ctx, p)
	if err != nil {
		return err
	}
	root := file.Root()
	if root == nil {
		return nil
	}
	var loginProps, uiWindowProps []property.Property
	for _, img := range root.Images() {
		switch {
		case strings.EqualFold(img.Name(), "Login"):
			props, err := img.Properties()
			if err != nil {
				return fmt.Errorf("ui worker: parse Login.img: %w", err)
			}
			loginProps = props
		case strings.EqualFold(img.Name(), "UIWindow"):
			props, err := img.Properties()
			if err != nil {
				return fmt.Errorf("ui worker: parse UIWindow.img: %w", err)
			}
			uiWindowProps = props
		}
	}

	prefix := minioAssetPrefix(p)

	// World icons: Login.img/ViewAllChar/WorldIcons.
	if loginProps == nil {
		l.Debugf("UI.wz has no Login.img; skipping world icon extraction")
	} else if worldIcons := findWorldIconCanvases(loginProps); len(worldIcons) == 0 {
		l.Debugf("UI.wz Login.img/ViewAllChar/WorldIcons absent or empty")
	} else {
		for worldId, cp := range worldIcons {
			data, err := file.ReadCanvasData(cp.DataOffset(), cp.DataSize())
			if err != nil {
				l.WithError(err).Warnf("read world icon canvas %s", worldId)
				continue
			}
			img, err := canvas.Decompress(data, cp.Width(), cp.Height(), cp.Format(), file.CanvasEncryptionKeyFor(cp.DataOffset()))
			if err != nil {
				l.WithError(err).Warnf("decompress world icon %s", worldId)
				continue
			}
			key := fmt.Sprintf("%s/world-icon/%s/icon.png", prefix, worldId)
			if err := putPNG(ctx, mc, key, img); err != nil {
				l.WithError(err).Warnf("upload world icon %s", worldId)
			}
		}
	}

	// Sealed-item lock overlay: UIWindow.img/ItemProtector/Icon. This is the
	// padlock the client draws on a sealed item; atlas-ui reuses it as the seal
	// badge (category `ui/item-protector`). One icon serves all lock durations.
	if uiWindowProps == nil {
		l.Debugf("UI.wz has no UIWindow.img; skipping item-protector icon extraction")
	} else if cp := findItemProtectorIcon(uiWindowProps); cp == nil {
		l.Debugf("UI.wz UIWindow.img/ItemProtector/Icon absent")
	} else if data, err := file.ReadCanvasData(cp.DataOffset(), cp.DataSize()); err != nil {
		l.WithError(err).Warnf("read item-protector icon canvas")
	} else if img, err := canvas.Decompress(data, cp.Width(), cp.Height(), cp.Format(), file.CanvasEncryptionKeyFor(cp.DataOffset())); err != nil {
		l.WithError(err).Warnf("decompress item-protector icon")
	} else {
		key := fmt.Sprintf("%s/ui/item-protector/icon.png", prefix)
		if err := putPNG(ctx, mc, key, img); err != nil {
			l.WithError(err).Warnf("upload item-protector icon")
		}
	}

	return nil
}

// findItemProtectorIcon walks UIWindow.img/ItemProtector and returns its "Icon"
// canvas (the padlock the client overlays on a sealed item). Handles both the
// direct-canvas layout (ItemProtector/Icon is itself a canvas) and the nested
// layout (ItemProtector/Icon is a container whose first canvas child is the art).
func findItemProtectorIcon(uiWindowProps []property.Property) *property.CanvasProperty {
	ip := findSub(uiWindowProps, "ItemProtector")
	if ip == nil {
		return nil
	}
	for _, child := range ip.Children() {
		if cp, ok := child.(*property.CanvasProperty); ok && strings.EqualFold(cp.Name(), "Icon") {
			return cp
		}
	}
	if iconSub := findSub(ip.Children(), "Icon"); iconSub != nil {
		for _, child := range iconSub.Children() {
			if cp, ok := child.(*property.CanvasProperty); ok {
				return cp
			}
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

