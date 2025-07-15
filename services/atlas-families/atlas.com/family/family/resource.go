package family

import (
	"atlas-family/rest"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
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
						writeErrorResponse(w, http.StatusInternalServerError, "Internal server error")
					}
					return
				}

				// Transform to REST model
				restModel, err := Transform(result)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to transform family member to REST model")
					writeErrorResponse(w, http.StatusInternalServerError, "Internal server error")
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
func breakLinkHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
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
						writeErrorResponse(w, http.StatusInternalServerError, "Internal server error")
					}
					return
				}

				// Transform to REST models
				rms, err := model.SliceMap(Transform)(model.FixedProvider(updatedMembers))(model.ParallelMap())()
				if err != nil {
					d.Logger().WithError(err).Error("Failed to transform family member to REST model")
					writeErrorResponse(w, http.StatusInternalServerError, "Internal server error")
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[[]RestFamilyMember](d.Logger())(w)(c.ServerInformation())(queryParams)(rms)
			}
		})
	}
}

// getFamilyTreeHandler handles GET /families/tree/{characterId}
func getFamilyTreeHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				// Get family tree
				familyTree, err := NewProcessor(d.Logger(), d.Context(), db).GetFamilyTree(characterId)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to get family tree")
					if errors.Is(err, ErrMemberNotFound) {
						writeErrorResponse(w, http.StatusNotFound, err.Error())
					} else {
						writeErrorResponse(w, http.StatusInternalServerError, "Internal server error")
					}
					return
				}

				// Transform to REST model
				restTree, err := TransformFamilyTree(familyTree)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to transform family tree to REST model")
					writeErrorResponse(w, http.StatusInternalServerError, "Internal server error")
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[RestFamilyTree](d.Logger())(w)(c.ServerInformation())(queryParams)(restTree)
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
