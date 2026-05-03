package world

import (
	"atlas-monsters/monster"
	"atlas-monsters/rest"

	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
)

const (
	getMonstersInMap     = "get_monsters_in_map"
	getMonstersInMapRect = "get_monsters_in_map_rect"
	createMonsterInMap   = "create_monster_in_map"
)

func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		r := router.PathPrefix("/worlds").Subrouter()
		r.HandleFunc("/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/monsters/in-rect", rest.RegisterHandler(l)(si)(getMonstersInMapRect, handleGetMonstersInMapRect)).Methods(http.MethodGet)
		r.HandleFunc("/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/monsters", rest.RegisterHandler(l)(si)(getMonstersInMap, handleGetMonstersInMap)).Methods(http.MethodGet)
		r.HandleFunc("/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/monsters", rest.RegisterHandler(l)(si)(getMonstersInMap, handleDeleteMonstersInMap)).Methods(http.MethodDelete)
		r.HandleFunc("/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/monsters", rest.RegisterInputHandler[monster.RestModel](l)(si)(createMonsterInMap, handleCreateMonsterInMap)).Methods(http.MethodPost)
	}
}

func handleGetMonstersInMap(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseWorldId(d.Logger(), func(worldId world.Id) http.HandlerFunc {
		return rest.ParseChannelId(d.Logger(), func(channelId channel.Id) http.HandlerFunc {
			return rest.ParseMapId(d.Logger(), func(mapId _map.Id) http.HandlerFunc {
				return rest.ParseInstanceId(d.Logger(), func(instance uuid.UUID) http.HandlerFunc {
					return func(w http.ResponseWriter, r *http.Request) {
						f := field.NewBuilder(worldId, channelId, mapId).SetInstance(instance).Build()
						p := monster.NewProcessor(d.Logger(), d.Context())
						ms, err := p.GetInField(f)
						if err != nil {
							d.Logger().WithError(err).Errorf("Unable to retrieve monsters in field.")
							w.WriteHeader(http.StatusInternalServerError)
							return
						}

						res, err := model.SliceMap(monster.Transform)(model.FixedProvider(ms))(model.ParallelMap())()
						if err != nil {
							d.Logger().WithError(err).Errorf("Creating REST model.")
							w.WriteHeader(http.StatusInternalServerError)
							return
						}

						server.MarshalResponse[[]monster.RestModel](d.Logger())(w)(c.ServerInformation())(r.URL.Query())(res)
					}
				})
			})
		})
	})
}

func handleDeleteMonstersInMap(d *rest.HandlerDependency, _ *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseWorldId(d.Logger(), func(worldId world.Id) http.HandlerFunc {
		return rest.ParseChannelId(d.Logger(), func(channelId channel.Id) http.HandlerFunc {
			return rest.ParseMapId(d.Logger(), func(mapId _map.Id) http.HandlerFunc {
				return rest.ParseInstanceId(d.Logger(), func(instance uuid.UUID) http.HandlerFunc {
					return func(w http.ResponseWriter, r *http.Request) {
						f := field.NewBuilder(worldId, channelId, mapId).SetInstance(instance).Build()
						p := monster.NewProcessor(d.Logger(), d.Context())
						err := p.DestroyInField(f)
						if err != nil {
							d.Logger().WithError(err).Errorf("Unable to remove monsters in field.")
							w.WriteHeader(http.StatusInternalServerError)
							return
						}
						w.WriteHeader(http.StatusAccepted)
					}
				})
			})
		})
	})
}

func handleGetMonstersInMapRect(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseWorldId(d.Logger(), func(worldId world.Id) http.HandlerFunc {
		return rest.ParseChannelId(d.Logger(), func(channelId channel.Id) http.HandlerFunc {
			return rest.ParseMapId(d.Logger(), func(mapId _map.Id) http.HandlerFunc {
				return rest.ParseInstanceId(d.Logger(), func(instance uuid.UUID) http.HandlerFunc {
					return func(w http.ResponseWriter, r *http.Request) {
						q := r.URL.Query()
						x1, e1 := parseInt16Query(q, "x1")
						y1, e2 := parseInt16Query(q, "y1")
						x2, e3 := parseInt16Query(q, "x2")
						y2, e4 := parseInt16Query(q, "y2")
						if e1 != nil || e2 != nil || e3 != nil || e4 != nil {
							w.WriteHeader(http.StatusBadRequest)
							return
						}
						limit, _ := parseUint32QueryOrDefault(q, "limit", 0)

						f := field.NewBuilder(worldId, channelId, mapId).SetInstance(instance).Build()
						p := monster.NewProcessor(d.Logger(), d.Context())
						ms, err := p.GetInFieldRect(f, x1, y1, x2, y2, limit)
						if err != nil {
							d.Logger().WithError(err).Errorf("Unable to retrieve monsters in field rect.")
							w.WriteHeader(http.StatusInternalServerError)
							return
						}
						res, err := model.SliceMap(monster.Transform)(model.FixedProvider(ms))(model.ParallelMap())()
						if err != nil {
							d.Logger().WithError(err).Errorf("Creating REST model.")
							w.WriteHeader(http.StatusInternalServerError)
							return
						}
						server.MarshalResponse[[]monster.RestModel](d.Logger())(w)(c.ServerInformation())(r.URL.Query())(res)
					}
				})
			})
		})
	})
}

// parseInt16Query parses a required int16 from the URL query.
func parseInt16Query(q url.Values, name string) (int16, error) {
	raw := q.Get(name)
	if raw == "" {
		return 0, fmt.Errorf("missing %s", name)
	}
	n, err := strconv.ParseInt(raw, 10, 16)
	if err != nil {
		return 0, err
	}
	return int16(n), nil
}

// parseUint32QueryOrDefault parses an optional uint32 from the URL query.
// Returns def if the value is missing or unparseable.
func parseUint32QueryOrDefault(q url.Values, name string, def uint32) (uint32, error) {
	raw := q.Get(name)
	if raw == "" {
		return def, nil
	}
	n, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		return def, err
	}
	return uint32(n), nil
}

func handleCreateMonsterInMap(d *rest.HandlerDependency, c *rest.HandlerContext, input monster.RestModel) http.HandlerFunc {
	return rest.ParseWorldId(d.Logger(), func(worldId world.Id) http.HandlerFunc {
		return rest.ParseChannelId(d.Logger(), func(channelId channel.Id) http.HandlerFunc {
			return rest.ParseMapId(d.Logger(), func(mapId _map.Id) http.HandlerFunc {
				return rest.ParseInstanceId(d.Logger(), func(instance uuid.UUID) http.HandlerFunc {
					return func(w http.ResponseWriter, r *http.Request) {
						f := field.NewBuilder(worldId, channelId, mapId).SetInstance(instance).Build()
						p := monster.NewProcessor(d.Logger(), d.Context())
						m, err := p.Create(f, input)
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

						server.MarshalResponse[monster.RestModel](d.Logger())(w)(c.ServerInformation())(r.URL.Query())(res)
					}
				})
			})
		})
	})
}
