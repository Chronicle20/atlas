package chalkboard

import (
	"atlas-chalkboards/character"
	"atlas-chalkboards/rest"
	"net/http"
	"sort"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
)

func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		registerGet := rest.RegisterHandler(l)(si)

		cr := router.PathPrefix("/chalkboards/{characterId}").Subrouter()
		cr.HandleFunc("", registerGet("chalkboard_by_character_id", handleGetChalkboard)).Methods(http.MethodGet)

		mr := router.PathPrefix("/worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/chalkboards").Subrouter()
		mr.HandleFunc("", registerGet("chalkboards_in_map", handleGetChalkboardsInMap)).Methods(http.MethodGet)
	}
}

func handleGetChalkboard(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			p, err := NewProcessor(d.Logger(), d.Context()).GetById(characterId)
			if err != nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}

			res, err := model.Map(Transform)(model.FixedProvider(p))()
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

func handleGetChalkboardsInMap(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
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
						cip := character.NewProcessor(d.Logger(), d.Context()).InMapProvider(f)
						fcip := model.FilteredProvider(cip, model.Filters[uint32](func(cid uint32) bool {
							_, err := NewProcessor(d.Logger(), d.Context()).GetById(cid)
							return err == nil
						}))
						cids, err := fcip()
						if err != nil {
							d.Logger().WithError(err).Errorf("Retrieving characters in map.")
							w.WriteHeader(http.StatusInternalServerError)
							return
						}

						// Sort by characterId (== RestModel.Id here, a genuinely
						// unique key per chalkboard), then page BEFORE the
						// per-item GetById fan-out -- also cuts the N+1 lookup
						// cost to just the current page.
						sorted := make([]uint32, len(cids))
						copy(sorted, cids)
						sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
						paged := paginate.Slice(sorted, page)

						cimp := model.SliceMap[uint32, Model](NewProcessor(d.Logger(), d.Context()).GetById)(model.FixedProvider(paged.Items))(model.ParallelMap())

						res, err := model.SliceMap(Transform)(cimp)(model.ParallelMap())()
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
