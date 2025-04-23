package pet

import (
	"atlas-pets/rest"
	"net/http"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(db)(si)
			r := router.PathPrefix("/characters/{characterId}/pets").Subrouter()
			r.HandleFunc("", registerGet("get_pets_for_character", handleGetPetsForCharacter)).Methods(http.MethodGet)
			r = router.PathPrefix("/pets/{petId}").Subrouter()
			r.HandleFunc("", registerGet("get_pet", handleGetPet)).Methods(http.MethodGet)
		}
	}
}

func handleGetPet(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParsePetId(d.Logger(), func(petId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			res, err := model.Map(Transform)(ByIdProvider(d.Context())(d.DB())(petId))()
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

func handleGetPetsForCharacter(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			res, err := model.SliceMap(Transform)(ByOwnerProvider(d.Context())(d.DB())(characterId))(model.ParallelMap())()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
		}
	})
}
