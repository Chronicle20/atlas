package weather

import (
	"atlas-maps/rest"
	"net/http"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
)

const (
	getWeatherInMap = "get_weather_in_map"
)

func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		r := router.PathPrefix("/worlds").Subrouter()
		r.HandleFunc("/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/weather", rest.RegisterHandler(l)(si)(getWeatherInMap, handleGetWeatherInMap)).Methods(http.MethodGet)
	}
}

func handleGetWeatherInMap(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseWorldId(d.Logger(), func(worldId world.Id) http.HandlerFunc {
		return rest.ParseChannelId(d.Logger(), func(channelId channel.Id) http.HandlerFunc {
			return rest.ParseMapId(d.Logger(), func(mapId _map.Id) http.HandlerFunc {
				return rest.ParseInstanceId(d.Logger(), func(instanceId uuid.UUID) http.HandlerFunc {
					return func(w http.ResponseWriter, r *http.Request) {
						f := field.NewBuilder(worldId, channelId, mapId).SetInstance(instanceId).Build()
						wp := NewProcessor(d.Logger(), d.Context())
						entry, ok := wp.GetActive(f)
						if !ok {
							w.WriteHeader(http.StatusNotFound)
							return
						}

						res, err := Transform(entry)
						if err != nil {
							d.Logger().WithError(err).Errorf("Creating REST model.")
							w.WriteHeader(http.StatusInternalServerError)
							return
						}

						server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(r.URL.Query())(res)
					}
				})
			})
		})
	})
}
