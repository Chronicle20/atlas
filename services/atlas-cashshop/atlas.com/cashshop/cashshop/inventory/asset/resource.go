package asset

import (
	"atlas-cashshop/rest"
	"errors"
	"net/http"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(si)
			registerInput := rest.RegisterInputHandler[RestModel](l)(si)

			r := router.PathPrefix("/accounts/{accountId}/cash-shop/inventory/compartments/{compartmentId}/assets").Subrouter()
			r.HandleFunc("/{assetId}", registerGet("get_asset_by_id", handleGetAssetById(db))).Methods(http.MethodGet)

			ir := router.PathPrefix("/cash-shop/assets").Subrouter()
			ir.HandleFunc("", registerInput("create_asset", handleCreateAsset(db))).Methods(http.MethodPost)
			ir.HandleFunc("/{assetId}", registerGet("get_asset", handleGetAsset(db))).Methods(http.MethodGet)
			ir.HandleFunc("/{assetId}", registerInput("update_asset", handleUpdateAsset(db))).Methods(http.MethodPatch)
			ir.HandleFunc("/{assetId}", registerGet("delete_asset", handleDeleteAsset(db))).Methods(http.MethodDelete)
		}
	}
}

func handleGetAssetById(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseAccountId(d.Logger(), func(accountId uint32) http.HandlerFunc {
			return rest.ParseCompartmentId(d.Logger(), func(compartmentId uuid.UUID) http.HandlerFunc {
				return rest.ParseAssetId(d.Logger(), func(assetId uint32) http.HandlerFunc {
					return func(w http.ResponseWriter, r *http.Request) {
						processor := NewProcessor(d.Logger(), d.Context(), db)

						rm, err := model.Map(Transform)(processor.ByIdProvider(assetId))()
						if err != nil {
							d.Logger().WithError(err).Errorf("Error retrieving asset with ID [%d]", assetId)
							w.WriteHeader(http.StatusNotFound)
							return
						}

						query := r.URL.Query()
						queryParams := jsonapi.ParseQueryFields(&query)
						server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
					}
				})
			})
		})
	}
}

func handleGetAsset(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseAssetId(d.Logger(), func(assetId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				ms, err := NewProcessor(d.Logger(), d.Context(), db).GetById(assetId)
				if errors.Is(err, gorm.ErrRecordNotFound) {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				res, err := model.Map(Transform)(model.FixedProvider(ms))()
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

func handleCreateAsset(db *gorm.DB) rest.InputHandler[RestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, i RestModel) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			im, err := Extract(i)
			if err != nil {
				d.Logger().WithError(err).Errorf("Extracting model.")
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			m, err := NewProcessor(d.Logger(), d.Context(), db).CreateAndEmit(im.CompartmentId(), im.TemplateId(), im.CommodityId(), im.Quantity(), im.PurchasedBy())
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating asset.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			restModel, err := Transform(m)
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(restModel)
		}
	}
}

func handleUpdateAsset(db *gorm.DB) rest.InputHandler[RestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
		return rest.ParseAssetId(d.Logger(), func(assetId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				err := NewProcessor(d.Logger(), d.Context(), db).UpdateQuantity(assetId, input.Quantity)
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to update asset [%d].", assetId)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			}
		})
	}
}

func handleDeleteAsset(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseAssetId(d.Logger(), func(assetId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				err := NewProcessor(d.Logger(), d.Context(), db).DeleteAndEmit(assetId)
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to delete asset [%d].", assetId)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			}
		})
	}
}
