package game

import (
	"atlas-mini-games/rest"
	"net/http"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
)

const (
	GetGamesInField = "get_games_in_field"
)

// InitResource wires GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/games
// (chalkboards resource.go:55-84 shape), returning every mini-game room
// currently registered in that field so atlas-channel can reconcile its
// local view on portal-enter (task-19 REST client).
func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		registerGet := rest.RegisterHandler(l)(si)

		r := router.PathPrefix("/worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/games").Subrouter()
		r.HandleFunc("", registerGet(GetGamesInField, handleGetGamesInField)).Methods(http.MethodGet)
	}
}

func handleGetGamesInField(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseWorldId(d.Logger(), func(worldId world.Id) http.HandlerFunc {
		return rest.ParseChannelId(d.Logger(), func(channelId channel.Id) http.HandlerFunc {
			return rest.ParseMapId(d.Logger(), func(mapId _map.Id) http.HandlerFunc {
				return rest.ParseInstanceId(d.Logger(), func(instanceId uuid.UUID) http.HandlerFunc {
					return func(w http.ResponseWriter, r *http.Request) {
						t := tenant.MustFromContext(d.Context())
						f := field.NewBuilder(worldId, channelId, mapId).SetInstance(instanceId).Build()

						rooms := GetRegistry().GetInField(t, f)

						res, err := model.SliceMap(Transform)(model.FixedProvider(rooms))(model.ParallelMap())()
						if err != nil {
							d.Logger().WithError(err).Errorf("Creating REST model.")
							w.WriteHeader(http.StatusInternalServerError)
							return
						}

						query := r.URL.Query()
						queryParams := jsonapi.ParseQueryFields(&query)
						server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
					}
				})
			})
		})
	})
}
