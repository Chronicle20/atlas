package gachapon

import (
	"atlas-reward-pools/rest"
	"errors"
	"net/http"

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

			r := router.PathPrefix("/gachapons").Subrouter()
			r.HandleFunc("", registerGet("get_all_gachapons", handleGetAllGachapons)).Methods(http.MethodGet)
			r.HandleFunc("", registerInput("create_gachapon", handleCreateGachapon)).Methods(http.MethodPost)
			r.HandleFunc("/{gachaponId}", registerGet("get_gachapon", handleGetGachapon)).Methods(http.MethodGet)
			r.HandleFunc("/{gachaponId}", registerInput("update_gachapon", handleUpdateGachapon)).Methods(http.MethodPatch)
			r.HandleFunc("/{gachaponId}", registerGet("delete_gachapon", handleDeleteGachapon)).Methods(http.MethodDelete)
		}
	}
}

func handleGetAllGachapons(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, err := paginate.ParseParams(r.URL.Query(), paginate.DefaultPageSize, paginate.MaxPageSize)
		if err != nil {
			server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
			return
		}

		paged, err := NewProcessor(d.Logger(), d.Context(), d.DB()).GetAll(page)()
		if err != nil {
			d.Logger().WithError(err).Errorf("Retrieving all gachapons.")
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

func handleGetGachapon(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseGachaponId(d.Logger(), func(gachaponId string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			m, err := NewProcessor(d.Logger(), d.Context(), d.DB()).GetById(gachaponId)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				d.Logger().WithError(err).Errorf("Retrieving gachapon [%s].", gachaponId)
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			rm, err := Transform(m)
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
		}
	})
}

func handleCreateGachapon(d *rest.HandlerDependency, c *rest.HandlerContext, rm RestModel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t := tenant.MustFromContext(d.Context())
		b := NewBuilder(t.Id(), rm.Id).
			SetName(rm.Name).
			SetNpcIds(rm.NpcIds).
			SetCommonWeight(rm.CommonWeight).
			SetUncommonWeight(rm.UncommonWeight).
			SetRareWeight(rm.RareWeight)
		// Kind is optional on the wire: an absent/empty "kind" attribute must
		// preserve the builder's "gachapon" default rather than blank it out.
		if rm.Kind != "" {
			b = b.SetKind(rm.Kind)
		}
		m, err := b.Build()
		if err != nil {
			d.Logger().WithError(err).Errorf("Building gachapon model.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		err = NewProcessor(d.Logger(), d.Context(), d.DB()).Create(m)
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating gachapon.")
			server.WriteErrorResponse(d.Logger())(w)(err)
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}

func handleUpdateGachapon(d *rest.HandlerDependency, c *rest.HandlerContext, rm RestModel) http.HandlerFunc {
	return rest.ParseGachaponId(d.Logger(), func(gachaponId string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			err := NewProcessor(d.Logger(), d.Context(), d.DB()).Update(gachaponId, rm.Name, rm.NpcIds, rm.CommonWeight, rm.UncommonWeight, rm.RareWeight)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				d.Logger().WithError(err).Errorf("Updating gachapon [%s].", gachaponId)
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		}
	})
}

func handleDeleteGachapon(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseGachaponId(d.Logger(), func(gachaponId string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			err := NewProcessor(d.Logger(), d.Context(), d.DB()).Delete(gachaponId)
			if err != nil {
				d.Logger().WithError(err).Errorf("Deleting gachapon [%s].", gachaponId)
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		}
	})
}
