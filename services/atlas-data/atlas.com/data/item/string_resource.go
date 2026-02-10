package item

import (
	"atlas-data/rest"
	"net/http"
	"strconv"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
)

func InitStringResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		registerGet := rest.RegisterHandler(l)(si)

		r := router.PathPrefix("/data/item-strings").Subrouter()
		r.HandleFunc("", registerGet("get_item_strings", handleGetItemStringsRequest)).Methods(http.MethodGet)
		r.HandleFunc("/{itemId}", registerGet("get_item_string", handleGetItemStringRequest)).Methods(http.MethodGet)
	}
}

func handleGetItemStringsRequest(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t := tenant.MustFromContext(d.Context())

		results, err := GetItemStringRegistry().GetAll(t)
		if err != nil {
			d.Logger().WithError(err).Errorf("Unable to retrieve item strings.")
			w.WriteHeader(http.StatusNotFound)
			return
		}

		res, err := model.SliceMap(TransformString)(model.FixedProvider(results))(model.ParallelMap())()
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating REST model.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		query := r.URL.Query()
		queryParams := jsonapi.ParseQueryFields(&query)
		server.MarshalResponse[[]StringRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
	}
}

func handleGetItemStringRequest(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseItemId(d.Logger(), func(itemId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			t := tenant.MustFromContext(d.Context())

			result, err := GetItemStringRegistry().Get(t, strconv.Itoa(int(itemId)))
			if err != nil {
				d.Logger().WithError(err).Debugf("Unable to locate item string for %d.", itemId)
				w.WriteHeader(http.StatusNotFound)
				return
			}

			rm, err := TransformString(result)
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[StringRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
		}
	})
}
