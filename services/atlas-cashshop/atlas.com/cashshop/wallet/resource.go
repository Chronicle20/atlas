package wallet

import (
	"atlas-cashshop/rest"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"net/http"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(si)
			r := router.PathPrefix("/accounts/{accountId}/wallet").Subrouter()
			r.HandleFunc("", registerGet("get_wallet", handleGetWallet(db))).Methods(http.MethodGet)
			r.HandleFunc("", rest.RegisterInputHandler[RestModel](l)(si)("create_wallet", handleCreateWallet(db))).Methods(http.MethodPost)
			r.HandleFunc("", rest.RegisterInputHandler[RestModel](l)(si)("update_wallet", handleUpdateWallet(db))).Methods(http.MethodPatch)
		}
	}
}

func handleGetWallet(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseAccountId(d.Logger(), func(accountId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				m, err := NewProcessor(d.Logger(), d.Context(), db).GetByAccountId(accountId)
				if errors.Is(err, gorm.ErrRecordNotFound) {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				res, err := model.Map(Transform)(model.FixedProvider(m))()
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST model.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
			}
		})
	}
}

func handleCreateWallet(db *gorm.DB) rest.InputHandler[RestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
		return rest.ParseAccountId(d.Logger(), func(accountId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				m, err := NewProcessor(d.Logger(), d.Context(), db).CreateAndEmit(accountId, input.Credit, input.Points, input.Prepaid)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				res, err := model.Map(Transform)(model.FixedProvider(m))()
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST model.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
			}
		})
	}
}

func handleUpdateWallet(db *gorm.DB) rest.InputHandler[RestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
		return rest.ParseAccountId(d.Logger(), func(accountId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				m, err := NewProcessor(d.Logger(), d.Context(), db).UpdateAndEmit(accountId, input.Credit, input.Points, input.Prepaid)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				res, err := model.Map(Transform)(model.FixedProvider(m))()
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST model.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
			}
		})
	}
}
