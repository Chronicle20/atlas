package mapr

import (
	"bytes"
	"context"
	"fmt"
	"image/png"
	"io"
	"net/http"
	"strconv"
	"time"

	"atlas-renders/storage"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// Handler serves the two map render paths:
//
//   - kind=minimap → 302 to /api/assets/.../minimap.png (atlas-ingress serves
//     the minimap PNG directly from MinIO via the existing assets route).
//   - kind=render → probe the renders bucket for a cached composite; on miss,
//     load layout.json + per-layer PNGs from atlas-assets, stack them in
//     layout.zmap order, encode PNG, best-effort PUT to the cache, stream
//     to the client.
//
// The route is declared in main.go as
// GET /api/wz/map/render/{tenant}/{region}/{version}/{mapId}/{kind}.png.
// Tenant middleware in main.go injects tenant.Model into context.
func Handler(l logrus.FieldLogger, s *storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s == nil {
			http.Error(w, "storage unavailable", http.StatusServiceUnavailable)
			return
		}

		vars := mux.Vars(r)
		kind := vars["kind"]
		mapIDStr := vars["mapId"]

		mapID64, err := strconv.ParseUint(mapIDStr, 10, 32)
		if err != nil {
			http.Error(w, "invalid mapId", http.StatusBadRequest)
			return
		}
		mapID := uint32(mapID64)

		t := tenant.MustFromContext(r.Context())
		version := fmt.Sprintf("%d.%d", t.MajorVersion(), t.MinorVersion())
		tenantID := t.Id().String()
		region := t.Region()

		switch kind {
		case "minimap":
			target := fmt.Sprintf("/api/assets/%s/%s/%s/map/%d/minimap.png",
				tenantID, region, version, mapID)
			http.Redirect(w, r, target, http.StatusFound)
			return
		case "render":
			serveRender(l, s, w, r, tenantID, region, version, mapID)
			return
		default:
			http.Error(w, "invalid kind; expected minimap|render", http.StatusBadRequest)
			return
		}
	}
}

func serveRender(l logrus.FieldLogger, s *storage.Storage, w http.ResponseWriter, r *http.Request, tenantID, region, version string, mapID uint32) {
	renderKey := fmt.Sprintf("tenants/%s/regions/%s/versions/%s/map/%d/render.png",
		tenantID, region, version, mapID)

	// 1) Try the cached render first.
	if rc, err := s.MC.Get(r.Context(), s.Cfg.BucketRenders, renderKey); err == nil {
		defer rc.Close()
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=86400, immutable")
		if _, copyErr := io.Copy(w, rc); copyErr != nil {
			l.WithError(copyErr).Debug("render cache stream interrupted")
		}
		return
	}

	// 2) Composite on miss. Resolve scope against the assets bucket so we
	//    look in tenants/<id>/ when overrides exist, shared/ otherwise.
	scope, err := s.ResolveScope(r.Context(), tenantID, region, version, fmt.Sprintf("map/%d", mapID))
	if err != nil {
		l.WithError(err).Warn("resolve scope failed")
		http.Error(w, "resolve scope: "+err.Error(), http.StatusInternalServerError)
		return
	}

	mapEntry, err := s.GetMap(r.Context(), scope, region, version, mapID)
	if err != nil {
		l.WithError(err).Warn("get map data failed")
		http.Error(w, "map data not found", http.StatusNotFound)
		return
	}

	img, err := Composite(l, mapEntry)
	if err != nil {
		l.WithError(err).Warn("map composite failed")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		l.WithError(err).Warn("png encode failed")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 3) Best-effort PUT to the renders bucket so the next request is a
	//    straight stream. Uses a fresh context so client cancellation does
	//    not abort the cache write.
	body := buf.Bytes()
	go func(payload []byte) {
		putCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := s.MC.Put(putCtx, s.Cfg.BucketRenders, renderKey, bytes.NewReader(payload), int64(len(payload)), "image/png"); err != nil {
			l.WithError(err).Debug("render cache put failed")
		}
	}(body)

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=86400, immutable")
	if _, err := w.Write(body); err != nil {
		l.WithError(err).Debug("render write failed")
	}
}
