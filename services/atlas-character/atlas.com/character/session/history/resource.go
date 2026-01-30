package history

import (
	"atlas-character/rest"
	"net/http"
	"strconv"
	"time"

	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(db)(si)
			r := router.PathPrefix("/characters/{characterId}/sessions").Subrouter()
			r.HandleFunc("", registerGet("get_character_sessions", handleGetSessions)).Methods(http.MethodGet)
			r.HandleFunc("/playtime", registerGet("get_character_playtime", handleGetPlaytime)).Methods(http.MethodGet)
		}
	}
}

func handleGetSessions(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Parse 'since' query parameter
			sinceStr := r.URL.Query().Get("since")
			var since time.Time
			if sinceStr != "" {
				sinceUnix, err := strconv.ParseInt(sinceStr, 10, 64)
				if err != nil {
					// Try parsing as RFC3339
					since, err = time.Parse(time.RFC3339, sinceStr)
					if err != nil {
						w.WriteHeader(http.StatusBadRequest)
						return
					}
				} else {
					since = time.Unix(sinceUnix, 0)
				}
			} else {
				// Default to 24 hours ago
				since = time.Now().Add(-24 * time.Hour)
			}

			p := NewProcessor(d.Logger(), d.Context(), d.DB())
			sessions, err := p.GetSessionsSince(characterId, since)
			if err != nil {
				d.Logger().WithError(err).Errorf("Failed to get sessions for character [%d].", characterId)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			result := TransformSliceToRest(sessions)

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(result)
		}
	})
}

func handleGetPlaytime(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Parse 'since' query parameter
			sinceStr := r.URL.Query().Get("since")
			var since time.Time
			if sinceStr != "" {
				sinceUnix, err := strconv.ParseInt(sinceStr, 10, 64)
				if err != nil {
					// Try parsing as RFC3339
					since, err = time.Parse(time.RFC3339, sinceStr)
					if err != nil {
						w.WriteHeader(http.StatusBadRequest)
						return
					}
				} else {
					since = time.Unix(sinceUnix, 0)
				}
			} else {
				// Default to 24 hours ago
				since = time.Now().Add(-24 * time.Hour)
			}

			p := NewProcessor(d.Logger(), d.Context(), d.DB())
			playtime, err := p.ComputePlaytimeSince(characterId, since)
			if err != nil {
				d.Logger().WithError(err).Errorf("Failed to compute playtime for character [%d].", characterId)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			result := PlaytimeResponse{
				Id:            strconv.FormatUint(uint64(characterId), 10),
				CharacterId:   characterId,
				TotalSeconds:  int64(playtime.Seconds()),
				FormattedTime: FormatDuration(playtime),
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[PlaytimeResponse](d.Logger())(w)(c.ServerInformation())(queryParams)(result)
		}
	})
}
