package global

import (
	"atlas-gachapons/rest"
	"net/http"
	"strconv"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(db)(si)
			registerInput := rest.RegisterInputHandler[RestModel](l)(db)(si)

			r := router.PathPrefix("/global-items").Subrouter()
			r.HandleFunc("", registerGet("get_all_global_items", handleGetAllGlobalItems)).Methods(http.MethodGet)
			r.HandleFunc("", registerInput("create_global_item", handleCreateGlobalItem)).Methods(http.MethodPost)
			r.HandleFunc("/{itemId}", registerGet("delete_global_item", handleDeleteGlobalItem)).Methods(http.MethodDelete)
		}
	}
}

func handleGetAllGlobalItems(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tier := r.URL.Query().Get("tier")

		var ms []Model
		var err error
		p := NewProcessor(d.Logger(), d.Context(), d.DB())
		if tier != "" {
			ms, err = p.GetByTier(tier)()
		} else {
			ms, err = p.GetAll()()
		}
		if err != nil {
			d.Logger().WithError(err).Errorf("Retrieving global items.")
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
}

func handleCreateGlobalItem(d *rest.HandlerDependency, c *rest.HandlerContext, rm RestModel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t := tenant.MustFromContext(d.Context())
		m, err := NewBuilder(t.Id(), 0).
			SetItemId(rm.ItemId).
			SetQuantity(rm.Quantity).
			SetTier(rm.Tier).
			Build()
		if err != nil {
			d.Logger().WithError(err).Errorf("Building global item model.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		err = NewProcessor(d.Logger(), d.Context(), d.DB()).Create(m)
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating global item.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}

func handleDeleteGlobalItem(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		itemIdStr := mux.Vars(r)["itemId"]
		itemId, err := strconv.ParseUint(itemIdStr, 10, 32)
		if err != nil {
			d.Logger().WithError(err).Errorf("Unable to parse itemId.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		err = NewProcessor(d.Logger(), d.Context(), d.DB()).Delete(uint32(itemId))
		if err != nil {
			d.Logger().WithError(err).Errorf("Deleting global item [%d].", itemId)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
