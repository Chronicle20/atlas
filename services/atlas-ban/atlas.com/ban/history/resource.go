package history

import (
	"atlas-ban/rest"
	"net/http"

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
			register := rest.RegisterHandler(l)(db)(si)

			r := router.PathPrefix("/history").Subrouter()
			r.HandleFunc("/", register("get_history", handleGetHistory)).Methods(http.MethodGet)
			r.HandleFunc("/accounts/{accountId}", register("get_history_by_account", handleGetHistoryByAccountId)).Methods(http.MethodGet)
		}
	}
}

func handleGetHistory(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := r.URL.Query().Get("ip")
		hwid := r.URL.Query().Get("hwid")

		var entries []Model
		var err error

		if ip != "" {
			entries, err = NewProcessor(d.Logger(), d.Context(), d.DB()).GetByIP(ip)
		} else if hwid != "" {
			entries, err = NewProcessor(d.Logger(), d.Context(), d.DB()).GetByHWID(hwid)
		} else {
			entries, err = NewProcessor(d.Logger(), d.Context(), d.DB()).GetByTenant()
		}

		if err != nil {
			d.Logger().WithError(err).Errorf("Unable to locate login history.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		res, err := model.SliceMap(Transform)(model.FixedProvider(entries))(model.ParallelMap())()
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating REST model.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		query := r.URL.Query()
		queryParams := jsonapi.ParseQueryFields(&query)
		server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
	}
}

func handleGetHistoryByAccountId(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseAccountId(d.Logger(), func(accountId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			entries, err := NewProcessor(d.Logger(), d.Context(), d.DB()).GetByAccountId(accountId)
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to locate login history for account [%d].", accountId)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			res, err := model.SliceMap(Transform)(model.FixedProvider(entries))(model.ParallelMap())()
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
