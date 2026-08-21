package jukebox

import (
	"atlas-maps/rest"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

const (
	getJukeboxInMap = "get_jukebox_in_map"
)

func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		r := router.PathPrefix("/worlds").Subrouter()
		r.HandleFunc("/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/jukebox", rest.RegisterHandler(l)(si)(getJukeboxInMap, handleGetJukeboxInMap)).Methods(http.MethodGet)
	}
}

func handleGetJukeboxInMap(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseWorldId(d.Logger(), func(worldId world.Id) http.HandlerFunc {
		return rest.ParseChannelId(d.Logger(), func(channelId channel.Id) http.HandlerFunc {
			return rest.ParseMapId(d.Logger(), func(mapId _map.Id) http.HandlerFunc {
				return rest.ParseInstanceId(d.Logger(), func(instanceId uuid.UUID) http.HandlerFunc {
					return func(w http.ResponseWriter, r *http.Request) {
						f := field.NewBuilder(worldId, channelId, mapId).SetInstance(instanceId).Build()
						jp := NewProcessor(d.Logger(), d.Context())
						entry, ok := jp.GetActive(f)
						if !ok {
							w.WriteHeader(http.StatusNotFound)
							return
						}

						res, err := Transform(entry)
						if err != nil {
							d.Logger().WithError(err).Errorf("Creating REST model.")
							server.WriteErrorResponse(d.Logger())(w)(err)
							return
						}

						server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(r.URL.Query())(res)
					}
				})
			})
		})
	})
}
