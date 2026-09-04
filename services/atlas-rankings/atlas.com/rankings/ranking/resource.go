package ranking

import (
	"atlas-rankings/rest"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(si)
			r := router.PathPrefix("/rankings").Subrouter()
			r.HandleFunc("", registerGet("get_leaderboard", handleGetLeaderboard(db))).Methods(http.MethodGet)
			r.HandleFunc("/characters", registerGet("get_rankings_for_characters", handleGetRankingsForCharacters(db))).Methods(http.MethodGet).Queries("ids", "{ids}")
			// Bare /characters (no ids query) is a caller error, not a missing route.
			r.HandleFunc("/characters", registerGet("get_rankings_missing_ids", handleMissingIds)).Methods(http.MethodGet)
			r.HandleFunc("/characters/{characterId}", registerGet("get_ranking_for_character", handleGetRankingForCharacter(db))).Methods(http.MethodGet)
		}
	}
}

func handleGetLeaderboard(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			q := r.URL.Query()

			rawWorld := q.Get("filter[worldId]")
			if rawWorld == "" {
				server.WriteBadRequest(d.Logger(), w, "filter[worldId] query parameter is required")
				return
			}
			wid64, err := strconv.ParseUint(rawWorld, 10, 8)
			if err != nil {
				server.WriteBadRequest(d.Logger(), w, "filter[worldId] must be a valid world id")
				return
			}
			worldId := world.Id(wid64)

			var jobCategory *uint16
			if rawCat := q.Get("filter[jobCategory]"); rawCat != "" {
				cat64, err := strconv.ParseUint(rawCat, 10, 16)
				if err != nil {
					server.WriteBadRequest(d.Logger(), w, "filter[jobCategory] must be a valid job category")
					return
				}
				cat := uint16(cat64)
				jobCategory = &cat
			}

			page, err := paginate.ParseParams(q, paginate.DefaultPageSize, paginate.MaxPageSize)
			if err != nil {
				server.WriteBadRequest(d.Logger(), w, err.Error())
				return
			}

			paged, err := NewProcessor(d.Logger(), d.Context(), db).LeaderboardProvider(worldId, jobCategory, page)()
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to read leaderboard for world [%d].", worldId)
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			res, err := model.SliceMap(TransformLeaderboard)(model.FixedProvider(paged.Items))(model.ParallelMap())()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating leaderboard REST model.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			queryParams := jsonapi.ParseQueryFields(&q)
			server.MarshalPaginatedResponse[[]LeaderboardRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res, paginate.EnvelopeFor(paged), r)
		}
	}
}

func handleMissingIds(d *rest.HandlerDependency, _ *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		server.WriteBadRequest(d.Logger(), w, "ids query parameter is required")
	}
}

// parseIds splits a comma-separated ids query value into character ids.
// Blank segments are skipped so trailing/leading/duplicate commas don't
// error; an empty, all-blank, or malformed (non-numeric) input is rejected.
func parseIds(raw string) ([]uint32, bool) {
	parts := strings.Split(raw, ",")
	ids := make([]uint32, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseUint(part, 10, 32)
		if err != nil {
			return nil, false
		}
		ids = append(ids, uint32(id))
	}
	if len(ids) == 0 {
		return nil, false
	}
	return ids, true
}

func handleGetRankingsForCharacters(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ids, ok := parseIds(mux.Vars(r)["ids"])
			if !ok {
				server.WriteBadRequest(d.Logger(), w, "ids query parameter must be a comma-separated list of character ids")
				return
			}

			ms, err := NewProcessor(d.Logger(), d.Context(), db).GetByCharacterIds(ids)
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to get rankings for characters.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			res, err := TransformSlice(ms)
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
		}
	}
}

func handleGetRankingForCharacter(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				m, err := NewProcessor(d.Logger(), d.Context(), db).GetByCharacterId(characterId)
				if err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					d.Logger().WithError(err).Errorf("Unable to get ranking for character [%d].", characterId)
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				res, err := Transform(m)
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST model.")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
			}
		})
	}
}
