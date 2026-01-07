package npc

import (
	"atlas-npc-conversations/conversation"
	"atlas-npc-conversations/rest"
	"encoding/json"
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
			registerHandler := rest.RegisterHandler(l)(db)(si)
			registerInputHandler := rest.RegisterInputHandler[RestModel](l)(db)(si)

			// Register handlers
			router.HandleFunc("/npcs/conversations", registerHandler("get_all_conversations", GetAllConversationsHandler)).Methods(http.MethodGet)
			router.HandleFunc("/npcs/conversations/{conversationId}", registerHandler("get_conversation", GetConversationHandler)).Methods(http.MethodGet)
			router.HandleFunc("/npcs/{npcId}/conversations", registerHandler("get_conversations_by_npc", GetConversationsByNpcHandler)).Methods(http.MethodGet)
			router.HandleFunc("/npcs/conversations", registerInputHandler("create_conversation", CreateConversationHandler)).Methods(http.MethodPost)
			router.HandleFunc("/npcs/conversations/{conversationId}", registerInputHandler("update_conversation", UpdateConversationHandler)).Methods(http.MethodPatch)
			router.HandleFunc("/npcs/conversations/{conversationId}", registerHandler("delete_conversation", DeleteConversationHandler)).Methods(http.MethodDelete)
			router.HandleFunc("/npcs/conversations/validate", registerInputHandler("validate_conversation", ValidateConversationHandler)).Methods(http.MethodPost)
			router.HandleFunc("/npcs/conversations/seed", registerHandler("seed_conversations", SeedConversationsHandler)).Methods(http.MethodPost)
		}
	}
}

// GetAllConversationsHandler handles GET /conversations
func GetAllConversationsHandler(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mp := NewProcessor(d.Logger(), d.Context(), d.DB()).AllProvider()
		rm, err := model.SliceMap(Transform)(mp)(model.ParallelMap())()
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating REST model.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		query := r.URL.Query()
		queryParams := jsonapi.ParseQueryFields(&query)
		server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
	}
}

// GetConversationHandler handles GET /conversations/{conversationId}
func GetConversationHandler(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseConversationId(d.Logger(), func(conversationId uuid.UUID) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			m, err := NewProcessor(d.Logger(), d.Context(), d.DB()).ByIdProvider(conversationId)()
			if errors.Is(err, gorm.ErrRecordNotFound) {
				d.Logger().WithError(err).Errorf("Conversation not found.")
				w.WriteHeader(http.StatusNotFound)
				return
			}
			if err != nil {
				d.Logger().WithError(err).Errorf("Retrieving conversation.")
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
}

// CreateConversationHandler handles POST /conversations
func CreateConversationHandler(d *rest.HandlerDependency, c *rest.HandlerContext, rm RestModel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract domain model from REST model
		m, err := Extract(rm)
		if err != nil {
			d.Logger().WithError(err).Errorf("Extracting domain model from REST model.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Create conversation
		createdModel, err := NewProcessor(d.Logger(), d.Context(), d.DB()).Create(m)
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating conversation.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Transform back to REST model
		createdRm, err := Transform(createdModel)
		if err != nil {
			d.Logger().WithError(err).Errorf("Transforming domain model to REST model.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Return created conversation
		query := r.URL.Query()
		queryParams := jsonapi.ParseQueryFields(&query)
		w.WriteHeader(http.StatusCreated)
		server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(createdRm)
	}
}

// UpdateConversationHandler handles PUT /conversations/{conversationId}
func UpdateConversationHandler(d *rest.HandlerDependency, c *rest.HandlerContext, rm RestModel) http.HandlerFunc {
	return rest.ParseConversationId(d.Logger(), func(conversationId uuid.UUID) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Extract domain model from REST model
			m, err := Extract(rm)
			if err != nil {
				d.Logger().WithError(err).Errorf("Extracting domain model from REST model.")
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			// Update conversation
			updatedModel, err := NewProcessor(d.Logger(), d.Context(), d.DB()).Update(conversationId, m)
			if err != nil {
				d.Logger().WithError(err).Errorf("Updating conversation.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// Transform back to REST model
			updatedRm, err := Transform(updatedModel)
			if err != nil {
				d.Logger().WithError(err).Errorf("Transforming domain model to REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// Return updated conversation
			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(updatedRm)
		}
	})
}

// DeleteConversationHandler handles DELETE /conversations/{conversationId}
func DeleteConversationHandler(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseConversationId(d.Logger(), func(conversationId uuid.UUID) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Delete conversation
			err := NewProcessor(d.Logger(), d.Context(), d.DB()).Delete(conversationId)
			if err != nil {
				d.Logger().WithError(err).Errorf("Deleting conversation.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// Return success
			w.WriteHeader(http.StatusNoContent)
		}
	})
}

// GetConversationsByNpcHandler handles GET /npcs/{npcId}/conversations
func GetConversationsByNpcHandler(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseNpcId(d.Logger(), func(npcId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			mp := NewProcessor(d.Logger(), d.Context(), d.DB()).AllByNpcIdProvider(npcId)
			rm, err := model.SliceMap(Transform)(mp)(model.ParallelMap())()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
		}
	})
}

// RestValidationError represents a validation error in REST format
type RestValidationError struct {
	StateId   string `json:"stateId,omitempty"`
	Field     string `json:"field"`
	ErrorType string `json:"errorType"`
	Message   string `json:"message"`
}

// RestValidationResult represents the validation result in REST format
type RestValidationResult struct {
	Valid  bool                  `json:"valid"`
	Errors []RestValidationError `json:"errors,omitempty"`
}

// ValidateConversationHandler handles POST /conversations/validate
func ValidateConversationHandler(d *rest.HandlerDependency, c *rest.HandlerContext, rm RestModel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract domain model from REST model
		m, err := Extract(rm)
		if err != nil {
			d.Logger().WithError(err).Errorf("Extracting domain model from REST model.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Validate conversation
		validator := conversation.NewValidator()
		result := validator.ValidateNpc(m)

		// Transform to REST validation result
		restErrors := make([]RestValidationError, len(result.Errors))
		for i, err := range result.Errors {
			restErrors[i] = RestValidationError{
				StateId:   err.StateId,
				Field:     err.Field,
				ErrorType: err.ErrorType,
				Message:   err.Message,
			}
		}

		restResult := RestValidationResult{
			Valid:  result.Valid,
			Errors: restErrors,
		}

		// Marshal and send response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Use standard JSON marshaling since this is not a JSON:API resource
		jsonData, err := jsonapi.Marshal(restResult)
		if err != nil {
			d.Logger().WithError(err).Errorf("Marshaling validation result.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write(jsonData)
	}
}

// SeedConversationsHandler handles POST /npcs/conversations/seed
func SeedConversationsHandler(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := NewProcessor(d.Logger(), d.Context(), d.DB()).Seed()
		if err != nil {
			d.Logger().WithError(err).Errorf("Seeding conversations.")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(result)
	}
}
