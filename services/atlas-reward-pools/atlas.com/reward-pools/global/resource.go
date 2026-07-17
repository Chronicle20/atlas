package global

import (
	"atlas-reward-pools/rest"
	"errors"
	"net/http"
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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
			r.HandleFunc("/{itemId}", registerInput("update_global_item", handleUpdateGlobalItem)).Methods(http.MethodPatch)
			r.HandleFunc("/{itemId}", registerGet("delete_global_item", handleDeleteGlobalItem)).Methods(http.MethodDelete)
		}
	}
}

func handleGetAllGlobalItems(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tier := r.URL.Query().Get("tier")

		page, err := paginate.ParseParams(r.URL.Query(), paginate.DefaultPageSize, paginate.MaxPageSize)
		if err != nil {
			server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
			return
		}

		var paged model.Paged[Model]
		p := NewProcessor(d.Logger(), d.Context(), d.DB())
		if tier != "" {
			paged, err = p.GetByTierPaged(tier, page)()
		} else {
			paged, err = p.GetAll(page)()
		}
		if err != nil {
			d.Logger().WithError(err).Errorf("Retrieving global items.")
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
			server.WriteErrorResponse(d.Logger())(w)(err)
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}

func handleUpdateGlobalItem(d *rest.HandlerDependency, c *rest.HandlerContext, rm RestModel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		itemIdStr := mux.Vars(r)["itemId"]
		itemId, err := strconv.ParseUint(itemIdStr, 10, 32)
		if err != nil {
			d.Logger().WithError(err).Errorf("Unable to parse itemId.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		err = NewProcessor(d.Logger(), d.Context(), d.DB()).Update(uint32(itemId), rm.ItemId, rm.Quantity, rm.Tier)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			if errors.Is(err, ErrInvalidTier) {
				server.WriteBadRequest(d.Logger(), w, err.Error())
				return
			}
			d.Logger().WithError(err).Errorf("Updating global item [%d].", itemId)
			server.WriteErrorResponse(d.Logger())(w)(err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
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
			server.WriteErrorResponse(d.Logger())(w)(err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
