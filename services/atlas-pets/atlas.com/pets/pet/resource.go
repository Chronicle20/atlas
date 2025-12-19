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
			r.HandleFunc("", rest.RegisterInputHandler[RestModel](l)(db)(si)("create_for_character", handleCreate)).Methods(http.MethodPost)
			r = router.PathPrefix("/pets").Subrouter()
			r.HandleFunc("", rest.RegisterInputHandler[RestModel](l)(db)(si)("create", handleCreate)).Methods(http.MethodPost)
			r.HandleFunc("/{petId}", registerGet("get_pet", handleGetPet)).Methods(http.MethodGet)
		}
	}
}

func handleGetPet(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParsePetId(d.Logger(), func(petId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			p := NewProcessor(d.Logger(), d.Context(), d.DB())
			res, err := model.Map(Transform)(p.ByIdProvider(petId))()
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
			p := NewProcessor(d.Logger(), d.Context(), d.DB())
			res, err := model.SliceMap(Transform)(p.ByOwnerProvider(characterId))(model.ParallelMap())()
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

func handleCreate(d *rest.HandlerDependency, c *rest.HandlerContext, i RestModel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := NewProcessor(d.Logger(), d.Context(), d.DB())
		ip, err := model.Map(Extract)(model.FixedProvider(i))()
		if err != nil {
			d.Logger().WithError(err).Errorf("Unable to create model from input.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		pm, err := p.CreateAndEmit(ip)
		if err != nil {
			d.Logger().WithError(err).Errorf("Unable to create model.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		res, err := model.Map(Transform)(model.FixedProvider(pm))()
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating REST model.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		query := r.URL.Query()
		queryParams := jsonapi.ParseQueryFields(&query)
		server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
	}
}
