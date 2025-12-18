package transport

import (
	"atlas-transports/rest"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"net/http"
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
func GetAllRoutesHandler(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rm, err := model.SliceMap(Transform)(NewProcessor(d.Logger(), d.Context()).AllRoutesProvider())(model.ParallelMap())()
		if err != nil {
			d.Logger().WithError(err).Errorln("Error retrieving routes")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Marshal response
		query := r.URL.Query()
		queryParams := jsonapi.ParseQueryFields(&query)
		server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
	}
}
