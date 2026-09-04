package script

import (
	"atlas-portal-actions/rest"
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
			router.HandleFunc("/portals/scripts", registerHandler("get_all_scripts", GetAllScriptsHandler(db))).Methods(http.MethodGet)
			router.HandleFunc("/portals/scripts/{scriptId}", registerHandler("get_script", GetScriptHandler(db))).Methods(http.MethodGet)
			router.HandleFunc("/portals/{portalId}/scripts", registerHandler("get_scripts_by_portal", GetScriptsByPortalHandler(db))).Methods(http.MethodGet)
			router.HandleFunc("/portals/scripts", registerInputHandler("create_script", CreateScriptHandler(db))).Methods(http.MethodPost)
			router.HandleFunc("/portals/scripts/{scriptId}", registerInputHandler("update_script", UpdateScriptHandler(db))).Methods(http.MethodPatch)
			router.HandleFunc("/portals/scripts/{scriptId}", registerHandler("delete_script", DeleteScriptHandler(db))).Methods(http.MethodDelete)
		}
	}
}

// GetAllScriptsHandler handles GET /portals/scripts
func GetAllScriptsHandler(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			page, err := paginate.ParseParams(r.URL.Query(), paginate.DefaultPageSize, paginate.MaxPageSize)
			if err != nil {
				server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
				return
			}

			paged, err := NewProcessor(d.Logger(), d.Context(), db).AllProvider(page)()
			if err != nil {
				d.Logger().WithError(err).Errorf("Retrieving scripts.")
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

// GetScriptHandler handles GET /portals/scripts/{scriptId}
func GetScriptHandler(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseScriptId(d.Logger(), func(scriptId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				m, err := NewProcessor(d.Logger(), d.Context(), db).ByIdProvider(scriptId)()
				if errors.Is(err, gorm.ErrRecordNotFound) {
					d.Logger().WithError(err).Errorf("Script not found.")
					w.WriteHeader(http.StatusNotFound)
					return
				}
				if err != nil {
					d.Logger().WithError(err).Errorf("Retrieving script.")
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

// GetScriptsByPortalHandler handles GET /portals/{portalId}/scripts
func GetScriptsByPortalHandler(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParsePortalId(d.Logger(), func(portalId string) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				m, err := NewProcessor(d.Logger(), d.Context(), db).ByPortalIdProvider(portalId)()
				if errors.Is(err, gorm.ErrRecordNotFound) {
					d.Logger().WithField("portalId", portalId).Debugf("No script configured for portal.")
					w.WriteHeader(http.StatusNotFound)
					return
				}
				if err != nil {
					d.Logger().WithError(err).Errorf("Retrieving script for portal [%s].", portalId)
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

// CreateScriptHandler handles POST /portals/scripts
func CreateScriptHandler(db *gorm.DB) rest.InputHandler[RestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, rm RestModel) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Extract domain model from REST model
			m, err := Extract(rm)
			if err != nil {
				d.Logger().WithError(err).Errorf("Extracting domain model from REST model.")
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			// Create script
			createdModel, err := NewProcessor(d.Logger(), d.Context(), db).Create(m)
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating script.")
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

			// Return created script
			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			w.WriteHeader(http.StatusCreated)
			server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(createdRm)
		}
	}
}

// UpdateScriptHandler handles PATCH /portals/scripts/{scriptId}
func UpdateScriptHandler(db *gorm.DB) rest.InputHandler[RestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, rm RestModel) http.HandlerFunc {
		return rest.ParseScriptId(d.Logger(), func(scriptId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				// Extract domain model from REST model
				m, err := Extract(rm)
				if err != nil {
					d.Logger().WithError(err).Errorf("Extracting domain model from REST model.")
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				// Update script
				updatedModel, err := NewProcessor(d.Logger(), d.Context(), db).Update(scriptId, m)
				if err != nil {
					d.Logger().WithError(err).Errorf("Updating script.")
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

				// Return updated script
				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(updatedRm)
			}
		})
	}
}

// DeleteScriptHandler handles DELETE /portals/scripts/{scriptId}
func DeleteScriptHandler(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, _ *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseScriptId(d.Logger(), func(scriptId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				// Delete script
				err := NewProcessor(d.Logger(), d.Context(), db).Delete(scriptId)
				if err != nil {
					d.Logger().WithError(err).Errorf("Deleting script.")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				// Return success
				w.WriteHeader(http.StatusNoContent)
			}
		})
	}
}
