package environment

import (
	mapKafka "atlas-maps/kafka/message/map"
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
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

const (
	getEnvironmentInMap   = "get_environment_in_map"
	setEnvironmentInMap   = "set_environment_in_map"
	resetEnvironmentInMap = "reset_environment_in_map"
)

func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		r := router.PathPrefix("/worlds").Subrouter()
		path := "/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/environment"
		r.HandleFunc(path, rest.RegisterHandler(l)(si)(getEnvironmentInMap, handleGetEnvironmentInMap)).Methods(http.MethodGet)
		r.HandleFunc(path, rest.RegisterInputHandler[RestModel](l)(si)(setEnvironmentInMap, handleSetEnvironmentInMap)).Methods(http.MethodPost)
		r.HandleFunc(path, rest.RegisterHandler(l)(si)(resetEnvironmentInMap, handleResetEnvironmentInMap)).Methods(http.MethodDelete)
	}
}

// handleGetEnvironmentInMap always responds 200 with a possibly-empty data
// array. Environment state is a collection, not a singleton like weather, so
// an untracked field is not a 404 -- it is simply nothing to report.
func handleGetEnvironmentInMap(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseWorldId(d.Logger(), func(worldId world.Id) http.HandlerFunc {
		return rest.ParseChannelId(d.Logger(), func(channelId channel.Id) http.HandlerFunc {
			return rest.ParseMapId(d.Logger(), func(mapId _map.Id) http.HandlerFunc {
				return rest.ParseInstanceId(d.Logger(), func(instanceId uuid.UUID) http.HandlerFunc {
					return func(w http.ResponseWriter, r *http.Request) {
						f := field.NewBuilder(worldId, channelId, mapId).SetInstance(instanceId).Build()
						entries := NewProcessor(d.Logger(), d.Context()).GetAll(f)

						res := make([]RestModel, 0, len(entries))
						for _, e := range entries {
							rm, err := Transform(e)
							if err != nil {
								d.Logger().WithError(err).Errorf("Creating REST model.")
								server.WriteErrorResponse(d.Logger())(w)(err)
								return
							}
							res = append(res, rm)
						}

						server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(r.URL.Query())(res)
					}
				})
			})
		})
	})
}

// handleSetEnvironmentInMap applies the same Processor.Set the Kafka command
// path uses and emits the same ENVIRONMENT_STATE_CHANGED event, so REST and
// command behave identically -- including the unconditional re-emit on an
// idempotent re-set.
func handleSetEnvironmentInMap(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
	return rest.ParseWorldId(d.Logger(), func(worldId world.Id) http.HandlerFunc {
		return rest.ParseChannelId(d.Logger(), func(channelId channel.Id) http.HandlerFunc {
			return rest.ParseMapId(d.Logger(), func(mapId _map.Id) http.HandlerFunc {
				return rest.ParseInstanceId(d.Logger(), func(instanceId uuid.UUID) http.HandlerFunc {
					return func(w http.ResponseWriter, r *http.Request) {
						kind, err := field.ParseObjectKind(input.Kind)
						if err != nil {
							w.WriteHeader(http.StatusBadRequest)
							return
						}

						f := field.NewBuilder(worldId, channelId, mapId).SetInstance(instanceId).Build()
						entry, err := NewProcessor(d.Logger(), d.Context()).Set(f, kind, input.Name, input.State)
						if err != nil {
							w.WriteHeader(http.StatusBadRequest)
							return
						}

						err = producer.ProviderImpl(d.Logger())(d.Context())(mapKafka.EnvEventTopicMapStatus)(EnvironmentStateChangedEventProvider(uuid.New(), f, entry))
						if err != nil {
							d.Logger().WithError(err).Errorf("Unable to produce environment state changed event for map [%d] instance [%s].", mapId, instanceId)
						}

						w.WriteHeader(http.StatusAccepted)
					}
				})
			})
		})
	})
}

// handleResetEnvironmentInMap clears the field's tracked entries and emits
// the same ENVIRONMENT_RESET event the Kafka command path emits, including
// for an untracked field (Reset returns an empty slice, still 204).
func handleResetEnvironmentInMap(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseWorldId(d.Logger(), func(worldId world.Id) http.HandlerFunc {
		return rest.ParseChannelId(d.Logger(), func(channelId channel.Id) http.HandlerFunc {
			return rest.ParseMapId(d.Logger(), func(mapId _map.Id) http.HandlerFunc {
				return rest.ParseInstanceId(d.Logger(), func(instanceId uuid.UUID) http.HandlerFunc {
					return func(w http.ResponseWriter, r *http.Request) {
						f := field.NewBuilder(worldId, channelId, mapId).SetInstance(instanceId).Build()
						cleared := NewProcessor(d.Logger(), d.Context()).Reset(f)

						err := producer.ProviderImpl(d.Logger())(d.Context())(mapKafka.EnvEventTopicMapStatus)(EnvironmentResetEventProvider(uuid.New(), f, cleared))
						if err != nil {
							d.Logger().WithError(err).Errorf("Unable to produce environment reset event for map [%d] instance [%s].", mapId, instanceId)
						}

						w.WriteHeader(http.StatusNoContent)
					}
				})
			})
		})
	})
}
