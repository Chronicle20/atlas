package consumable

import (
	"atlas-data/rest"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"net/http"
	"strconv"
)

func InitResource(db *gorm.DB) func(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(si jsonapi.ServerInformation) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(si)

			r := router.PathPrefix("/data/consumables").Subrouter()
			r.HandleFunc("", registerGet("get_consumables", handleGetConsumablesRequest(db))).Methods(http.MethodGet)
			r.HandleFunc("/{itemId}", registerGet("get_consumable", handleGetConsumableRequest(db))).Methods(http.MethodGet)
		}
	}
}

func handleGetConsumablesRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			var filters []model.Filter[RestModel]
			query := r.URL.Query()
			if rechargeableFilter, ok := query["filter[rechargeable]"]; ok && len(rechargeableFilter) > 0 {
				if rechargeableFilter[0] == "true" {
					filters = append(filters, func(rm RestModel) bool {
						return rm.Rechargeable == true
					})
				} else {
					filters = append(filters, func(rm RestModel) bool {
						return rm.Rechargeable == false
					})
				}
			}

			res, err := model.FilteredProvider(NewStorage(d.Logger(), db).AllProvider(d.Context()), filters)()
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
		}
	}
}

func handleGetConsumableRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseItemId(d.Logger(), func(itemId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				s := NewStorage(d.Logger(), db)
				res, err := s.GetById(d.Context())(strconv.Itoa(int(itemId)))
				if err != nil {
					d.Logger().WithError(err).Debugf("Unable to locate consumable %d.", itemId)
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
