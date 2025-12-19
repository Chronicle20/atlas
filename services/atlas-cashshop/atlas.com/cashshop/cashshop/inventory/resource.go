package inventory

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

// InitResource initializes the cash inventory resource
func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(si)
			registerInput := rest.RegisterInputHandler[RestModel](l)(si)
			r := router.PathPrefix("/accounts/{accountId}/cash-shop/inventory").Subrouter()
			r.HandleFunc("", registerGet("get_cash_inventory", handleGetCashInventory(db))).Methods(http.MethodGet)
			r.HandleFunc("", registerInput("create_cash_inventory", handleCreateCashInventory(db))).Methods(http.MethodPost)
		}
	}
}

// handleGetCashInventory handles the GET request for a cash inventory
func handleGetCashInventory(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseAccountId(d.Logger(), func(accountId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				// Get the inventory by account ID
				m, err := NewProcessor(d.Logger(), d.Context(), db).GetByAccountId(accountId)
				if errors.Is(err, gorm.ErrRecordNotFound) {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				if err != nil {
					d.Logger().WithError(err).Errorf("Error retrieving cash inventory for account [%d]", accountId)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				// Transform the inventory to a REST model
				res, err := model.Map(Transform)(model.FixedProvider(m))()
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST model.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				// Marshal the response
				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
			}
		})
	}
}

// handleCreateCashInventory handles the POST request for creating a cash inventory
func handleCreateCashInventory(db *gorm.DB) rest.InputHandler[RestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
		return rest.ParseAccountId(d.Logger(), func(accountId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				// Create the inventory
				m, err := NewProcessor(d.Logger(), d.Context(), db).CreateAndEmit(accountId)
				if err != nil {
					d.Logger().WithError(err).Errorf("Error creating cash inventory for account [%d]", accountId)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				// Transform the inventory to a REST model
				res, err := model.Map(Transform)(model.FixedProvider(m))()
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST model.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				// Marshal the response
				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
			}
		})
	}
}