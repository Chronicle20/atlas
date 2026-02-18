package reactor

import (
	"atlas-reactors/kafka/producer"
	"atlas-reactors/rest"
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

func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		registerGet := rest.RegisterHandler(l)(si)
		r := router.PathPrefix("/reactors").Subrouter()
		r.HandleFunc("/{reactorId}", registerGet("get_by_id", handleGetById)).Methods(http.MethodGet)

		r = router.PathPrefix("/worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/reactors").Subrouter()
		r.HandleFunc("", rest.RegisterInputHandler[RestModel](l)(si)("create_in_map", handleCreateInMap)).Methods(http.MethodPost)
		r.HandleFunc("", registerGet("get_in_map", handleGetInMap)).Methods(http.MethodGet)
		r.HandleFunc("/{reactorId}", registerGet("get_by_id", handleGetByIdInMap)).Methods(http.MethodGet)
	}
}

func handleGetById(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseReactorId(d.Logger(), func(reactorId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			m, err := GetById(d.Logger())(d.Context())(reactorId)
			if err != nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}

			res, err := model.Map(Transform)(model.FixedProvider(m))()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			server.Marshal[RestModel](d.Logger())(w)(c.ServerInformation())(res)
		}
	})
}

func handleGetByIdInMap(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseWorldId(d.Logger(), func(worldId world.Id) http.HandlerFunc {
		return rest.ParseChannelId(d.Logger(), func(channelId channel.Id) http.HandlerFunc {
			return rest.ParseMapId(d.Logger(), func(mapId _map.Id) http.HandlerFunc {
				return rest.ParseInstanceId(d.Logger(), func(instanceId uuid.UUID) http.HandlerFunc {
					return rest.ParseReactorId(d.Logger(), func(reactorId uint32) http.HandlerFunc {
						return func(w http.ResponseWriter, r *http.Request) {
							m, err := GetById(d.Logger())(d.Context())(reactorId)
							if err != nil || m.WorldId() != worldId || m.ChannelId() != channelId || m.MapId() != mapId {
								w.WriteHeader(http.StatusNotFound)
								return
							}

							res, err := model.Map(Transform)(model.FixedProvider(m))()
							if err != nil {
								d.Logger().WithError(err).Errorf("Creating REST model.")
								w.WriteHeader(http.StatusInternalServerError)
								return
							}

							server.Marshal[RestModel](d.Logger())(w)(c.ServerInformation())(res)
						}
					})
				})
			})
		})
	})
}

func handleCreateInMap(d *rest.HandlerDependency, _ *rest.HandlerContext, i RestModel) http.HandlerFunc {
	return rest.ParseWorldId(d.Logger(), func(worldId world.Id) http.HandlerFunc {
		return rest.ParseChannelId(d.Logger(), func(channelId channel.Id) http.HandlerFunc {
			return rest.ParseMapId(d.Logger(), func(mapId _map.Id) http.HandlerFunc {
				return rest.ParseInstanceId(d.Logger(), func(instanceId uuid.UUID) http.HandlerFunc {
					return func(w http.ResponseWriter, r *http.Request) {
						f := field.NewBuilder(worldId, channelId, mapId).SetInstance(instanceId).Build()
						err := producer.ProviderImpl(d.Logger())(d.Context())(EnvCommandTopic)(createCommandProvider(f, i.Classification, i.Name, i.State, i.X, i.Y, i.Delay, i.Direction))
						if err != nil {
							d.Logger().WithError(err).Errorf("Unable to accept reactor creation request for processing.")
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

func handleGetInMap(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseWorldId(d.Logger(), func(worldId world.Id) http.HandlerFunc {
		return rest.ParseChannelId(d.Logger(), func(channelId channel.Id) http.HandlerFunc {
			return rest.ParseMapId(d.Logger(), func(mapId _map.Id) http.HandlerFunc {
				return rest.ParseInstanceId(d.Logger(), func(instanceId uuid.UUID) http.HandlerFunc {
					return func(w http.ResponseWriter, r *http.Request) {
						f := field.NewBuilder(worldId, channelId, mapId).SetInstance(instanceId).Build()
						ms, err := GetInField(d.Logger())(d.Context())(f)
						if err != nil {
							w.WriteHeader(http.StatusInternalServerError)
							return
						}

						// Filter by name if query parameter provided.
						if name := r.URL.Query().Get("name"); name != "" {
							filtered := make([]Model, 0)
							for _, m := range ms {
								if m.Name() == name {
									filtered = append(filtered, m)
								}
							}
							ms = filtered
						}

						res, err := model.SliceMap(Transform)(model.FixedProvider(ms))()()
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
