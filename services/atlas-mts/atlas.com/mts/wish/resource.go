package wish

import (
	"atlas-mts/rest"
	"net/http"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
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

// handleGetCharacterWishlist is a game-capped list (a character's wishlist is
// bounded by game rules) — page[size] defaults to and caps at
// paginate.MaxPageSize (task-117).
func handleGetCharacterWishlist(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			page, perr := paginate.ParseParams(r.URL.Query(), paginate.MaxPageSize, paginate.MaxPageSize)
			if perr != nil {
				server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
				return
			}

			p := NewProcessor(d.Logger(), d.Context(), d.DB())
			// Optional `type` filter (cart/wanted) so the Cart and Wanted views fetch
			// only their own entries; absent, return the full wishlist.
			var paged model.Paged[Model]
			var err error
			if wishType := r.URL.Query().Get("type"); wishType != "" {
				paged, err = p.ByCharacterAndTypePagedProvider(characterId, wishType, page)()
			} else {
				paged, err = p.ByCharacterPagedProvider(characterId, page)()
			}
			if err != nil {
				d.Logger().WithError(err).Errorf("Retrieving wishlist for character [%d].", characterId)
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			res, err := model.SliceMap(Transform)(model.FixedProvider(paged.Items))(model.ParallelMap())()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalPaginatedResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res, paginate.EnvelopeFor(paged), r)
		}
	})
}

// handleGetWorldWishlist returns every want-ad (type=wanted) in a world, across
// all characters — the channel's cross-character Wanted browse tab. The seller
// column is rendered channel-side from each entry's CharacterId. Treated as a
// game-capped list (page[size] defaults to and caps at paginate.MaxPageSize,
// task-117) for uniformity with the character-scoped wishlist.
func handleGetWorldWishlist(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseWorldId(d.Logger(), func(worldId world.Id) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			page, perr := paginate.ParseParams(r.URL.Query(), paginate.MaxPageSize, paginate.MaxPageSize)
			if perr != nil {
				server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
				return
			}

			paged, err := NewProcessor(d.Logger(), d.Context(), d.DB()).WantedByWorldPagedProvider(worldId, page)()
			if err != nil {
				d.Logger().WithError(err).Errorf("Retrieving world wishlist for world [%d].", byte(worldId))
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			res, err := model.SliceMap(Transform)(model.FixedProvider(paged.Items))(model.ParallelMap())()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalPaginatedResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res, paginate.EnvelopeFor(paged), r)
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
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			res, err := Transform(created)
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				server.WriteErrorResponse(d.Logger())(w)(err)
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
			server.WriteErrorResponse(d.Logger())(w)(err)
			return
		}
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
