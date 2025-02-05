package _map

import (
	"atlas-drops/drop"
	"atlas-drops/rest"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/manyminds/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"net/http"
)

func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		registerGet := rest.RegisterHandler(l)(si)
		r := router.PathPrefix("/worlds/{worldId}/channels/{channelId}/maps/{mapId}/drops").Subrouter()
		r.HandleFunc("", registerGet("get_drops_in_map", handleGetDropsInMap)).Methods(http.MethodGet)
	}
}

func handleGetDropsInMap(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseWorldId(d.Logger(), func(worldId byte) http.HandlerFunc {
		return rest.ParseChannelId(d.Logger(), func(channelId byte) http.HandlerFunc {
			return rest.ParseMapId(d.Logger(), func(mapId uint32) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					ds, err := drop.GetForMap(d.Logger())(d.Context())(worldId, channelId, mapId)
					if err != nil {
						w.WriteHeader(http.StatusNotFound)
						return
					}

					res, err := model.SliceMap(drop.Transform)(model.FixedProvider(ds))()()
					if err != nil {
						d.Logger().WithError(err).Errorf("Creating REST model.")
						w.WriteHeader(http.StatusInternalServerError)
						return
					}

					server.Marshal[[]drop.RestModel](d.Logger())(w)(c.ServerInformation())(res)
				}
			})
		})
	})
}
