package item

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

// itemIdRangeMin and itemIdRangeMax bound the scripted-item family per PRD §5.
const (
	itemIdRangeMin = 2430000
	itemIdRangeMax = 2439999
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerHandler := rest.RegisterHandler(l)(db)(si)
			registerInputHandler := rest.RegisterInputHandler[RestModel](l)(db)(si)

			// Register handlers
			router.HandleFunc("/items/conversations", registerHandler("get_all_item_conversations", GetAllConversationsHandler)).Methods(http.MethodGet)
			router.HandleFunc("/items/conversations/{conversationId}", registerHandler("get_item_conversation", GetConversationHandler)).Methods(http.MethodGet)
			router.HandleFunc("/items/conversations", registerInputHandler("create_item_conversation", CreateConversationHandler)).Methods(http.MethodPost)
			router.HandleFunc("/items/conversations/{conversationId}", registerInputHandler("update_item_conversation", UpdateConversationHandler)).Methods(http.MethodPatch)
			router.HandleFunc("/items/conversations/{conversationId}", registerHandler("delete_item_conversation", DeleteConversationHandler)).Methods(http.MethodDelete)
		}
	}
}

// GetAllConversationsHandler handles GET /items/conversations
func GetAllConversationsHandler(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, err := paginate.ParseParams(r.URL.Query(), paginate.DefaultPageSize, paginate.MaxPageSize)
		if err != nil {
			server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
			return
		}

		paged, err := NewProcessor(d.Logger(), d.Context(), d.DB()).AllProvider(page)()
		if err != nil {
			d.Logger().WithError(err).Errorf("Retrieving item conversations.")
			server.WriteErrorResponse(d.Logger())(w)(err)
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

// GetConversationHandler handles GET /items/conversations/{conversationId}
func GetConversationHandler(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseConversationId(d.Logger(), func(conversationId uuid.UUID) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			m, err := NewProcessor(d.Logger(), d.Context(), d.DB()).ByIdProvider(conversationId)()
			if errors.Is(err, gorm.ErrRecordNotFound) {
				d.Logger().WithError(err).Errorf("Item conversation not found.")
				w.WriteHeader(http.StatusNotFound)
				return
			}
			if err != nil {
				d.Logger().WithError(err).Errorf("Retrieving item conversation.")
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

// CreateConversationHandler handles POST /items/conversations
func CreateConversationHandler(d *rest.HandlerDependency, c *rest.HandlerContext, rm RestModel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if rm.ItemId < itemIdRangeMin || rm.ItemId > itemIdRangeMax {
			d.Logger().Errorf("Item conversation for item [%d] is outside the scripted-item range %d-%d.", rm.ItemId, itemIdRangeMin, itemIdRangeMax)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

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
			d.Logger().WithError(err).Errorf("Creating item conversation.")
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

// UpdateConversationHandler handles PATCH /items/conversations/{conversationId}
func UpdateConversationHandler(d *rest.HandlerDependency, c *rest.HandlerContext, rm RestModel) http.HandlerFunc {
	return rest.ParseConversationId(d.Logger(), func(conversationId uuid.UUID) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if rm.ItemId < itemIdRangeMin || rm.ItemId > itemIdRangeMax {
				d.Logger().Errorf("Item conversation for item [%d] is outside the scripted-item range %d-%d.", rm.ItemId, itemIdRangeMin, itemIdRangeMax)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

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
				d.Logger().WithError(err).Errorf("Updating item conversation.")
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

// DeleteConversationHandler handles DELETE /items/conversations/{conversationId}
func DeleteConversationHandler(d *rest.HandlerDependency, _ *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseConversationId(d.Logger(), func(conversationId uuid.UUID) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Delete conversation
			err := NewProcessor(d.Logger(), d.Context(), d.DB()).Delete(conversationId)
			if err != nil {
				d.Logger().WithError(err).Errorf("Deleting item conversation.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			// Return success
			w.WriteHeader(http.StatusNoContent)
		}
	})
}
