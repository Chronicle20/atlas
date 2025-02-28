package world

import (
	"atlas-monsters/monster"
	"atlas-monsters/rest"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"net/http"
)

const (
	getMonstersInMap   = "get_monsters_in_map"
	createMonsterInMap = "create_monster_in_map"
)

func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		r := router.PathPrefix("/worlds").Subrouter()
		r.HandleFunc("/{worldId}/channels/{channelId}/maps/{mapId}/monsters", rest.RegisterHandler(l)(si)(getMonstersInMap, handleGetMonstersInMap)).Methods(http.MethodGet)
		r.HandleFunc("/{worldId}/channels/{channelId}/maps/{mapId}/monsters", rest.RegisterHandler(l)(si)(getMonstersInMap, handleDeleteMonstersInMap)).Methods(http.MethodDelete)
		r.HandleFunc("/{worldId}/channels/{channelId}/maps/{mapId}/monsters", rest.RegisterInputHandler[monster.RestModel](l)(si)(createMonsterInMap, handleCreateMonsterInMap)).Methods(http.MethodPost)
	}
}

func handleGetMonstersInMap(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseWorldId(d.Logger(), func(worldId byte) http.HandlerFunc {
		return rest.ParseChannelId(d.Logger(), func(channelId byte) http.HandlerFunc {
			return rest.ParseMapId(d.Logger(), func(mapId uint32) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					ms, err := monster.GetInMap(d.Context())(worldId, channelId, mapId)
					if err != nil {
						d.Logger().WithError(err).Errorf("Unable to retrieve monsters in map.")
						w.WriteHeader(http.StatusInternalServerError)
						return
					}

					res, err := model.SliceMap(monster.Transform)(model.FixedProvider(ms))(model.ParallelMap())()
					if err != nil {
						d.Logger().WithError(err).Errorf("Creating REST model.")
						w.WriteHeader(http.StatusInternalServerError)
						return
					}

					server.Marshal[[]monster.RestModel](d.Logger())(w)(c.ServerInformation())(res)
				}
			})
		})
	})
}

func handleDeleteMonstersInMap(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseWorldId(d.Logger(), func(worldId byte) http.HandlerFunc {
		return rest.ParseChannelId(d.Logger(), func(channelId byte) http.HandlerFunc {
			return rest.ParseMapId(d.Logger(), func(mapId uint32) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					err := monster.DestroyInMap(d.Logger())(d.Context())(worldId, channelId, mapId)
					if err != nil {
						d.Logger().WithError(err).Errorf("Unable to remove monsters in map.")
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					w.WriteHeader(http.StatusAccepted)
				}
			})
		})
	})
}

func handleCreateMonsterInMap(d *rest.HandlerDependency, c *rest.HandlerContext, input monster.RestModel) http.HandlerFunc {
	return rest.ParseWorldId(d.Logger(), func(worldId byte) http.HandlerFunc {
		return rest.ParseChannelId(d.Logger(), func(channelId byte) http.HandlerFunc {
			return rest.ParseMapId(d.Logger(), func(mapId uint32) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					m, err := monster.CreateMonster(d.Logger())(d.Context())(worldId, channelId, mapId, input)
					if err != nil {
						d.Logger().WithError(err).Errorf("Unable to create monsters.")
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					res, err := model.Map(monster.Transform)(model.FixedProvider(m))()
					if err != nil {
						d.Logger().WithError(err).Errorf("Creating REST model.")
						w.WriteHeader(http.StatusInternalServerError)
						return
					}

					server.Marshal[monster.RestModel](d.Logger())(w)(c.ServerInformation())(res)
				}
			})
		})
	})
}
