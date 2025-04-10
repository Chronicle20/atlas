package compartment

import (
	"atlas-inventory/rest"
	"errors"
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
			r := router.PathPrefix("/characters/{characterId}/inventory/compartments").Subrouter()
			r.HandleFunc("/{compartmentId}", registerGet("get_compartment", handleGetCompartment(db))).Methods(http.MethodGet)
		}
	}
}

func handleGetCompartment(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return rest.ParseCompartmentId(d.Logger(), func(compartmentId uuid.UUID) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					m, err := NewProcessor(d.Logger(), d.Context(), db).GetById(compartmentId)
					if errors.Is(err, gorm.ErrRecordNotFound) {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					if err != nil {
						w.WriteHeader(http.StatusInternalServerError)
						return
					}

					rm, err := model.Map(Transform)(model.FixedProvider(m))()
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
		})
	}
}
