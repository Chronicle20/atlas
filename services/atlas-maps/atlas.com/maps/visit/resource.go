package visit

import (
	"atlas-maps/rest"
	"errors"
	"net/http"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerHandler := rest.RegisterHandler(l)(si)

			r := router.PathPrefix("/characters").Subrouter()
			r.HandleFunc("/{characterId}/visits", registerHandler("get_character_visits", handleGetCharacterVisits(db))).Methods(http.MethodGet)
			r.HandleFunc("/{characterId}/visits/{mapId}", registerHandler("get_character_visit", handleGetCharacterVisit(db))).Methods(http.MethodGet)
		}
	}
}

func handleGetCharacterVisits(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				page, err := paginate.ParseParams(r.URL.Query(), paginate.DefaultPageSize, paginate.MaxPageSize)
				if err != nil {
					server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
					return
				}

				p := NewProcessor(d.Logger(), d.Context(), db)
				paged, err := p.ByCharacterIdProvider(characterId, page)()
				if err != nil {
					d.Logger().WithError(err).Errorf("Retrieving visits.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				rm, err := model.SliceMap(Transform)(model.FixedProvider(paged.Items))(model.ParallelMap())()
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST model.")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalPaginatedResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm, paginate.EnvelopeFor(paged), r)
			}
		})
	}
}

func handleGetCharacterVisit(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return rest.ParseMapId(d.Logger(), func(mapId _map.Id) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					p := NewProcessor(d.Logger(), d.Context(), db)
					v, err := p.ByCharacterIdAndMapIdProvider(characterId, mapId)()
					if errors.Is(err, gorm.ErrRecordNotFound) {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					if err != nil {
						d.Logger().WithError(err).Errorf("Retrieving visit.")
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}

					rm, err := model.Map(Transform)(model.FixedProvider(v))()
					if err != nil {
						d.Logger().WithError(err).Errorf("Creating REST model.")
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}

					query := r.URL.Query()
					queryParams := jsonapi.ParseQueryFields(&query)
					server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
				}
			})
		})
	}
}
