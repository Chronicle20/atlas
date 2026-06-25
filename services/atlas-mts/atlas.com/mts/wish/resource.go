package wish

import (
	"atlas-mts/rest"
	"net/http"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// InitResource registers the wish-list routes:
//   - GET    /characters/{characterId}/mts/wishlist          — list a character's wishes
//   - POST   /characters/{characterId}/mts/wishlist          — add a wish (JSON:API envelope)
//   - DELETE /characters/{characterId}/mts/wishlist/{wishId} — remove a wish
//   - GET    /worlds/{worldId}/mts/wishlist                  — every want-ad in a world (cross-character)
//
// Wish CRUD touches no custody, so it has no saga and is safe to land here. The
// world-scoped GET backs the channel's cross-character Wanted browse tab.
func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(db)(si)
			registerInput := rest.RegisterInputHandler[RestModel](l)(db)(si)

			r := router.PathPrefix("/characters/{characterId}/mts/wishlist").Subrouter()
			r.HandleFunc("", registerGet("get_character_wishlist", handleGetCharacterWishlist)).Methods(http.MethodGet)
			r.HandleFunc("", registerInput("create_wish", handleCreateWish)).Methods(http.MethodPost)
			r.HandleFunc("/{wishId}", registerGet("delete_wish", handleDeleteWish)).Methods(http.MethodDelete)

			wr := router.PathPrefix("/worlds/{worldId}/mts/wishlist").Subrouter()
			wr.HandleFunc("", registerGet("get_world_wishlist", handleGetWorldWishlist)).Methods(http.MethodGet)
		}
	}
}

func handleGetCharacterWishlist(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			p := NewProcessor(d.Logger(), d.Context(), d.DB())
			// Optional `type` filter (cart/wanted) so the Cart and Wanted views fetch
			// only their own entries; absent, return the full wishlist.
			var ms []Model
			var err error
			if wishType := r.URL.Query().Get("type"); wishType != "" {
				ms, err = p.GetByCharacterAndType(characterId, wishType)
			} else {
				ms, err = p.GetByCharacter(characterId)
			}
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

// handleGetWorldWishlist returns every want-ad (type=wanted) in a world, across
// all characters — the channel's cross-character Wanted browse tab. The seller
// column is rendered channel-side from each entry's CharacterId.
func handleGetWorldWishlist(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseWorldId(d.Logger(), func(worldId world.Id) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ms, err := NewProcessor(d.Logger(), d.Context(), d.DB()).GetWantedByWorld(worldId)
			if err != nil {
				d.Logger().WithError(err).Errorf("Retrieving world wishlist for world [%d].", byte(worldId))
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
			m, err := NewBuilder(t.Id(), characterId, rm.ItemId).
				SetWorldId(world.Id(rm.WorldId)).
				Build()
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
		if _, err := uuid.Parse(wishId); err != nil {
			// A malformed wishId must be rejected, never degraded to a nil-delete
			// (which the tenant callback would scope into a tenant-wide wipe).
			d.Logger().WithError(err).Errorf("Malformed wishId [%s] in delete path.", wishId)
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
