package gachapon

import (
	"atlas-gachapons/rest"
	"errors"
	"net/http"

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
		ms, err := NewProcessor(d.Logger(), d.Context(), d.DB()).GetAll()()
		if err != nil {
			d.Logger().WithError(err).Errorf("Retrieving all gachapons.")
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
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			rm, err := Transform(m)
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
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
		m, err := NewBuilder(t.Id(), rm.Id).
			SetName(rm.Name).
			SetNpcIds(rm.NpcIds).
			SetCommonWeight(rm.CommonWeight).
			SetUncommonWeight(rm.UncommonWeight).
			SetRareWeight(rm.RareWeight).
			Build()
		if err != nil {
			d.Logger().WithError(err).Errorf("Building gachapon model.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		err = NewProcessor(d.Logger(), d.Context(), d.DB()).Create(m)
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating gachapon.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}

func handleUpdateGachapon(d *rest.HandlerDependency, c *rest.HandlerContext, rm RestModel) http.HandlerFunc {
	return rest.ParseGachaponId(d.Logger(), func(gachaponId string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			err := NewProcessor(d.Logger(), d.Context(), d.DB()).Update(gachaponId, rm.Name, rm.CommonWeight, rm.UncommonWeight, rm.RareWeight)
			if err != nil {
				d.Logger().WithError(err).Errorf("Updating gachapon [%s].", gachaponId)
				w.WriteHeader(http.StatusInternalServerError)
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
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		}
	})
}
