package blocked

import (
	"atlas-portals/rest"
	"net/http"
	"sort"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
)

// InitResource registers the routes with the router
func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(r *mux.Router, l logrus.FieldLogger) {
		blockedRouter := r.PathPrefix("/portals/blocked").Subrouter()
		blockedRouter.HandleFunc("", rest.RegisterHandler(l)(si)("get_blocked_portals", handleGetBlockedPortals)).Methods(http.MethodGet).Queries("characterId", "{characterId}")
		blockedRouter.HandleFunc("", rest.RegisterHandler(l)(si)("get_blocked_portals", handleGetBlockedPortals)).Methods(http.MethodGet)
	}
}

// handleGetBlockedPortals returns a handler for the GET /portals/blocked endpoint
func handleGetBlockedPortals(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			page, err := paginate.ParseParams(r.URL.Query(), paginate.DefaultPageSize, paginate.MaxPageSize)
			if err != nil {
				server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
				return
			}

			// Get blocked portals from registry
			blockedPortals := GetRegistry().GetForCharacter(d.Context(), characterId)

			d.Logger().Debugf("Found [%d] blocked portals for character [%d].", len(blockedPortals), characterId)

			sorted := make([]Model, len(blockedPortals))
			copy(sorted, blockedPortals)
			sort.Slice(sorted, func(i, j int) bool {
				if sorted[i].MapId() != sorted[j].MapId() {
					return sorted[i].MapId() < sorted[j].MapId()
				}
				return sorted[i].PortalId() < sorted[j].PortalId()
			})
			paged := paginate.Slice(sorted, page)

			// Transform to REST models
			rms, err := model.SliceMap(Transform)(model.FixedProvider(paged.Items))(model.ParallelMap())()
			if err != nil {
				d.Logger().WithError(err).Error("Failed to transform blocked portals")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// Marshal response
			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalPaginatedResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rms, paginate.EnvelopeFor(paged), r)
		}
	})
}
