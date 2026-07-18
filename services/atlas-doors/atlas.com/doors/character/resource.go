package character

import (
	"atlas-doors/door"
	"atlas-doors/rest"
	"net/http"
	"sort"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
)

const (
	getDoorsByOwner = "get_doors_by_owner"
)

// InitResource registers the /characters/{characterId}/doors route, returning
// the live doors owned by the character for the tenant in context.
func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		r := router.PathPrefix("/characters").Subrouter()
		r.HandleFunc("/{characterId}/doors",
			rest.RegisterHandler(l)(si)(getDoorsByOwner, handleGetDoorsByOwner)).Methods(http.MethodGet)
	}
}

func handleGetDoorsByOwner(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId character.Id) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			page, err := paginate.ParseParams(r.URL.Query(), paginate.MaxPageSize, paginate.MaxPageSize)
			if err != nil {
				server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
				return
			}

			p := door.NewProcessor(d.Logger(), d.Context())
			ms, err := p.GetByOwner(characterId)
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to retrieve doors for owner [%d].", characterId)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			sorted := make([]door.Model, len(ms))
			copy(sorted, ms)
			sort.Slice(sorted, func(i, j int) bool { return sorted[i].PairId() < sorted[j].PairId() })
			paged := paginate.Slice(sorted, page)

			res, err := model.SliceMap(door.Transform)(model.FixedProvider(paged.Items))()()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			server.MarshalPaginatedResponse[[]door.RestModel](d.Logger())(w)(c.ServerInformation())(r.URL.Query())(res, paginate.EnvelopeFor(paged), r)
		}
	})
}
