package character

import (
	"atlas-buffs/buff"
	"atlas-buffs/rest"
	"errors"
	"net/http"
	"sort"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
)

func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		registerGet := rest.RegisterHandler(l)(si)
		r := router.PathPrefix("/characters").Subrouter()
		r.HandleFunc("/{characterId}/buffs", registerGet("get_character_buffs", handleGetBuffs)).Methods(http.MethodGet)
	}
}

// handleGetBuffs is registry-backed (character.GetRegistry() — a live
// Redis-held TTL cache of active buff state, not a DB table), so
// database.PagedQuery does not apply; paginate.Slice is used per the
// task-117 registry-list convention. cm.Buffs() returns a
// map[string]buff.Model keyed by a composite "<sourceId>" or
// "<sourceId>:<statType>" string (see character.Model.buffs doc comment) —
// that key is guaranteed unique per stored buff (accumulate-mode buffs
// share a SourceId but not a map key), and Go map iteration order is
// explicitly randomized, so the key is used as the stable sort field
// before slicing rather than SourceId (not unique) or insertion order (not
// deterministic). buff.Transform is a pure field copy — CreatedAt/
// ExpiresAt are the raw stored timestamps, not a computed
// remaining-duration — so no extra decoration stage is needed beyond the
// registry read itself.
func handleGetBuffs(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			page, err := paginate.ParseParams(r.URL.Query(), paginate.MaxPageSize, paginate.MaxPageSize)
			if err != nil {
				server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
				return
			}

			cm, err := NewProcessor(d.Logger(), d.Context()).GetById(characterId)
			if errors.Is(err, ErrNotFound) {
				w.WriteHeader(http.StatusNotFound)
				return
			}

			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			buffsByKey := cm.Buffs()
			keys := make([]string, 0, len(buffsByKey))
			for k := range buffsByKey {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			sorted := make([]buff.Model, 0, len(keys))
			for _, k := range keys {
				sorted = append(sorted, buffsByKey[k])
			}

			paged := paginate.Slice(sorted, page)

			res, err := model.SliceMap(buff.Transform)(model.FixedProvider(paged.Items))(model.ParallelMap())()
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalPaginatedResponse[[]buff.RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res, paginate.EnvelopeFor(paged), r)
		}
	})
}
