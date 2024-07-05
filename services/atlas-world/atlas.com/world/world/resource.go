package world

import (
	"atlas-world/rest"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/manyminds/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"net/http"
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

func handleGetWorld(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseWorldId(d.Logger(), func(worldId byte) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ws, err := GetWorld(d.Logger(), c.Tenant())(worldId)
			if err != nil {
				if errors.Is(err, errWorldNotFound) {
					w.WriteHeader(http.StatusNotFound)
					return
				}

				d.Logger().WithError(err).Errorf("Unable to get all channel servers for world.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			res, err := model.Transform(ws, Transform)
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			server.Marshal[RestModel](d.Logger())(w)(c.ServerInformation())(res)
		}
	})
}

func handleGetWorlds(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ws, err := GetWorlds(d.Logger(), c.Tenant())
		if err != nil {
			d.Logger().WithError(err).Errorf("Unable to get all channel servers.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		res, err := model.TransformAll(ws, Transform)
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating REST model.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		server.Marshal[[]RestModel](d.Logger())(w)(c.ServerInformation())(res)
	}
}
