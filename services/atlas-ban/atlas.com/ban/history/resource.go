package history

import (
	"atlas-ban/rest"
	"net/http"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
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

		if ip != "" || hwid != "" {
			page, err := paginate.ParseParams(r.URL.Query(), paginate.DefaultPageSize, paginate.MaxPageSize)
			if err != nil {
				server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
				return
			}

			var paged model.Paged[Model]
			if ip != "" {
				paged, err = NewProcessor(d.Logger(), d.Context(), d.DB()).ByIPPagedProvider(ip, page)()
			} else {
				paged, err = NewProcessor(d.Logger(), d.Context(), d.DB()).ByHWIDPagedProvider(hwid, page)()
			}
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to locate login history.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			res, err := model.SliceMap(Transform)(model.FixedProvider(paged.Items))(model.ParallelMap())()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalPaginatedResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res, paginate.EnvelopeFor(paged), r)
			return
		}

		page, err := paginate.ParseParams(r.URL.Query(), paginate.DefaultPageSize, paginate.MaxPageSize)
		if err != nil {
			server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
			return
		}

		paged, err := NewProcessor(d.Logger(), d.Context(), d.DB()).AllProvider(page)()
		if err != nil {
			d.Logger().WithError(err).Errorf("Unable to locate login history.")
			server.WriteErrorResponse(d.Logger())(w)(err)
			return
		}

		res, err := model.SliceMap(Transform)(model.FixedProvider(paged.Items))(model.ParallelMap())()
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating REST model.")
			server.WriteErrorResponse(d.Logger())(w)(err)
			return
		}

		query := r.URL.Query()
		queryParams := jsonapi.ParseQueryFields(&query)
		server.MarshalPaginatedResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res, paginate.EnvelopeFor(paged), r)
	}
}

func handleGetHistoryByAccountId(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseAccountId(d.Logger(), func(accountId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			page, err := paginate.ParseParams(r.URL.Query(), paginate.DefaultPageSize, paginate.MaxPageSize)
			if err != nil {
				server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
				return
			}

			paged, err := NewProcessor(d.Logger(), d.Context(), d.DB()).ByAccountIdProvider(accountId, page)()
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to locate login history for account [%d].", accountId)
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			res, err := model.SliceMap(Transform)(model.FixedProvider(paged.Items))(model.ParallelMap())()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalPaginatedResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res, paginate.EnvelopeFor(paged), r)
		}
	})
}
