package definition

import (
	"atlas-party-quests/rest"
	"encoding/json"
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
			registerHandler := rest.RegisterHandler(l)(db)(si)
			registerInputHandler := rest.RegisterInputHandler[RestModel](l)(db)(si)

			router.HandleFunc("/party-quests/definitions", registerHandler("get_all_definitions", GetAllDefinitionsHandler)).Methods(http.MethodGet)
			router.HandleFunc("/party-quests/definitions/{definitionId}", registerHandler("get_definition", GetDefinitionHandler)).Methods(http.MethodGet)
			router.HandleFunc("/party-quests/definitions/quest/{questId}", registerHandler("get_definition_by_quest_id", GetDefinitionByQuestIdHandler)).Methods(http.MethodGet)
			router.HandleFunc("/party-quests/definitions", registerInputHandler("create_definition", CreateDefinitionHandler)).Methods(http.MethodPost)
			router.HandleFunc("/party-quests/definitions/{definitionId}", registerInputHandler("update_definition", UpdateDefinitionHandler)).Methods(http.MethodPatch)
			router.HandleFunc("/party-quests/definitions/{definitionId}", registerHandler("delete_definition", DeleteDefinitionHandler)).Methods(http.MethodDelete)
			router.HandleFunc("/party-quests/definitions/validate", registerHandler("validate_definitions", ValidateDefinitionsHandler)).Methods(http.MethodPost)
		}
	}
}

func GetAllDefinitionsHandler(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, err := paginate.ParseParams(r.URL.Query(), paginate.DefaultPageSize, paginate.MaxPageSize)
		if err != nil {
			server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
			return
		}

		paged, err := NewProcessor(d.Logger(), d.Context(), d.DB()).AllProvider(page)()
		if err != nil {
			d.Logger().WithError(err).Errorf("Retrieving definitions.")
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

func GetDefinitionHandler(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseDefinitionId(d.Logger(), func(definitionId uuid.UUID) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			m, err := NewProcessor(d.Logger(), d.Context(), d.DB()).ByIdProvider(definitionId)()
			if errors.Is(err, gorm.ErrRecordNotFound) {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			if err != nil {
				d.Logger().WithError(err).Errorf("Retrieving definition.")
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

func GetDefinitionByQuestIdHandler(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseQuestId(d.Logger(), func(questId string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			m, err := NewProcessor(d.Logger(), d.Context(), d.DB()).ByQuestIdProvider(questId)()
			if errors.Is(err, gorm.ErrRecordNotFound) {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			if err != nil {
				d.Logger().WithError(err).Errorf("Retrieving definition by quest ID.")
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

func CreateDefinitionHandler(d *rest.HandlerDependency, c *rest.HandlerContext, rm RestModel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m, err := Extract(rm)
		if err != nil {
			d.Logger().WithError(err).Errorf("Extracting domain model.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		createdModel, err := NewProcessor(d.Logger(), d.Context(), d.DB()).Create(m)
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating definition.")
			server.WriteErrorResponse(d.Logger())(w)(err)
			return
		}

		createdRm, err := Transform(createdModel)
		if err != nil {
			d.Logger().WithError(err).Errorf("Transforming domain model.")
			server.WriteErrorResponse(d.Logger())(w)(err)
			return
		}

		query := r.URL.Query()
		queryParams := jsonapi.ParseQueryFields(&query)
		w.WriteHeader(http.StatusCreated)
		server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(createdRm)
	}
}

func UpdateDefinitionHandler(d *rest.HandlerDependency, c *rest.HandlerContext, rm RestModel) http.HandlerFunc {
	return rest.ParseDefinitionId(d.Logger(), func(definitionId uuid.UUID) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			m, err := Extract(rm)
			if err != nil {
				d.Logger().WithError(err).Errorf("Extracting domain model.")
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			updatedModel, err := NewProcessor(d.Logger(), d.Context(), d.DB()).Update(definitionId, m)
			if err != nil {
				d.Logger().WithError(err).Errorf("Updating definition.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			updatedRm, err := Transform(updatedModel)
			if err != nil {
				d.Logger().WithError(err).Errorf("Transforming domain model.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(updatedRm)
		}
	})
}

func DeleteDefinitionHandler(d *rest.HandlerDependency, _ *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseDefinitionId(d.Logger(), func(definitionId uuid.UUID) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			err := NewProcessor(d.Logger(), d.Context(), d.DB()).Delete(definitionId)
			if err != nil {
				d.Logger().WithError(err).Errorf("Deleting definition.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			w.WriteHeader(http.StatusNoContent)
		}
	})
}

func ValidateDefinitionsHandler(d *rest.HandlerDependency, _ *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := NewProcessor(d.Logger(), r.Context(), d.DB())
		results := p.ValidateDefinitions()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(results)
	}
}
