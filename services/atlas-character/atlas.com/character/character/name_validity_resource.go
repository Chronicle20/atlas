package character

import (
	"atlas-character/rest"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

type NameValidityResponse struct {
	Valid  bool   `json:"valid"`
	Reason string `json:"reason,omitempty"`
	Detail string `json:"detail,omitempty"`
}

func handleGetNameValidity(d *rest.HandlerDependency, _ *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		name := q.Get("name")
		widRaw := q.Get("worldId")
		if name == "" || widRaw == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		wid, err := strconv.ParseUint(widRaw, 10, 8)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		res, err := NewProcessor(d.Logger(), d.Context(), d.DB()).CheckNameValidity(name, world.Id(wid))
		if err != nil {
			d.Logger().WithError(err).Error("name-validity check failed")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(NameValidityResponse{
			Valid:   res.Valid,
			Reason:  res.Reason,
			Detail:  res.Detail,
		})
	}
}
