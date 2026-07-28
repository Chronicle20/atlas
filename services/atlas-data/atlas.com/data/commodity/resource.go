package commodity

import (
	"atlas-data/rest"
	"net/http"
	"sort"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
)

func InitResource(db *gorm.DB) func(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(si jsonapi.ServerInformation) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(si)

			r := router.PathPrefix("/data/commodity/items").Subrouter()
			r.HandleFunc("", registerGet("get_commodity_items", handleGetCommodityItemsRequest(db))).Methods(http.MethodGet)
			r.HandleFunc("/{itemId}", registerGet("get_commodity_item", handleGetCommodityItemRequest(db))).Methods(http.MethodGet)

			byItem := router.PathPrefix("/data/commodity/by-item").Subrouter()
			byItem.HandleFunc("/{itemId}", registerGet("get_commodities_by_item", handleGetCommoditiesByItemRequest(db))).Methods(http.MethodGet)
		}
	}
}

func handleGetCommodityItemsRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
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
				d.Logger().WithError(err).Errorf("Unable to retrieve commodity items.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalPaginatedResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(paged.Items, paginate.EnvelopeFor(paged), r)
		}
	}
}

func handleGetCommoditiesByItemRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseItemId(d.Logger(), func(itemId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				query := r.URL.Query()
				page, err := paginate.ParseParams(query, paginate.DefaultPageSize, paginate.MaxPageSize)
				if err != nil {
					server.WriteBadRequest(d.Logger(), w, err.Error())
					return
				}

				s := NewStorage(d.Logger(), db)
				all, err := s.DrainAllProvider(d.Context())()
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to retrieve commodities for itemId=%d.", itemId)
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				filtered := make([]RestModel, 0)
				for _, cm := range all {
					if cm.ItemId == itemId {
						filtered = append(filtered, cm)
					}
				}

				sort.SliceStable(filtered, func(i, j int) bool {
					return filtered[i].Id < filtered[j].Id
				})

				paged := paginate.Slice(filtered, page)
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalPaginatedResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(paged.Items, paginate.EnvelopeFor(paged), r)
			}
		})
	}
}

func handleGetCommodityItemRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseItemId(d.Logger(), func(itemId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				s := NewStorage(d.Logger(), db)
				res, err := s.GetById(d.Context())(strconv.Itoa(int(itemId)))
				if err != nil {
					d.Logger().WithError(err).Debugf("Unable to locate commodity item %d.", itemId)
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
