package asset

import (
	"atlas-cashshop/rest"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"net/http"
)

// InitResource initializes the asset resource
func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(si)

			r := router.PathPrefix("/accounts/{accountId}/cash-shop/inventory/compartments/{compartmentId}/assets").Subrouter()
			r.HandleFunc("/{assetId}", registerGet("get_asset_by_id", handleGetAssetById(db))).Methods(http.MethodGet)
		}
	}
}

// handleGetAssetById handles the GET request for a specific asset by ID
func handleGetAssetById(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseAccountId(d.Logger(), func(accountId uint32) http.HandlerFunc {
			return rest.ParseCompartmentId(d.Logger(), func(compartmentId uuid.UUID) http.HandlerFunc {
				return rest.ParseAssetId(d.Logger(), func(assetId uuid.UUID) http.HandlerFunc {
					return func(w http.ResponseWriter, r *http.Request) {
						// Create an asset processor
						processor := NewProcessor(d.Logger(), d.Context(), db)

						// Get the asset by ID
						assetProvider := processor.ByIdProvider(assetId)

						// Transform the asset to a REST model
						rm, err := model.Map(Transform)(assetProvider)()
						if err != nil {
							d.Logger().WithError(err).Errorf("Error retrieving asset with ID [%s]", assetId)
							w.WriteHeader(http.StatusNotFound)
							return
						}

						// Verify that the asset belongs to the specified compartment
						if rm.CompartmentId != compartmentId {
							d.Logger().Errorf("Asset with ID [%s] does not belong to compartment with ID [%s]", assetId, compartmentId)
							w.WriteHeader(http.StatusNotFound)
							return
						}

						// Marshal the response
						query := r.URL.Query()
						queryParams := jsonapi.ParseQueryFields(&query)
						server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
					}
				})
			})
		})
	}
}
