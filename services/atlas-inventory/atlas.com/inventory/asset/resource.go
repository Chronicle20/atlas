package asset

import (
	"atlas-inventory/rest"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/google/uuid"
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
			r := router.PathPrefix("/characters/{characterId}/inventory/compartments/{compartmentId}/assets").Subrouter()
			r.HandleFunc("", registerGet("get_assets", handleGetAssets(db))).Methods(http.MethodGet)
			r.HandleFunc("/{assetId}", registerGet("delete_asset", handleDeleteAsset(db))).Methods(http.MethodDelete)
		}
	}
}

func handleGetAssets(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return rest.ParseCompartmentId(d.Logger(), func(compartmentId uuid.UUID) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					ms, err := NewProcessor(d.Logger(), d.Context(), db).GetByCompartmentId(compartmentId)
					if err != nil {
						w.WriteHeader(http.StatusInternalServerError)
						return
					}

					rm, err := model.SliceMap(Transform)(model.FixedProvider(ms))(model.ParallelMap())()
					if err != nil {
						d.Logger().WithError(err).Errorf("Creating REST model.")
						w.WriteHeader(http.StatusInternalServerError)
						return
					}

					query := r.URL.Query()
					queryParams := jsonapi.ParseQueryFields(&query)
					server.MarshalResponse[[]BaseRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
				}
			})
		})
	}
}

func handleDeleteAsset(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return rest.ParseCompartmentId(d.Logger(), func(compartmentId uuid.UUID) http.HandlerFunc {
				return rest.ParseAssetId(d.Logger(), func(assetId uint32) http.HandlerFunc {
					return func(w http.ResponseWriter, r *http.Request) {
						err := NewProcessor(d.Logger(), d.Context(), db).DeleteAndEmit(uuid.New(), characterId, compartmentId, assetId)
						if err != nil {
							d.Logger().WithError(err).Errorf("Unable to delete asset [%d].", assetId)
							w.WriteHeader(http.StatusInternalServerError)
							return
						}
						w.WriteHeader(http.StatusNoContent)
					}
				})
			})
		})
	}
}
