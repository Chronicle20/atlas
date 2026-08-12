package kite

import (
	"atlas-kites/rest"
	"errors"
	"net/http"
	"sort"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
)

func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		registerGet := rest.RegisterHandler(l)(si)

		cr := router.PathPrefix("/kites/{characterId}").Subrouter()
		cr.HandleFunc("", registerGet("kite_by_character_id", handleGetKite)).Methods(http.MethodGet)

		mr := router.PathPrefix("/worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/kites").Subrouter()
		mr.HandleFunc("", registerGet("kites_in_map", handleGetKitesInMap)).Methods(http.MethodGet)
	}
}

func handleGetKite(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			m, err := NewProcessor(d.Logger(), d.Context()).GetByCharacterId(characterId)
			if err != nil {
				if errors.Is(err, ErrNotFound) {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				d.Logger().WithError(err).Errorf("Retrieving kite for character [%d].", characterId)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			res, err := model.Map(Transform)(model.FixedProvider(m))()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
		}
	})
}

func handleGetKitesInMap(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseWorldId(d.Logger(), func(worldId world.Id) http.HandlerFunc {
		return rest.ParseChannelId(d.Logger(), func(channelId channel.Id) http.HandlerFunc {
			return rest.ParseMapId(d.Logger(), func(mapId _map.Id) http.HandlerFunc {
				return rest.ParseInstanceId(d.Logger(), func(instanceId uuid.UUID) http.HandlerFunc {
					return func(w http.ResponseWriter, r *http.Request) {
						page, err := paginate.ParseParams(r.URL.Query(), paginate.MaxPageSize, paginate.MaxPageSize)
						if err != nil {
							server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
							return
						}

						f := field.NewBuilder(worldId, channelId, mapId).SetInstance(instanceId).Build()
						ms, err := NewProcessor(d.Logger(), d.Context()).GetInMap(f)
						if err != nil {
							d.Logger().WithError(err).Errorf("Retrieving kites in map.")
							w.WriteHeader(http.StatusInternalServerError)
							return
						}

						// Sort by kite id (RestModel.Id) BEFORE paginate.Slice, so
						// the page boundary is stable across requests instead of
						// following the registry's unordered iteration order.
						sorted := make([]Model, len(ms))
						copy(sorted, ms)
						sort.Slice(sorted, func(i, j int) bool { return sorted[i].Id() < sorted[j].Id() })
						paged := paginate.Slice(sorted, page)

						res, err := model.SliceMap(Transform)(model.FixedProvider(paged.Items))(model.ParallelMap())()
						if err != nil {
							d.Logger().WithError(err).Errorf("Creating REST model.")
							w.WriteHeader(http.StatusInternalServerError)
							return
						}

						query := r.URL.Query()
						queryParams := jsonapi.ParseQueryFields(&query)
						server.MarshalPaginatedResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res, paginate.EnvelopeFor(paged), r)
					}
				})
			})
		})
	})
}
