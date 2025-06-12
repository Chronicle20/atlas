package compartment

import (
	"atlas-cashshop/rest"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"net/http"
)

// InitResource initializes the cash compartment resource
func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(si)
			r := router.PathPrefix("/accounts/{accountId}/cash-shop/inventory/compartments").Subrouter()
			r.HandleFunc("", registerGet("get_cash_compartments", handleGetCompartments(db))).Methods(http.MethodGet).Queries("type", "{type}")
		}
	}
}

// handleGetCompartments handles the GET request for cash compartments
func handleGetCompartments(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseAccountId(d.Logger(), func(accountId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				// Get the compartment type from the query parameter
				query := r.URL.Query()
				typeParam := query.Get("type")

				// Create a compartment processor
				processor := NewProcessor(d.Logger(), d.Context(), db)

				// If a type parameter was provided, get only that compartment
				if typeParam != "" {
					compartmentType := CompartmentType(typeParam)

					// Get the compartment by account ID and type
					compartmentProvider := processor.ByAccountIdAndTypeProvider(accountId, compartmentType)

					// Transform the compartment to a REST model
					res, err := model.Map(Transform)(compartmentProvider)()
					if err != nil {
						d.Logger().WithError(err).Errorf("Creating REST model.")
						w.WriteHeader(http.StatusInternalServerError)
						return
					}

					// Marshal the response
					queryParams := jsonapi.ParseQueryFields(&query)
					server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
					return
				}

				// If no type parameter was provided, get all compartments
				res, err := model.SliceMap(Transform)(processor.AllByAccountIdProvider(accountId))(model.ParallelMap())()
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST model.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				// Marshal the response
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
			}
		})
	}
}
