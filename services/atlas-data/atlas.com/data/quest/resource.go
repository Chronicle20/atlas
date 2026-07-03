package quest

import (
	"atlas-data/rest"
	"net/http"
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitResource(db *gorm.DB) func(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(si jsonapi.ServerInformation) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(si)

			r := router.PathPrefix("/data/quests").Subrouter()
			r.HandleFunc("", registerGet("get_quests", handleGetQuests(db))).Methods(http.MethodGet)
			r.HandleFunc("/auto-start", registerGet("get_auto_start_quests", handleGetAutoStartQuests(db))).Methods(http.MethodGet)
			r.HandleFunc("/{questId}", registerGet("get_quest", handleGetQuest(db))).Methods(http.MethodGet)
		}
	}
}

func handleGetQuests(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query()
			page, err := paginate.ParseParams(query, paginate.DefaultPageSize, paginate.MaxPageSize)
			if err != nil {
				server.WriteBadRequest(d.Logger(), w, err.Error())
				return
			}

			s := NewStorage(d.Logger(), db)
			paged, err := s.AllPagedProvider(d.Context())(page)()
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to get quests.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalPaginatedResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(paged.Items, paginate.EnvelopeFor(paged), r)
		}
	}
}

func handleGetQuest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseQuestId(d.Logger(), func(questId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				s := NewStorage(d.Logger(), db)
				res, err := s.GetById(d.Context())(strconv.Itoa(int(questId)))
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to get quest %d.", questId)
					w.WriteHeader(http.StatusNotFound)
					return
				}
				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
			}
		})
	}
}

func handleGetAutoStartQuests(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query()
			page, err := paginate.ParseParams(query, paginate.DefaultPageSize, paginate.MaxPageSize)
			if err != nil {
				server.WriteBadRequest(d.Logger(), w, err.Error())
				return
			}

			s := NewStorage(d.Logger(), db)
			allQuests, err := s.GetAll(d.Context())
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to get quests for auto-start filter.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			// Filter for auto-start quests
			autoStartQuests := make([]RestModel, 0)
			for _, q := range allQuests {
				if q.AutoStart {
					autoStartQuests = append(autoStartQuests, q)
				}
			}

			paged := paginate.Slice(autoStartQuests, page)
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalPaginatedResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(paged.Items, paginate.EnvelopeFor(paged), r)
		}
	}
}
