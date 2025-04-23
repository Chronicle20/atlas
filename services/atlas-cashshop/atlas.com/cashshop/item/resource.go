package item

import (
	"atlas-cashshop/rest"
	"errors"
	"net/http"
	"strconv"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(si)
			r := router.PathPrefix("/cash-shop/items").Subrouter()
			r.HandleFunc("", registerGet("get_item", handleGetItem(db))).Methods(http.MethodGet)
		}
	}
}

func handleGetItem(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return ParseItemId(d.Logger(), func(itemId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				ms, err := GetById(d.Context())(db)(itemId)
				if errors.Is(err, gorm.ErrRecordNotFound) {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				res, err := model.SliceMap(Transform)(model.FixedProvider(ms))(model.ParallelMap())()
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST model.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
			}
		})
	}
}

// TODO refactor
type ItemIdHandler func(itemId uint32) http.HandlerFunc

func ParseItemId(l logrus.FieldLogger, next ItemIdHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		itemId, err := strconv.Atoi(mux.Vars(r)["itemId"])
		if err != nil {
			l.WithError(err).Errorf("Unable to properly parse itemId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(uint32(itemId))(w, r)
	}
}
