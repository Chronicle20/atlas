package world

import (
	"atlas-world/rest"
	"errors"
	"net/http"

	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
)

const (
	getWorlds = "get_worlds"
	getWorld  = "get_world"
)

func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		registerGet := rest.RegisterHandler(l)(si)

		r := router.PathPrefix("/worlds").Subrouter()
		r.HandleFunc("/", registerGet(getWorlds, handleGetWorlds)).Methods(http.MethodGet)
		r.HandleFunc("/{worldId}", registerGet(getWorld, handleGetWorld)).Methods(http.MethodGet)
	}
}

func decoratorsFromInclude(r *http.Request, p Processor) []model.Decorator[Model] {
	var decorators = make([]model.Decorator[Model], 0)
	include := r.URL.Query().Get("include")
	if include == "channels" {
		decorators = append(decorators, p.ChannelDecorator)
	}
	return decorators
}

func handleGetWorld(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseWorldId(d.Logger(), func(worldId world.Id) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			p := NewProcessor(d.Logger(), d.Context())
			ws, err := p.GetWorld(decoratorsFromInclude(r, p)...)(worldId)
			if err != nil {
				if errors.Is(err, errWorldNotFound) {
					w.WriteHeader(http.StatusNotFound)
					return
				}

				d.Logger().WithError(err).Errorf("Unable to get all channel servers for world.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// Transform world to REST model
			rm, err := model.Map(Transform)(model.FixedProvider(ws))()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating world REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
		}
	})
}

func handleGetWorlds(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := NewProcessor(d.Logger(), d.Context())
		ws, err := p.GetWorlds(decoratorsFromInclude(r, p)...)
		if err != nil {
			d.Logger().WithError(err).Errorf("Unable to get all worlds.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Transform worlds to REST models
		rms, err := model.SliceMap(Transform)(model.FixedProvider(ws))(model.ParallelMap())()
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating world REST models.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		query := r.URL.Query()
		queryParams := jsonapi.ParseQueryFields(&query)
		server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rms)
	}
}
