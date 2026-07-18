package character

import (
	"atlas-invites/invite"
	"atlas-invites/rest"
	"net/http"
	"sort"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
)

const (
	GetCharacterInvites = "get_character_invites"
)

func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		registerGet := rest.RegisterHandler(l)(si)
		r := router.PathPrefix("/characters").Subrouter()
		r.HandleFunc("/{characterId}/invites", registerGet(GetCharacterInvites, handleGetCharacterInvites)).Methods(http.MethodGet)
	}
}

// handleGetCharacterInvites is registry-backed (Redis-indexed, invite.GetRegistry().GetForCharacter),
// not a DB query, so database.PagedQuery does not apply. GetByCharacterId/
// ByCharacterIdProvider have no other internal caller, so they are left
// unpaged/unchanged and materialized in full here, stable-sorted by Id
// (unique) for determinism — the registry index lookup order is not
// guaranteed — then paginate.Slice applied. invite.Transform is a pure
// field copy (no live remaining-duration/computed decoration beyond the
// raw registry read itself), so no extra decoration stage is needed.
func handleGetCharacterInvites(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			page, err := paginate.ParseParams(r.URL.Query(), paginate.MaxPageSize, paginate.MaxPageSize)
			if err != nil {
				server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
				return
			}

			is, err := invite.NewProcessor(d.Logger(), d.Context()).GetByCharacterId(characterId)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			sorted := make([]invite.Model, len(is))
			copy(sorted, is)
			sort.SliceStable(sorted, func(i, j int) bool {
				return sorted[i].Id() < sorted[j].Id()
			})

			paged := paginate.Slice(sorted, page)

			res, err := model.SliceMap(invite.Transform)(model.FixedProvider(paged.Items))()()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// Marshal response
			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalPaginatedResponse[[]invite.RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res, paginate.EnvelopeFor(paged), r)
		}
	})
}
