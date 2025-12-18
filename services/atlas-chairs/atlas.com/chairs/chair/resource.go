package chair

import (
	"atlas-chairs/character"
	"atlas-chairs/rest"
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"net/http"
)

func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		registerGet := rest.RegisterHandler(l)(si)

		cr := router.PathPrefix("/chairs/{characterId}").Subrouter()
		cr.HandleFunc("", registerGet("chairs_by_character_id", handleGetChair)).Methods(http.MethodGet)

		mr := router.PathPrefix("/worlds/{worldId}/channels/{channelId}/maps/{mapId}/chairs").Subrouter()
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
				return func(w http.ResponseWriter, r *http.Request) {
					f := field.NewBuilder(worldId, channelId, mapId).Build()
					cip := character.NewProcessor(d.Logger(), d.Context()).InMapProvider(f)
					fcip := model.FilteredProvider(cip, model.Filters[uint32](func(cid uint32) bool {
						_, err := NewProcessor(d.Logger(), d.Context()).GetById(cid)
						return err == nil
					}))
					res, err := model.SliceMap(func(cid uint32) (RestModel, error) {
						cm, err := NewProcessor(d.Logger(), d.Context()).GetById(cid)
						if err != nil {
							return RestModel{}, err
						}
						return Transform(cid)(cm)
					})(fcip)(model.ParallelMap())()
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
}
