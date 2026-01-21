package blocked

import (
	"atlas-portals/rest"
	"net/http"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
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
			t := tenant.MustFromContext(d.Context())

			// Get blocked portals from cache
			blockedPortals := GetCache().GetForCharacter(t.Id(), characterId)

			d.Logger().Debugf("Found [%d] blocked portals for character [%d].", len(blockedPortals), characterId)

			// Transform to REST models
			rms, err := model.SliceMap(Transform)(model.FixedProvider(blockedPortals))(model.ParallelMap())()
			if err != nil {
				d.Logger().WithError(err).Error("Failed to transform blocked portals")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// Marshal response
			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rms)
		}
	})
}
