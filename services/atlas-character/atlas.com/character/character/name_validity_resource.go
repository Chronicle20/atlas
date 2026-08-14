package character

import (
	"atlas-character/rest"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

type NameValidityResponse struct {
	Valid    bool   `json:"valid"`
	Reason   string `json:"reason,omitempty"`
	Detail   string `json:"detail,omitempty"`
	Reserved bool   `json:"reserved,omitempty"`
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

		// scope defaults to WORLD so the existing atlas-character-factory
		// client (creation) is unaffected by this endpoint gaining a new
		// query parameter. Any other value is a client error.
		scope := NameScope(q.Get("scope"))
		if scope == "" {
			scope = NameScopeWorld
		}
		if scope != NameScopeWorld && scope != NameScopeTenant {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		res, err := NewProcessor(d.Logger(), d.Context(), d.DB()).CheckNameValidity(name, world.Id(wid), scope)
		if err != nil {
			d.Logger().WithError(err).Error("name-validity check failed")
			server.WriteErrorResponse(d.Logger())(w)(err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(NameValidityResponse{
			Valid:    res.Valid,
			Reason:   res.Reason,
			Detail:   res.Detail,
			Reserved: res.Reason == "reserved",
		})
	}
}
