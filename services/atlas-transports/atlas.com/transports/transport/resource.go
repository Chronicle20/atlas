package transport

import (
	"atlas-transports/rest"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"net/http"
)

// InitResource registers the transport routes with the router
func InitResource(si jsonapi.ServerInformation) func(p *ProcessorImpl) server.RouteInitializer {
	return func(p *ProcessorImpl) server.RouteInitializer {
		return func(r *mux.Router, l logrus.FieldLogger) {
			registerHandler := rest.RegisterHandler(l)(si)
			r.HandleFunc("/routes/{id}", registerHandler("get_route", GetRouteHandler(p))).Methods(http.MethodGet)
			r.HandleFunc("/routes/{id}/state", registerHandler("get_route_state", GetRouteStateHandler(p))).Methods(http.MethodGet)
			r.HandleFunc("/routes/{id}/schedule", registerHandler("get_route_schedule", GetRouteScheduleHandler(p))).Methods(http.MethodGet)
		}
	}

}

// GetRouteHandler returns a handler for the GET /routes/:id endpoint
func GetRouteHandler(processor *ProcessorImpl) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Extract route ID from URL
			vars := mux.Vars(r)
			routeID := vars["id"]

			// Get route from processor
			routeProvider := processor.ByIdProvider(routeID)
			route, err := routeProvider()
			if err != nil {
				d.Logger().WithError(err).Errorln("Error getting route")
				w.WriteHeader(http.StatusNotFound)
				return
			}

			// Transform route to REST model
			restModel, err := Transform(route)
			if err != nil {
				d.Logger().WithError(err).Errorln("Error transforming route")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// Marshal response
			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(restModel)
		}
	}
}

// GetRouteStateHandler returns a handler for the GET /routes/:id/state endpoint
func GetRouteStateHandler(processor *ProcessorImpl) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Extract route ID from URL
			vars := mux.Vars(r)
			routeID := vars["id"]

			// Get route state from processor
			stateProvider := processor.RouteStateByIdProvider(routeID)
			state, err := stateProvider()
			if err != nil {
				d.Logger().WithError(err).Errorln("Error getting route state")
				w.WriteHeader(http.StatusNotFound)
				return
			}

			// Transform state to REST model
			restModel, err := TransformState(state)
			if err != nil {
				d.Logger().WithError(err).Errorln("Error transforming route state")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// Marshal response
			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[RouteStateRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(restModel)
		}
	}
}

// GetRouteScheduleHandler returns a handler for the GET /routes/:id/schedule endpoint
func GetRouteScheduleHandler(processor *ProcessorImpl) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Extract route ID from URL
			vars := mux.Vars(r)
			routeID := vars["id"]

			// Get route schedule from processor
			scheduleProvider := processor.RouteScheduleByIdProvider(routeID)
			schedule, err := scheduleProvider()
			if err != nil {
				d.Logger().WithError(err).Errorln("Error getting route schedule")
				w.WriteHeader(http.StatusNotFound)
				return
			}

			// Transform schedule to REST models
			restModels := make([]TripScheduleRestModel, 0, len(schedule))
			for _, trip := range schedule {
				restModel, err := TransformSchedule(trip)
				if err != nil {
					d.Logger().WithError(err).Errorln("Error transforming trip schedule")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				restModels = append(restModels, restModel)
			}

			// Marshal response
			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[[]TripScheduleRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(restModels)
		}
	}
}
