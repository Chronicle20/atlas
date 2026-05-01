package characterrender

import (
	"atlas-wz-extractor/characterimage"
	"bytes"
	"errors"
	"image/png"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// Handler holds the dependencies a render handler needs.
type Handler struct {
	AssetsRoot string
	Compositor *characterimage.Compositor
}

// NewHandler returns a fully constructed Handler.
func NewHandler(assetsRoot string, c *characterimage.Compositor) *Handler {
	return &Handler{AssetsRoot: assetsRoot, Compositor: c}
}

// HandleRender is the http.HandlerFunc.
func (h *Handler) HandleRender(l logrus.FieldLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		path, err := ParseRenderPath(mux.Vars(r))
		if err != nil {
			IncrementError("invalid-input")
			WriteError(w, http.StatusBadRequest, ErrorBody{
				Code: "invalid-input", Title: "Invalid path", Detail: err.Error(),
			})
			return
		}
		query, err := ParseRenderQuery(r.URL.Query())
		if err != nil {
			IncrementError("invalid-input")
			WriteError(w, http.StatusBadRequest, ErrorBody{
				Code: "invalid-input", Title: "Invalid query", Detail: err.Error(),
			})
			return
		}

		canonical := CanonicalLoadoutString(
			path.Tenant, path.Region, path.MajorVersion, path.MinorVersion,
			query.Skin, query.Hair, query.Face,
			query.Stance, query.Frame, query.Resize,
			query.Items,
		)
		expected := LoadoutHash(canonical)
		if expected != path.Hash {
			IncrementError("hash-mismatch")
			WriteError(w, http.StatusBadRequest, ErrorBody{
				Code: "hash-mismatch", Title: "URL hash does not match query",
				Meta: map[string]any{"expected": expected, "got": path.Hash},
			})
			return
		}

		assetsRoot := filepath.Join(h.AssetsRoot, path.Tenant, path.Region,
			strconv.FormatUint(uint64(path.MajorVersion), 10)+"."+strconv.FormatUint(uint64(path.MinorVersion), 10))

		req := characterimage.CompositeRequest{
			AssetsRoot: assetsRoot,
			Skin:       query.Skin,
			Hair:       query.Hair,
			Face:       query.Face,
			Equipment:  itemsToSlotMap(query.Items),
			Stance:     query.Stance,
			Frame:      query.Frame,
			Resize:     query.Resize,
			IsMale:     false, // gender selection deferred — see plan note
		}

		res, err := h.Compositor.Composite(req)
		if err != nil {
			h.writeCompositorError(w, l, err)
			return
		}

		var buf bytes.Buffer
		if err := png.Encode(&buf, res.Image); err != nil {
			IncrementError("compositor-error")
			WriteError(w, http.StatusInternalServerError, ErrorBody{
				Code: "compositor-error", Title: "PNG encode failed",
			})
			return
		}

		dst := filepath.Join(assetsRoot, "character", path.Hash+".png")
		if err := AtomicWritePNG(dst, bytes.NewReader(buf.Bytes())); err != nil {
			l.WithError(err).Errorf("atomic write %s", dst)
			IncrementError("compositor-error")
			WriteError(w, http.StatusInternalServerError, ErrorBody{
				Code: "compositor-error", Title: "Failed to persist render",
			})
			return
		}

		IncrementRender(res.ResolvedStance, res.TwoHandedOverride)
		ObserveDurationMs(float64(time.Since(start).Milliseconds()))

		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=86400, immutable")
		w.Header().Set("ETag", "\""+path.Hash+"\"")
		w.Header().Set("X-Render-Cache", "miss")
		w.Header().Set("X-Render-Ms", strconv.FormatInt(time.Since(start).Milliseconds(), 10))
		_, _ = w.Write(buf.Bytes())
	}
}

// itemsToSlotMap converts the sorted item list into a map keyed by a synthetic
// slot id derived from item-classification. The compositor's joint-tree only
// needs the slot grouping for joint resolution; the actual slot indices in
// the request URL are not preserved (they're hidden in the canonical hash).
//
// We assign slots:
//
//	1xxxxxxx item id -> slot derived from id/10000:
//	  100xxxx hat        -> -1
//	  101xxxx face acc   -> -2
//	  102xxxx eye acc    -> -3
//	  103xxxx earrings   -> -4
//	  104xxxx top        -> -5
//	  105xxxx overall    -> -5
//	  106xxxx bottom     -> -6
//	  107xxxx shoes      -> -7
//	  108xxxx gloves     -> -8
//	  109xxxx shield     -> -10
//	  110xxxx-114xxxx cape -> -9
//	  130xxxx-149xxxx weapon -> -11
//
// Items whose classifications fall outside these ranges are silently dropped.
func itemsToSlotMap(items []int) map[int]int {
	out := map[int]int{}
	for _, id := range items {
		slot, ok := slotForItem(id)
		if !ok {
			continue
		}
		out[slot] = id
	}
	return out
}

func slotForItem(id int) (int, bool) {
	c := id / 10000
	switch {
	case c == 100:
		return -1, true
	case c == 101:
		return -2, true
	case c == 102:
		return -3, true
	case c == 103:
		return -4, true
	case c == 104, c == 105:
		return -5, true
	case c == 106:
		return -6, true
	case c == 107:
		return -7, true
	case c == 108:
		return -8, true
	case c == 109:
		return -10, true
	case c >= 110 && c <= 114:
		return -9, true
	case c >= 130 && c <= 149:
		return -11, true
	}
	return 0, false
}

func (h *Handler) writeCompositorError(w http.ResponseWriter, l logrus.FieldLogger, err error) {
	switch {
	case errors.Is(err, characterimage.ErrUnknownTemplateId):
		IncrementError("unknown-template-id")
		WriteError(w, http.StatusBadRequest, ErrorBody{
			Code: "unknown-template-id", Title: "Equipment templateId not present in extract",
			Detail: err.Error(),
		})
	case errors.Is(err, characterimage.ErrInvalidStance):
		IncrementError("invalid-stance")
		WriteError(w, http.StatusBadRequest, ErrorBody{
			Code: "invalid-stance", Title: "Unknown stance",
			Meta: map[string]any{"supported": characterimage.SupportedStances()},
			Detail: err.Error(),
		})
	case errors.Is(err, characterimage.ErrFrameOutOfRange):
		IncrementError("frame-out-of-range")
		WriteError(w, http.StatusBadRequest, ErrorBody{
			Code: "frame-out-of-range", Title: "Frame index out of range",
			Detail: err.Error(),
		})
	case errors.Is(err, characterimage.ErrAssetsMissing):
		IncrementError("missing-asset")
		WriteError(w, http.StatusNotFound, ErrorBody{
			Code: "missing-asset", Title: "Required sprite missing from extract",
			Detail: err.Error(),
		})
	default:
		l.WithError(err).Error("compositor error")
		IncrementError("compositor-error")
		WriteError(w, http.StatusInternalServerError, ErrorBody{
			Code: "compositor-error", Title: "Compositor failed",
		})
	}
}
