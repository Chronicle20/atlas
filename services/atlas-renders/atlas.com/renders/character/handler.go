package character

import (
	"bytes"
	"context"
	"errors"
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

// Handler is the character composite render endpoint. The route
//
//	GET /api/wz/character/render/{tenant}/{region}/{version}/{hash}.png
//
// resolves a loadout from the query string, verifies the URL hash matches,
// then either streams a cached PNG out of the atlas-renders MinIO bucket or
// composites one on demand and writes it back to the cache.
//
// Ported from services/atlas-wz-extractor/atlas.com/wz-extractor/
// characterrender/handler.go, rewritten to source assets from MinIO via
// storage.Storage rather than the on-disk extract tree.
func Handler(l logrus.FieldLogger, s *storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		if s == nil {
			WriteError(w, http.StatusServiceUnavailable, ErrorBody{
				Code: "storage-unavailable", Title: "MinIO storage not configured",
			})
			return
		}

		t, err := tenant.FromContext(r.Context())()
		if err != nil {
			WriteError(w, http.StatusBadRequest, ErrorBody{
				Code: "tenant-mismatch", Title: "Tenant not present in request context",
				Detail: err.Error(),
			})
			return
		}

		vars := mux.Vars(r)
		urlHash := vars["hash"]
		urlTenant := vars["tenant"]
		urlRegion := vars["region"]
		urlVersion := vars["version"]
		if urlHash == "" || urlTenant == "" || urlRegion == "" || urlVersion == "" {
			WriteError(w, http.StatusBadRequest, ErrorBody{
				Code: "invalid-input", Title: "Missing path component",
			})
			return
		}
		// Verify the URL's tenant/region/version match the request-context
		// tenant. This prevents cross-tenant cache poisoning via crafted URLs.
		if urlTenant != t.Id().String() || urlRegion != t.Region() ||
			urlVersion != fmt.Sprintf("%d.%d", t.MajorVersion(), t.MinorVersion()) {
			WriteError(w, http.StatusBadRequest, ErrorBody{
				Code: "tenant-mismatch", Title: "Path tenant does not match request context",
			})
			return
		}

		q, err := ParseRenderQuery(r.URL.Query())
		if err != nil {
			WriteError(w, http.StatusBadRequest, ErrorBody{
				Code: "invalid-input", Title: "Invalid query", Detail: err.Error(),
			})
			return
		}

		canonical := CanonicalLoadoutString(
			urlTenant, urlRegion, t.MajorVersion(), t.MinorVersion(),
			q.Skin, q.Hair, q.Face, q.Stance, q.Frame, q.Resize, q.Items,
		)
		if expected := LoadoutHash(canonical); expected != urlHash {
			WriteError(w, http.StatusBadRequest, ErrorBody{
				Code: "hash-mismatch", Title: "URL hash does not match query",
				Meta: map[string]any{"expected": expected, "got": urlHash},
			})
			return
		}

		// 1) Try the cached render in atlas-renders bucket. The key shape
		//    matches design §4.4 and the atlas-ingress nginx rewrite.
		renderKey := fmt.Sprintf("tenants/%s/regions/%s/versions/%s/character/%s.png",
			urlTenant, urlRegion, urlVersion, urlHash)
		if rc, err := s.MC.Get(r.Context(), s.Cfg.BucketRenders, renderKey); err == nil {
			defer rc.Close()
			w.Header().Set("Content-Type", "image/png")
			w.Header().Set("Cache-Control", "public, max-age=86400, immutable")
			w.Header().Set("ETag", "\""+urlHash+"\"")
			w.Header().Set("X-Render-Cache", "hit")
			if _, copyErr := io.Copy(w, rc); copyErr != nil {
				l.WithError(copyErr).Debug("cached render copy failed")
			}
			return
		}

		// 2) Cache miss → composite from scratch.
		img, resolvedStance, twoHandedOverride, cerr := Composite(r.Context(), l, s, t, q)
		if cerr != nil {
			writeCompositorError(w, l, cerr)
			return
		}
		_ = resolvedStance
		_ = twoHandedOverride

		// Optional integer-multiple upscale per the donor's `resize` param.
		if q.Resize > 1 {
			img = NearestNeighborUpscale(img, q.Resize)
		}

		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			WriteError(w, http.StatusInternalServerError, ErrorBody{
				Code: "compositor-error", Title: "PNG encode failed", Detail: err.Error(),
			})
			return
		}

		// 3) Best-effort PUT to the renders bucket so the next identical
		//    request short-circuits in the atlas-ingress probe. We pass a
		//    fresh background context because the client request context may
		//    be canceled the moment we finish writing the response.
		payload := buf.Bytes()
		go func() {
			putCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := s.MC.Put(putCtx, s.Cfg.BucketRenders, renderKey,
				bytes.NewReader(payload), int64(len(payload)), "image/png"); err != nil {
				l.WithError(err).Warn("best-effort render PUT failed")
			}
		}()

		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=86400, immutable")
		w.Header().Set("ETag", "\""+urlHash+"\"")
		w.Header().Set("X-Render-Cache", "miss")
		w.Header().Set("X-Render-Ms", strconv.FormatInt(time.Since(start).Milliseconds(), 10))
		if _, err := w.Write(payload); err != nil {
			l.WithError(err).Debug("response write failed")
		}
	}
}

// writeCompositorError maps a Composite error onto the donor's status-code
// envelope so the API surface matches characterrender exactly.
func writeCompositorError(w http.ResponseWriter, l logrus.FieldLogger, err error) {
	switch {
	case errors.Is(err, ErrInvalidStance):
		WriteError(w, http.StatusBadRequest, ErrorBody{
			Code: "invalid-stance", Title: "Unknown stance",
			Meta:   map[string]any{"supported": SupportedStances()},
			Detail: err.Error(),
		})
	case errors.Is(err, ErrUnknownSkin):
		WriteError(w, http.StatusBadRequest, ErrorBody{
			Code: "invalid-skin", Title: "Skin id out of range", Detail: err.Error(),
		})
	case errors.Is(err, ErrFrameOutOfRange):
		WriteError(w, http.StatusBadRequest, ErrorBody{
			Code: "frame-out-of-range", Title: "Frame index out of range",
			Detail: err.Error(),
		})
	case errors.Is(err, ErrAssetMissing):
		WriteError(w, http.StatusNotFound, ErrorBody{
			Code: "missing-asset", Title: "Required sprite missing from extract",
			Detail: err.Error(),
		})
	default:
		l.WithError(err).Error("compositor error")
		WriteError(w, http.StatusInternalServerError, ErrorBody{
			Code: "compositor-error", Title: "Compositor failed", Detail: err.Error(),
		})
	}
}
