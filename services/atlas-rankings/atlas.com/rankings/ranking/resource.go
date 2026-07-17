package ranking

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"atlas-rankings/rest"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(db)(si)
			r := router.PathPrefix("/rankings").Subrouter()
			r.HandleFunc("/characters", registerGet("get_rankings_for_characters", handleGetRankingsForCharacters)).Methods(http.MethodGet).Queries("ids", "{ids}")
			// Bare /characters (no ids query) is a caller error, not a missing route.
			r.HandleFunc("/characters", registerGet("get_rankings_missing_ids", handleMissingIds)).Methods(http.MethodGet)
			r.HandleFunc("/characters/{characterId}", registerGet("get_ranking_for_character", handleGetRankingForCharacter)).Methods(http.MethodGet)
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

func handleGetRankingsForCharacters(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ids, ok := parseIds(mux.Vars(r)["ids"])
		if !ok {
			server.WriteBadRequest(d.Logger(), w, "ids query parameter must be a comma-separated list of character ids")
			return
		}

		ms, err := NewProcessor(d.Logger(), d.Context(), d.DB()).GetByCharacterIds(ids)
		if err != nil {
			d.Logger().WithError(err).Errorf("Unable to get rankings for characters.")
			server.WriteErrorResponse(d.Logger())(w)(err)
			return
		}

		res, err := model.SliceMap(Transform)(model.FixedProvider(ms))(model.ParallelMap())()
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

func handleGetRankingForCharacter(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			m, err := NewProcessor(d.Logger(), d.Context(), d.DB()).GetByCharacterId(characterId)
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
