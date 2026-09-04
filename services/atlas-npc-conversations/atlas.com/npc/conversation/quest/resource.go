package quest

import (
	"atlas-npc-conversations/rest"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerHandler := rest.RegisterHandler(l)(si)
			registerInputHandler := rest.RegisterInputHandler[RestModel](l)(si)

			// Register handlers
			router.HandleFunc("/quests/conversations", registerHandler("get_all_quest_conversations", GetAllConversationsHandler(db))).Methods(http.MethodGet)
			router.HandleFunc("/quests/conversations/{conversationId}", registerHandler("get_quest_conversation", GetConversationHandler(db))).Methods(http.MethodGet)
			router.HandleFunc("/quests/{questId}/conversation", registerHandler("get_conversation_by_quest", GetConversationByQuestHandler(db))).Methods(http.MethodGet)
			router.HandleFunc("/quests/conversations", registerInputHandler("create_quest_conversation", CreateConversationHandler(db))).Methods(http.MethodPost)
			router.HandleFunc("/quests/conversations/{conversationId}", registerInputHandler("update_quest_conversation", UpdateConversationHandler(db))).Methods(http.MethodPatch)
			router.HandleFunc("/quests/conversations/{conversationId}", registerHandler("delete_quest_conversation", DeleteConversationHandler(db))).Methods(http.MethodDelete)
		}
	}
}

// GetAllConversationsHandler handles GET /quests/conversations
func GetAllConversationsHandler(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			page, err := paginate.ParseParams(r.URL.Query(), paginate.DefaultPageSize, paginate.MaxPageSize)
			if err != nil {
				server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
				return
			}

			paged, err := NewProcessor(d.Logger(), d.Context(), db).AllProvider(page)()
			if err != nil {
				d.Logger().WithError(err).Errorf("Retrieving quest conversations.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			rm, err := model.SliceMap(Transform)(model.FixedProvider(paged.Items))(model.ParallelMap())()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalPaginatedResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm, paginate.EnvelopeFor(paged), r)
		}
	}
}

// GetConversationHandler handles GET /quests/conversations/{conversationId}
func GetConversationHandler(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseConversationId(d.Logger(), func(conversationId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				m, err := NewProcessor(d.Logger(), d.Context(), db).ByIdProvider(conversationId)()
				if errors.Is(err, gorm.ErrRecordNotFound) {
					d.Logger().WithError(err).Errorf("Quest conversation not found.")
					w.WriteHeader(http.StatusNotFound)
					return
				}
				if err != nil {
					d.Logger().WithError(err).Errorf("Retrieving quest conversation.")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}
				rm, err := model.Map(Transform)(model.FixedProvider(m))()
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST model.")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
			}
		})
	}
}

// GetConversationByQuestHandler handles GET /quests/{questId}/conversation
func GetConversationByQuestHandler(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseQuestId(d.Logger(), func(questId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				m, err := NewProcessor(d.Logger(), d.Context(), db).ByQuestIdProvider(questId)()
				if errors.Is(err, gorm.ErrRecordNotFound) {
					d.Logger().WithError(err).Errorf("Quest conversation not found for quest [%d].", questId)
					w.WriteHeader(http.StatusNotFound)
					return
				}
				if err != nil {
					d.Logger().WithError(err).Errorf("Retrieving quest conversation.")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}
				rm, err := model.Map(Transform)(model.FixedProvider(m))()
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST model.")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
			}
		})
	}
}

// CreateConversationHandler handles POST /quests/conversations
func CreateConversationHandler(db *gorm.DB) rest.InputHandler[RestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, rm RestModel) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Extract domain model from REST model
			m, err := Extract(rm)
			if err != nil {
				d.Logger().WithError(err).Errorf("Extracting domain model from REST model.")
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			// Create conversation
			createdModel, err := NewProcessor(d.Logger(), d.Context(), db).Create(m)
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating quest conversation.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			// Transform back to REST model
			createdRm, err := Transform(createdModel)
			if err != nil {
				d.Logger().WithError(err).Errorf("Transforming domain model to REST model.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			// Return created conversation
			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			w.WriteHeader(http.StatusCreated)
			server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(createdRm)
		}
	}
}

// UpdateConversationHandler handles PATCH /quests/conversations/{conversationId}
func UpdateConversationHandler(db *gorm.DB) rest.InputHandler[RestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, rm RestModel) http.HandlerFunc {
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
				updatedModel, err := NewProcessor(d.Logger(), d.Context(), db).Update(conversationId, m)
				if err != nil {
					d.Logger().WithError(err).Errorf("Updating quest conversation.")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				// Transform back to REST model
				updatedRm, err := Transform(updatedModel)
				if err != nil {
					d.Logger().WithError(err).Errorf("Transforming domain model to REST model.")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				// Return updated conversation
				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(updatedRm)
			}
		})
	}
}

// DeleteConversationHandler handles DELETE /quests/conversations/{conversationId}
func DeleteConversationHandler(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, _ *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseConversationId(d.Logger(), func(conversationId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				// Delete conversation
				err := NewProcessor(d.Logger(), d.Context(), db).Delete(conversationId)
				if err != nil {
					d.Logger().WithError(err).Errorf("Deleting quest conversation.")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				// Return success
				w.WriteHeader(http.StatusNoContent)
			}
		})
	}
}
