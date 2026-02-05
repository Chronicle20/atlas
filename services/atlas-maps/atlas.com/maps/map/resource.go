package _map

import (
	"atlas-maps/rest"
	"net/http"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
)

const (
	getCharactersInMap = "get_characters_in_map"
)

func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		r := router.PathPrefix("/worlds").Subrouter()
		r.HandleFunc("/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/characters", rest.RegisterHandler(l)(si)(getCharactersInMap, handleGetCharactersInMap)).Methods(http.MethodGet)
	}
}

func handleGetCharactersInMap(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseWorldId(d.Logger(), func(worldId world.Id) http.HandlerFunc {
		return rest.ParseChannelId(d.Logger(), func(channelId channel.Id) http.HandlerFunc {
			return rest.ParseMapId(d.Logger(), func(mapId _map.Id) http.HandlerFunc {
				return rest.ParseInstanceId(d.Logger(), func(instanceId uuid.UUID) http.HandlerFunc {
					return func(w http.ResponseWriter, r *http.Request) {
						transactionId := uuid.New()
						f := field.NewBuilder(worldId, channelId, mapId).SetInstance(instanceId).Build()
						mp := NewProcessor(d.Logger(), d.Context(), nil)
						ids, err := mp.GetCharactersInMap(transactionId, f)
						if err != nil {
							w.WriteHeader(http.StatusInternalServerError)
							return
						}
						res, err := model.SliceMap(Transform)(model.FixedProvider(ids))(model.ParallelMap())()
						if err != nil {
							d.Logger().WithError(err).Errorf("Creating REST model.")
							w.WriteHeader(http.StatusInternalServerError)
							return
						}

						server.Marshal[[]RestModel](d.Logger())(w)(c.ServerInformation())(res)
					}
				})
			})
		})
	})
}
