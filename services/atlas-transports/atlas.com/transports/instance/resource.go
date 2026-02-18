package instance

import (
	"atlas-transports/rest"
	"net/http"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
)

func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(r *mux.Router, l logrus.FieldLogger) {
		registerHandler := rest.RegisterHandler(l)(si)
		r.HandleFunc("/transports/instance-routes", registerHandler("get_all_instance_routes", GetAllInstanceRoutesHandler)).Methods(http.MethodGet)
		r.HandleFunc("/transports/instance-routes/{routeId}", registerHandler("get_instance_route", GetInstanceRouteHandler)).Methods(http.MethodGet)
		r.HandleFunc("/transports/instance-routes/{routeId}/status", registerHandler("get_instance_route_status", GetInstanceRouteStatusHandler)).Methods(http.MethodGet)
		r.HandleFunc("/transports/instance-routes/{routeId}/start", rest.RegisterInputHandler[StartTransportRestModel](l)(si)("start_instance_transport", StartInstanceTransportHandler)).Methods(http.MethodPost)
	}
}

func GetAllInstanceRoutesHandler(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := NewProcessor(d.Logger(), d.Context())
		routes := p.GetRoutes()

		rm, err := model.SliceMap(TransformRoute)(model.FixedProvider(routes))(model.ParallelMap())()
		if err != nil {
			d.Logger().WithError(err).Errorln("Error transforming instance routes")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		query := r.URL.Query()
		queryParams := jsonapi.ParseQueryFields(&query)
		server.MarshalResponse[[]RouteRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
	}
}

func GetInstanceRouteHandler(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseRouteId(d.Logger(), func(routeId uuid.UUID) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			p := NewProcessor(d.Logger(), d.Context())
			route, ok := p.GetRoute(routeId)
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}

			rm, err := TransformRoute(route)
			if err != nil {
				d.Logger().WithError(err).Errorln("Error transforming instance route")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[RouteRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
		}
	})
}

func StartInstanceTransportHandler(d *rest.HandlerDependency, c *rest.HandlerContext, input StartTransportRestModel) http.HandlerFunc {
	return rest.ParseRouteId(d.Logger(), func(routeId uuid.UUID) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			f := field.NewBuilder(input.WorldId, input.ChannelId, 0).Build()
			err := NewProcessor(d.Logger(), d.Context()).StartTransportAndEmit(input.CharacterId, routeId, f)
			if err != nil {
				d.Logger().WithError(err).Errorf("Error starting instance transport for character [%d].", input.CharacterId)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		}
	})
}

func GetInstanceRouteStatusHandler(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseRouteId(d.Logger(), func(routeId uuid.UUID) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ir := getInstanceRegistry()
			instances := ir.GetInstancesByRoute(uuid.Nil, routeId)

			statuses := make([]InstanceStatusRestModel, 0)
			for _, inst := range instances {
				s, err := TransformInstanceStatus(inst)
				if err == nil {
					statuses = append(statuses, s)
				}
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[[]InstanceStatusRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(statuses)
		}
	})
}
