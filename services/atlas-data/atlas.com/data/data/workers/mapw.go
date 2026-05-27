package workers

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/Chronicle20/atlas/libs/atlas-wz/mapimage"
	"github.com/Chronicle20/atlas/libs/atlas-wz/maplayout"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	_map "atlas-data/map"
	"atlas-data/npc"
	minio "atlas-data/storage/minio"
)

type Map struct{}

func (Map) Name() string        { return "MAP" }
func (Map) ArchiveName() string { return "Map.wz" }

func (Map) Run(ctx context.Context, l logrus.FieldLogger, db *gorm.DB, mc *minio.Client, file *wz.File, p Params) error {
	ctx, t, err := withTenant(ctx, p)
	if err != nil {
		return err
	}
	root, err := serializeArchive(l, p, file)
	if err != nil {
		return fmt.Errorf("serialize Map.wz: %w", err)
	}
	if _, err := fetchAndSerializeArchive(ctx, l, mc, p, "String.wz"); err != nil {
		l.WithError(err).Warnf("String.wz unavailable; map/npc names will be empty")
	} else {
		if err := _map.InitString(t, filepath.Join(root, "String.wz", "Map.img.xml")); err != nil {
			l.WithError(err).Warnf("map.InitString failed")
		}
		defer func() { _ = _map.GetMapStringRegistry().Clear(t) }()
		if err := npc.InitString(t, filepath.Join(root, "String.wz", "Npc.img.xml")); err != nil {
			l.WithError(err).Warnf("npc.InitString failed (needed by map reader)")
		}
		// Don't clear NPC registry; the NPC worker may still need it.
	}

	// Map registrations live under Map.wz/Map/Map<digit>/<id>.img.xml.
	mapDir := filepath.Join(root, "Map.wz", "Map")
	if err := registerAllInDirectory(l, ctx, mapDir, _map.RegisterMap(db)); err != nil {
		return err
	}

	// Emit per-map layout JSON + minimap PNG to MinIO. Per-layer composites
	// are NOT produced here anymore — atlas-renders fetches Map.wz from MinIO
	// and composites on first request (see docs/tasks/task-071-.../lazy-map-render.md).
	// Dropping the layer composite pass takes Map worker wall-clock from
	// ~30 min to ~2-5 min on a v83 GMS dataset (PR-544 evidence).
	prefix := minioAssetPrefix(p)
	idx := mapimage.NewIndex(file)
	var scanned, layoutsWritten, minimapsWritten, extractLayoutErrs, extractMinimapErrs int
	for _, img := range idx.Maps() {
		mapId, ok := imgID(img.Name())
		if !ok {
			continue
		}
		scanned++
		layout, err := mapimage.ExtractLayout(img)
		if err != nil {
			extractLayoutErrs++
			l.WithError(err).Debugf("extract layout map %d", mapId)
		} else {
			data, mErr := maplayout.Marshal(layout)
			if mErr != nil {
				l.WithError(mErr).Warnf("marshal layout map %d", mapId)
			} else {
				key := fmt.Sprintf("%s/map/%d/layout.json", prefix, mapId)
				if err := putJSON(ctx, mc, key, data); err != nil {
					l.WithError(err).Warnf("upload layout map %d", mapId)
				} else {
					layoutsWritten++
				}
			}
		}
		mm, err := mapimage.ExtractMinimap(img)
		if err != nil {
			if !errors.Is(err, mapimage.ErrNoMinimap) {
				extractMinimapErrs++
				l.WithError(err).Debugf("extract minimap map %d", mapId)
			}
			continue
		}
		key := fmt.Sprintf("%s/map/%d/minimap.png", prefix, mapId)
		if err := putPNG(ctx, mc, key, mm); err != nil {
			l.WithError(err).Warnf("upload minimap map %d", mapId)
			continue
		}
		minimapsWritten++
	}
	l.Infof("map assets: scanned=%d layouts=%d minimaps=%d extractLayoutErrs=%d extractMinimapErrs=%d",
		scanned, layoutsWritten, minimapsWritten, extractLayoutErrs, extractMinimapErrs)
	return nil
}
