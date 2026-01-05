package quest

import (
	"atlas-data/rest"
	"net/http"
	"strconv"

	"github.com/Chronicle20/atlas-rest/server"
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
			r.HandleFunc("/{questId}", registerGet("get_quest", handleGetQuest(db))).Methods(http.MethodGet)
		}
	}
}

func handleGetQuests(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			s := NewStorage(d.Logger(), db)
			res, err := s.GetAll(d.Context())
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to get quests.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
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
