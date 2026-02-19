package reactor

import (
	"atlas-data/rest"
	"net/http"
	"strconv"
	"strings"

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

			r := router.PathPrefix("/data/reactors").Subrouter()
			r.HandleFunc("", registerGet("get_reactors", handleGetReactorsRequest(db))).Methods(http.MethodGet)
			r.HandleFunc("/{reactorId}", registerGet("get_reactor", handleGetReactorRequest(db))).Methods(http.MethodGet)
		}
	}
}

func handleGetReactorsRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			s := NewStorage(d.Logger(), db)
			results, err := s.GetAll(d.Context())
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to retrieve reactors.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			query := r.URL.Query()

			searchQuery := query.Get("search")
			if searchQuery != "" {
				results = filterReactors(results, searchQuery, 50)
			}

			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(results)
		}
	}
}

func filterReactors(reactors []RestModel, search string, limit int) []RestModel {
	searchLower := strings.ToLower(search)
	results := make([]RestModel, 0)
	for _, reactor := range reactors {
		if strings.HasPrefix(strconv.Itoa(int(reactor.Id)), search) {
			results = append(results, reactor)
		} else if strings.Contains(strings.ToLower(reactor.Name), searchLower) {
			results = append(results, reactor)
		}
		if len(results) >= limit {
			break
		}
	}
	return results
}

func handleGetReactorRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseReactorId(d.Logger(), func(reactorId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				s := NewStorage(d.Logger(), db)
				res, err := s.GetById(d.Context())(strconv.Itoa(int(reactorId)))
				if err != nil {
					d.Logger().WithError(err).Debugf("Unable to locate reactor %d.", reactorId)
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
