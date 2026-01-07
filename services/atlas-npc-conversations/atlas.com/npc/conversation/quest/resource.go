package quest

import (
	"atlas-npc-conversations/rest"
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

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerHandler := rest.RegisterHandler(l)(db)(si)
			registerInputHandler := rest.RegisterInputHandler[RestModel](l)(db)(si)

			// Register handlers
			router.HandleFunc("/quests/conversations", registerHandler("get_all_quest_conversations", GetAllConversationsHandler)).Methods(http.MethodGet)
			router.HandleFunc("/quests/conversations/{conversationId}", registerHandler("get_quest_conversation", GetConversationHandler)).Methods(http.MethodGet)
			router.HandleFunc("/quests/{questId}/conversation", registerHandler("get_conversation_by_quest", GetConversationByQuestHandler)).Methods(http.MethodGet)
			router.HandleFunc("/quests/conversations", registerInputHandler("create_quest_conversation", CreateConversationHandler)).Methods(http.MethodPost)
			router.HandleFunc("/quests/conversations/{conversationId}", registerInputHandler("update_quest_conversation", UpdateConversationHandler)).Methods(http.MethodPatch)
			router.HandleFunc("/quests/conversations/{conversationId}", registerHandler("delete_quest_conversation", DeleteConversationHandler)).Methods(http.MethodDelete)
			router.HandleFunc("/quests/conversations/seed", registerHandler("seed_quest_conversations", SeedConversationsHandler)).Methods(http.MethodPost)
		}
	}
}

// GetAllConversationsHandler handles GET /quests/conversations
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

// GetConversationHandler handles GET /quests/conversations/{conversationId}
func GetConversationHandler(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseConversationId(d.Logger(), func(conversationId uuid.UUID) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			m, err := NewProcessor(d.Logger(), d.Context(), d.DB()).ByIdProvider(conversationId)()
			if errors.Is(err, gorm.ErrRecordNotFound) {
				d.Logger().WithError(err).Errorf("Quest conversation not found.")
				w.WriteHeader(http.StatusNotFound)
				return
			}
			if err != nil {
				d.Logger().WithError(err).Errorf("Retrieving quest conversation.")
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

// GetConversationByQuestHandler handles GET /quests/{questId}/conversation
func GetConversationByQuestHandler(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseQuestId(d.Logger(), func(questId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			m, err := NewProcessor(d.Logger(), d.Context(), d.DB()).ByQuestIdProvider(questId)()
			if errors.Is(err, gorm.ErrRecordNotFound) {
				d.Logger().WithError(err).Errorf("Quest conversation not found for quest [%d].", questId)
				w.WriteHeader(http.StatusNotFound)
				return
			}
			if err != nil {
				d.Logger().WithError(err).Errorf("Retrieving quest conversation.")
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

// CreateConversationHandler handles POST /quests/conversations
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
			d.Logger().WithError(err).Errorf("Creating quest conversation.")
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

// UpdateConversationHandler handles PATCH /quests/conversations/{conversationId}
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
				d.Logger().WithError(err).Errorf("Updating quest conversation.")
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

// DeleteConversationHandler handles DELETE /quests/conversations/{conversationId}
func DeleteConversationHandler(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseConversationId(d.Logger(), func(conversationId uuid.UUID) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Delete conversation
			err := NewProcessor(d.Logger(), d.Context(), d.DB()).Delete(conversationId)
			if err != nil {
				d.Logger().WithError(err).Errorf("Deleting quest conversation.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// Return success
			w.WriteHeader(http.StatusNoContent)
		}
	})
}

// SeedConversationsHandler handles POST /quests/conversations/seed
func SeedConversationsHandler(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := NewProcessor(d.Logger(), d.Context(), d.DB()).Seed()
		if err != nil {
			d.Logger().WithError(err).Errorf("Seeding quest conversations.")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(result)
	}
}
