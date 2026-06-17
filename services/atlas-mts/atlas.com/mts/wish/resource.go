package wish

import (
	"atlas-mts/rest"
	"net/http"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// InitResource registers the wish-list routes:
//   - GET    /characters/{characterId}/mts/wishlist          — list a character's wishes
//   - POST   /characters/{characterId}/mts/wishlist          — add a wish (JSON:API envelope)
//   - DELETE /characters/{characterId}/mts/wishlist/{wishId} — remove a wish
//
// Wish CRUD touches no custody, so it has no saga and is safe to land here.
func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(db)(si)
			registerInput := rest.RegisterInputHandler[RestModel](l)(db)(si)

			r := router.PathPrefix("/characters/{characterId}/mts/wishlist").Subrouter()
			r.HandleFunc("", registerGet("get_character_wishlist", handleGetCharacterWishlist)).Methods(http.MethodGet)
			r.HandleFunc("", registerInput("create_wish", handleCreateWish)).Methods(http.MethodPost)
			r.HandleFunc("/{wishId}", registerGet("delete_wish", handleDeleteWish)).Methods(http.MethodDelete)
		}
	}
}

func handleGetCharacterWishlist(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ms, err := NewProcessor(d.Logger(), d.Context(), d.DB()).GetByCharacter(characterId)
			if err != nil {
				d.Logger().WithError(err).Errorf("Retrieving wishlist for character [%d].", characterId)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			res, err := model.SliceMap(Transform)(model.FixedProvider(ms))(model.ParallelMap())()
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

func handleCreateWish(d *rest.HandlerDependency, c *rest.HandlerContext, rm RestModel) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			t := tenant.MustFromContext(d.Context())
			m, err := NewBuilder(t.Id(), characterId, rm.ItemId).Build()
			if err != nil {
				d.Logger().WithError(err).Errorf("Building wish model.")
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			created, err := NewProcessor(d.Logger(), d.Context(), d.DB()).Create(m)
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating wish entry.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			res, err := Transform(created)
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			w.WriteHeader(http.StatusCreated)
			server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
		}
	})
}

func handleDeleteWish(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wishId, ok := mux.Vars(r)["wishId"]
		if !ok || wishId == "" {
			d.Logger().Errorf("Unable to properly parse wishId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		ok, err := NewProcessor(d.Logger(), d.Context(), d.DB()).Delete(wishId)
		if err != nil {
			d.Logger().WithError(err).Errorf("Deleting wish entry [%s].", wishId)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
