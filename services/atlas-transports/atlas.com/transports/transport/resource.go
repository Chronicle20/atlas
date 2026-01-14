package transport

import (
	"atlas-transports/rest"
	map2 "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"net/http"
	"strconv"
)

// InitResource registers the transport routes with the router
func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(r *mux.Router, l logrus.FieldLogger) {
		registerHandler := rest.RegisterHandler(l)(si)
		r.HandleFunc("/transports/routes", registerHandler("get_all_routes", GetAllRoutesHandler)).Methods(http.MethodGet)
		r.HandleFunc("/transports/routes/{routeId}", registerHandler("get_route", GetRouteHandler)).Methods(http.MethodGet)
	}
}

// GetRouteHandler returns a handler for the GET /transports/routes/:id endpoint
func GetRouteHandler(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseRouteId(d.Logger(), func(routeId uuid.UUID) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			rm, err := model.Map(Transform)(NewProcessor(d.Logger(), d.Context()).ByIdProvider(routeId))()
			if err != nil {
				d.Logger().WithError(err).Errorln("Error retrieving route")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// Marshal response
			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
		}
	})
}

// GetAllRoutesHandler returns a handler for the GET /transports/routes endpoint
// Supports optional filter[startMapId] query parameter for filtering by start map
func GetAllRoutesHandler(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		startMapIdFilter := query.Get("filter[startMapId]")

		var rm []RestModel
		var err error

		if startMapIdFilter != "" {
			// Parse the start map ID from the query parameter
			mapId, parseErr := strconv.ParseUint(startMapIdFilter, 10, 32)
			if parseErr != nil {
				d.Logger().WithError(parseErr).Errorf("Invalid filter[startMapId] parameter: %s", startMapIdFilter)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			// Get route by start map ID
			processor := NewProcessor(d.Logger(), d.Context())
			route, err := processor.GetByStartMap(map2.Id(mapId))
			if err != nil {
				// Route not found - return empty array (RESTful collection behavior)
				d.Logger().WithError(err).Debugf("No route found for start map %d", mapId)
				rm = []RestModel{}
			} else {
				// Transform single route to REST model and wrap in array
				restModel, transformErr := Transform(route)
			if transformErr != nil {
				d.Logger().WithError(transformErr).Errorf("Error transforming route for start map %d", mapId)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
				rm = []RestModel{restModel}
			}
		} else {
			// No filter - return all routes (existing behavior)
			rm, err = model.SliceMap(Transform)(NewProcessor(d.Logger(), d.Context()).AllRoutesProvider())(model.ParallelMap())()
			if err != nil {
				d.Logger().WithError(err).Errorln("Error retrieving routes")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}

		// Marshal response
		queryParams := jsonapi.ParseQueryFields(&query)
		server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
	}
}
