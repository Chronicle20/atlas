package script

import (
	"atlas-map-actions/rest"
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

			// Register handlers - specific routes before parameterized routes
			router.HandleFunc("/maps/actions/seed", registerHandler("seed_scripts", SeedScriptsHandler)).Methods(http.MethodPost)
			router.HandleFunc("/maps/actions", registerHandler("get_all_scripts", GetAllScriptsHandler)).Methods(http.MethodGet)
			router.HandleFunc("/maps/actions", registerInputHandler("create_script", CreateScriptHandler)).Methods(http.MethodPost)
			router.HandleFunc("/maps/actions/{scriptId}", registerHandler("get_script", GetScriptHandler)).Methods(http.MethodGet)
			router.HandleFunc("/maps/actions/{scriptId}", registerInputHandler("update_script", UpdateScriptHandler)).Methods(http.MethodPatch)
			router.HandleFunc("/maps/actions/{scriptId}", registerHandler("delete_script", DeleteScriptHandler)).Methods(http.MethodDelete)
			router.HandleFunc("/maps/{scriptName}/actions", registerHandler("get_scripts_by_name", GetScriptsByNameHandler)).Methods(http.MethodGet)
		}
	}
}

// GetAllScriptsHandler handles GET /maps/actions
func GetAllScriptsHandler(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
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

// GetScriptHandler handles GET /maps/actions/{scriptId}
func GetScriptHandler(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseScriptId(d.Logger(), func(scriptId uuid.UUID) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			m, err := NewProcessor(d.Logger(), d.Context(), d.DB()).ByIdProvider(scriptId)()
			if errors.Is(err, gorm.ErrRecordNotFound) {
				d.Logger().WithError(err).Errorf("Script not found.")
				w.WriteHeader(http.StatusNotFound)
				return
			}
			if err != nil {
				d.Logger().WithError(err).Errorf("Retrieving script.")
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

// GetScriptsByNameHandler handles GET /maps/{scriptName}/actions
func GetScriptsByNameHandler(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseScriptName(d.Logger(), func(scriptName string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			mp := NewProcessor(d.Logger(), d.Context(), d.DB()).ByScriptNameProvider(scriptName)
			rm, err := model.SliceMap(Transform)(mp)(model.ParallelMap())()
			if err != nil {
				d.Logger().WithError(err).Errorf("Retrieving scripts for [%s].", scriptName)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
		}
	})
}

// CreateScriptHandler handles POST /maps/actions
func CreateScriptHandler(d *rest.HandlerDependency, c *rest.HandlerContext, rm RestModel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m, err := Extract(rm)
		if err != nil {
			d.Logger().WithError(err).Errorf("Extracting domain model from REST model.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		createdModel, err := NewProcessor(d.Logger(), d.Context(), d.DB()).Create(m)
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating script.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		createdRm, err := Transform(createdModel)
		if err != nil {
			d.Logger().WithError(err).Errorf("Transforming domain model to REST model.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		query := r.URL.Query()
		queryParams := jsonapi.ParseQueryFields(&query)
		w.WriteHeader(http.StatusCreated)
		server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(createdRm)
	}
}

// UpdateScriptHandler handles PATCH /maps/actions/{scriptId}
func UpdateScriptHandler(d *rest.HandlerDependency, c *rest.HandlerContext, rm RestModel) http.HandlerFunc {
	return rest.ParseScriptId(d.Logger(), func(scriptId uuid.UUID) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			m, err := Extract(rm)
			if err != nil {
				d.Logger().WithError(err).Errorf("Extracting domain model from REST model.")
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			updatedModel, err := NewProcessor(d.Logger(), d.Context(), d.DB()).Update(scriptId, m)
			if err != nil {
				d.Logger().WithError(err).Errorf("Updating script.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			updatedRm, err := Transform(updatedModel)
			if err != nil {
				d.Logger().WithError(err).Errorf("Transforming domain model to REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(updatedRm)
		}
	})
}

// DeleteScriptHandler handles DELETE /maps/actions/{scriptId}
func DeleteScriptHandler(d *rest.HandlerDependency, _ *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseScriptId(d.Logger(), func(scriptId uuid.UUID) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			err := NewProcessor(d.Logger(), d.Context(), d.DB()).Delete(scriptId)
			if err != nil {
				d.Logger().WithError(err).Errorf("Deleting script.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusNoContent)
		}
	})
}

// SeedScriptsHandler handles POST /maps/actions/seed
func SeedScriptsHandler(d *rest.HandlerDependency, _ *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := NewProcessor(d.Logger(), d.Context(), d.DB()).Seed()
		if err != nil {
			d.Logger().WithError(err).Errorf("Seeding scripts.")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(result)
	}
}
