package _map

import (
	"atlas-drops/drop"
	"atlas-drops/rest"
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
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
	return rest.ParseWorldId(d.Logger(), func(worldId world.Id) http.HandlerFunc {
		return rest.ParseChannelId(d.Logger(), func(channelId channel.Id) http.HandlerFunc {
			return rest.ParseMapId(d.Logger(), func(mapId _map.Id) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					p := drop.NewProcessor(d.Logger(), d.Context())
					ds, err := p.GetForMap(worldId, channelId, mapId)
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

					query := r.URL.Query()
					queryParams := jsonapi.ParseQueryFields(&query)
					server.MarshalResponse[[]drop.RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
				}
			})
		})
	})
}
