package projection

import (
	"atlas-storage/asset"
	"atlas-storage/rest"
	"net/http"
	"strconv"

	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
)

func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		registerGet := rest.RegisterHandler(l)(si)

		r := router.PathPrefix("/storage/projections").Subrouter()
		r.HandleFunc("/{characterId}", registerGet("get_projection", handleGetProjectionRequest())).Methods(http.MethodGet)
		r.HandleFunc("/{characterId}/compartments/{compartmentType}/assets/{slot}", registerGet("get_projection_asset", handleGetProjectionAssetRequest())).Methods(http.MethodGet)
	}
}

type CharacterIdHandler func(characterId uint32) http.HandlerFunc

func ParseCharacterId(l logrus.FieldLogger, next CharacterIdHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		characterId, err := strconv.Atoi(vars["characterId"])
		if err != nil {
			l.WithError(err).Errorf("Error parsing characterId as uint32")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(uint32(characterId))(w, r)
	}
}

type CompartmentTypeHandler func(compartmentType inventory.Type) http.HandlerFunc

func ParseCompartmentType(l logrus.FieldLogger, next CompartmentTypeHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		compartmentTypeStr := vars["compartmentType"]

		// Try parsing as number first (1-5)
		if compartmentTypeInt, err := strconv.Atoi(compartmentTypeStr); err == nil {
			if compartmentTypeInt >= 1 && compartmentTypeInt <= 5 {
				next(inventory.Type(compartmentTypeInt))(w, r)
				return
			}
		}

		// Try parsing as name (equip, use, setup, etc, cash)
		compartmentType := inventoryTypeFromName(compartmentTypeStr)
		if compartmentType == 0 {
			l.Errorf("Invalid compartment type: %s", compartmentTypeStr)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(compartmentType)(w, r)
	}
}

type SlotHandler func(slot int16) http.HandlerFunc

func ParseSlot(l logrus.FieldLogger, next SlotHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		slot, err := strconv.Atoi(vars["slot"])
		if err != nil {
			l.WithError(err).Errorf("Error parsing slot as int16")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(int16(slot))(w, r)
	}
}

func handleGetProjectionRequest() func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				// Get projection from manager
				proj, ok := GetManager().Get(characterId)
				if !ok {
					d.Logger().Debugf("Projection not found for character [%d]", characterId)
					w.WriteHeader(http.StatusNotFound)
					return
				}

				// Transform to REST model
				restModel, err := Transform(proj)
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to transform projection for character [%d]", characterId)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(restModel)
			}
		})
	}
}

func handleGetProjectionAssetRequest() func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return ParseCompartmentType(d.Logger(), func(compartmentType inventory.Type) http.HandlerFunc {
				return ParseSlot(d.Logger(), func(slot int16) http.HandlerFunc {
					return func(w http.ResponseWriter, r *http.Request) {
						// Get projection from manager
						proj, ok := GetManager().Get(characterId)
						if !ok {
							d.Logger().Debugf("Projection not found for character [%d]", characterId)
							w.WriteHeader(http.StatusNotFound)
							return
						}

						// Get asset by slot
						assetModel, ok := proj.GetAssetBySlot(compartmentType, slot)
						if !ok {
							d.Logger().Debugf("Asset not found at slot [%d] in compartment [%d] for character [%d]",
								slot, compartmentType, characterId)
							w.WriteHeader(http.StatusNotFound)
							return
						}

						// Transform to REST model
						restModel, err := asset.Transform(assetModel)
						if err != nil {
							d.Logger().WithError(err).Errorf("Unable to transform asset for character [%d]", characterId)
							w.WriteHeader(http.StatusInternalServerError)
							return
						}

						query := r.URL.Query()
						queryParams := jsonapi.ParseQueryFields(&query)
						server.MarshalResponse[asset.RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(restModel)
					}
				})
			})
		})
	}
}
