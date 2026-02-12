package item

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

func InitStringResource(db *gorm.DB) func(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(si jsonapi.ServerInformation) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(si)

			r := router.PathPrefix("/data/item-strings").Subrouter()
			r.HandleFunc("", registerGet("get_item_strings", handleGetItemStringsRequest(db))).Methods(http.MethodGet)
			r.HandleFunc("/{itemId}", registerGet("get_item_string", handleGetItemStringRequest(db))).Methods(http.MethodGet)
		}
	}
}

func handleGetItemStringsRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			s := NewStringStorage(d.Logger(), db)
			res, err := s.GetAll(d.Context())
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to retrieve item strings.")
				w.WriteHeader(http.StatusNotFound)
				return
			}

			searchQuery := r.URL.Query().Get("search")
			if searchQuery != "" {
				res = filterItemStrings(res, searchQuery, 50)
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[[]StringRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
		}
	}
}

func filterItemStrings(items []StringRestModel, search string, limit int) []StringRestModel {
	searchLower := strings.ToLower(search)
	results := make([]StringRestModel, 0)
	for _, item := range items {
		if strings.HasPrefix(item.Id, search) {
			results = append(results, item)
		} else if strings.Contains(strings.ToLower(item.Name), searchLower) {
			results = append(results, item)
		}
		if len(results) >= limit {
			break
		}
	}
	return results
}

func handleGetItemStringRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseItemId(d.Logger(), func(itemId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				s := NewStringStorage(d.Logger(), db)
				res, err := s.GetById(d.Context())(strconv.Itoa(int(itemId)))
				if err != nil {
					d.Logger().WithError(err).Debugf("Unable to locate item string for %d.", itemId)
					w.WriteHeader(http.StatusNotFound)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[StringRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
			}
		})
	}
}
