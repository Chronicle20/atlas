package family

import (
	"atlas-family/rest"
	"encoding/json"
	"errors"
	"net/http"
	"sort"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// RegisterRoutes registers all family-related REST endpoints
func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {

			// Family management endpoints
			router.HandleFunc("/families/{characterId}/juniors", rest.RegisterInputHandler[AddJuniorRequest](l)(si)("add_junior", addJuniorHandler(db))).Methods(http.MethodPost)
			router.HandleFunc("/families/links/{characterId}", rest.RegisterHandler(l)(si)("break_link", breakLinkHandler(db))).Methods(http.MethodDelete)
			router.HandleFunc("/families/tree/{characterId}", rest.RegisterHandler(l)(si)("get_family_tree", getFamilyTreeHandler(db))).Methods(http.MethodGet)
		}
	}
}

// addJuniorHandler handles POST /families/{characterId}/juniors
func addJuniorHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext, input AddJuniorRequest) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, input AddJuniorRequest) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				// Validate request
				if input.JuniorId == 0 {
					writeErrorResponse(w, http.StatusBadRequest, "Junior ID is required")
					return
				}

				if characterId == input.JuniorId {
					writeErrorResponse(w, http.StatusBadRequest, "Cannot add self as junior")
					return
				}

				// Process the request
				result, err := NewProcessor(d.Logger(), d.Context(), db).AddJuniorAndEmit(uuid.New(), input.WorldId, characterId, input.SeniorLevel, input.JuniorId, input.JuniorLevel)()
				if err != nil {
					d.Logger().WithError(err).Error("Failed to add junior")

					// Map specific errors to HTTP status codes
					switch {
					case errors.Is(err, ErrSeniorNotFound), errors.Is(err, ErrJuniorNotFound), errors.Is(err, ErrMemberNotFound):
						writeErrorResponse(w, http.StatusNotFound, err.Error())
					case errors.Is(err, ErrSeniorHasTooManyJuniors), errors.Is(err, ErrJuniorAlreadyLinked), errors.Is(err, ErrLevelDifferenceTooLarge), errors.Is(err, ErrNotOnSameMap):
						writeErrorResponse(w, http.StatusConflict, err.Error())
					case errors.Is(err, ErrSelfReference):
						writeErrorResponse(w, http.StatusBadRequest, err.Error())
					default:
						server.WriteErrorResponse(d.Logger())(w)(err)
					}
					return
				}

				// Transform to REST model
				restModel, err := Transform(result)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to transform family member to REST model")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[RestFamilyMember](d.Logger())(w)(c.ServerInformation())(queryParams)(restModel)
			}
		})
	}
}

// breakLinkHandler handles DELETE /families/links/{characterId}
//
// The updated member set is bounded (self + senior + juniors + siblings,
// same shape as getFamilyTreeHandler above), materialized in full by
// BreakLinkAndEmit, stable-sorted by CharacterId (unique within the set)
// for determinism, then paginate.Slice applied — same paginated-collection
// envelope as every other route in this task family (task-117 Task 25).
func breakLinkHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				page, err := paginate.ParseParams(r.URL.Query(), paginate.MaxPageSize, paginate.MaxPageSize)
				if err != nil {
					server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
					return
				}

				reason := r.URL.Query().Get("reason")
				if reason == "" {
					reason = "Member requested link break"
				}

				// Process the request
				updatedMembers, err := NewProcessor(d.Logger(), d.Context(), db).BreakLinkAndEmit(uuid.New(), characterId, reason)()
				if err != nil {
					d.Logger().WithError(err).Error("Failed to break family link")
					switch {
					case errors.Is(err, ErrMemberNotFound):
						writeErrorResponse(w, http.StatusNotFound, err.Error())
					case errors.Is(err, ErrNoLinkToBreak):
						writeErrorResponse(w, http.StatusConflict, err.Error())
					default:
						server.WriteErrorResponse(d.Logger())(w)(err)
					}
					return
				}

				sorted := make([]FamilyMember, len(updatedMembers))
				copy(sorted, updatedMembers)
				sort.SliceStable(sorted, func(i, j int) bool {
					return sorted[i].CharacterId() < sorted[j].CharacterId()
				})

				paged := paginate.Slice(sorted, page)

				// Transform to REST models
				rms, err := model.SliceMap(Transform)(model.FixedProvider(paged.Items))(model.ParallelMap())()
				if err != nil {
					d.Logger().WithError(err).Error("Failed to transform family member to REST model")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalPaginatedResponse[[]RestFamilyMember](d.Logger())(w)(c.ServerInformation())(queryParams)(rms, paginate.EnvelopeFor(paged), r)
			}
		})
	}
}

// getFamilyTreeHandler handles GET /families/tree/{characterId}
//
// GetFamilyTree is a bounded graph traversal (self + senior + juniors +
// siblings via several sequential queries), not a single WHERE-filterable
// query, so database.PagedQuery does not apply — same class of adapter gap
// as atlas-keys'/atlas-monster-book's composite-PK fallback (task-117
// Tasks 23/24). It is materialized in full (existing GetFamilyTree,
// unchanged — it has no other internal caller), stable-sorted by
// CharacterId (unique within one tree) for determinism, then paginate.Slice
// applied.
//
// The old response wrapped the member list in a single "familyTrees"
// JSON:API resource (RestFamilyTree{ID, Type, Members}) with Members as a
// plain embedded attribute array — not a top-level JSON:API collection, so
// it could not carry the standard meta/links pagination envelope without a
// bespoke, one-off marshaling path outside the documented paginate toolkit.
// A repo-wide grep found no consumers (backend or otherwise) of that
// wrapper shape, so this task replaces it with the same standard
// []RestFamilyMember paginated-collection response used by every other
// route in this task family (documented deviation, same class as
// atlas-monster-book's hand-rolled-pagination replacement in Task 24).
func getFamilyTreeHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				page, err := paginate.ParseParams(r.URL.Query(), paginate.MaxPageSize, paginate.MaxPageSize)
				if err != nil {
					server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
					return
				}

				// Get family tree
				familyTree, err := NewProcessor(d.Logger(), d.Context(), db).GetFamilyTree(characterId)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to get family tree")
					if errors.Is(err, ErrMemberNotFound) {
						writeErrorResponse(w, http.StatusNotFound, err.Error())
					} else {
						server.WriteErrorResponse(d.Logger())(w)(err)
					}
					return
				}

				sorted := make([]FamilyMember, len(familyTree))
				copy(sorted, familyTree)
				sort.SliceStable(sorted, func(i, j int) bool {
					return sorted[i].CharacterId() < sorted[j].CharacterId()
				})

				paged := paginate.Slice(sorted, page)

				// Transform to REST models
				rms, err := model.SliceMap(Transform)(model.FixedProvider(paged.Items))(model.ParallelMap())()
				if err != nil {
					d.Logger().WithError(err).Error("Failed to transform family tree to REST model")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalPaginatedResponse[[]RestFamilyMember](d.Logger())(w)(c.ServerInformation())(queryParams)(rms, paginate.EnvelopeFor(paged), r)
			}
		})
	}
}

// Helper functions for HTTP responses

// writeErrorResponse writes an error response in JSON format
func writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResponse := map[string]interface{}{
		"error": map[string]interface{}{
			"status": statusCode,
			"title":  http.StatusText(statusCode),
			"detail": message,
		},
	}

	json.NewEncoder(w).Encode(errorResponse)
}
