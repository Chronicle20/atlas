package chair

import (
	"atlas-chairs/character"
	"atlas-chairs/rest"
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

		cr := router.PathPrefix("/chairs/{characterId}").Subrouter()
		cr.HandleFunc("", registerGet("chairs_by_character_id", handleGetChair)).Methods(http.MethodGet)

		mr := router.PathPrefix("/worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/chairs").Subrouter()
		mr.HandleFunc("", registerGet("chairs_in_map", handleGetChairsInMap)).Methods(http.MethodGet)
	}
}

func handleGetChair(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			p, err := NewProcessor(d.Logger(), d.Context()).GetById(characterId)
			if err != nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}

			res, err := model.Map(Transform(characterId))(model.FixedProvider(p))()
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

func handleGetChairsInMap(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
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

						// Sort by characterId, the collection's true unique key --
						// RestModel.Id is the sat-in chair's item/object id, which
						// is NOT unique across characters sharing the same chair
						// type (a pre-existing, unrelated JSON:API id-collision
						// risk, not introduced or fixed here).
						sorted := make([]uint32, len(cids))
						copy(sorted, cids)
						sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
						paged := paginate.Slice(sorted, page)

						res, err := model.SliceMap(func(cid uint32) (RestModel, error) {
							cm, err := NewProcessor(d.Logger(), d.Context()).GetById(cid)
							if err != nil {
								return RestModel{}, err
							}
							return Transform(cid)(cm)
						})(model.FixedProvider(paged.Items))(model.ParallelMap())()
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
