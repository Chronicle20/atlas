package templates

import (
	"atlas-configurations/data"
	"atlas-configurations/rest"
	"atlas-configurations/templates/characters/preset"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			r := router.PathPrefix("/configurations/templates").Subrouter()
			r.HandleFunc("", rest.RegisterInputHandler[RestModel](l)(si)("create_configuration_template", handleCreateConfigurationTemplate(db))).Methods(http.MethodPost)
			r.HandleFunc("", rest.RegisterHandler(l)(si)("get_configuration_template", handleGetConfigurationTemplate(db))).Methods(http.MethodGet).Queries("region", "{region}", "majorVersion", "{majorVersion}", "minorVersion", "{minorVersion}")
			r.HandleFunc("", rest.RegisterHandler(l)(si)("get_configuration_templates", handleGetConfigurationTemplates(db))).Methods(http.MethodGet)
			r.HandleFunc("/{templateId}", rest.RegisterHandler(l)(si)("get_configuration_template", handleGetConfigurationTemplateById(db))).Methods(http.MethodGet)
			r.HandleFunc("/{templateId}", rest.RegisterInputHandler[RestModel](l)(si)("update_configuration_template", handleUpdateConfigurationTemplate(db))).Methods(http.MethodPatch)
			r.HandleFunc("/{templateId}", rest.RegisterHandler(l)(si)("delete_configuration_template", handleDeleteConfigurationTemplate(db))).Methods(http.MethodDelete)
			r.HandleFunc("/{templateId}/reseed", rest.RegisterHandler(l)(si)("reseed_configuration_template", handleReseedConfigurationTemplate(db))).Methods(http.MethodPost)
		}
	}
}

// viewProcessor is the read/re-seed processor: the ordinary processor with the
// shipped-template catalog attached, so drift is computable. The write paths
// (create/update) deliberately do NOT need it.
func viewProcessor(d *rest.HandlerDependency, db *gorm.DB) Processor {
	return NewProcessor(d.Logger(), d.Context(), db).WithCatalog(ShippedCatalog())
}

func handleCreateConfigurationTemplate(db *gorm.DB) rest.InputHandler[RestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			templateId, err := NewProcessor(d.Logger(), d.Context(), db).Create(input)
			if err != nil {
				var ve *validationFailureError
				if errors.As(err, &ve) {
					w.Header().Set("Content-Type", "application/vnd.api+json")
					w.WriteHeader(http.StatusBadRequest)
					_ = json.NewEncoder(w).Encode(map[string]any{"errors": ve.AsJSONAPIErrors()})
					return
				}
				d.Logger().WithError(err).Errorf("Unable to create configuration template.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			// Read back through the view provider so POST returns exactly
			// what a subsequent GET returns - same attributes, same
			// computed revisions (design D3).
			view, err := viewProcessor(d, db).ViewByIdProvider(templateId)()
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to read back created configuration template.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			// Set the Location header to the URL of the newly created resource
			w.Header().Set("Location", "/configurations/templates/"+templateId.String())

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			w.WriteHeader(http.StatusCreated)
			server.MarshalResponse[ViewRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(view)
		}
	}
}

func handleGetConfigurationTemplate(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseRegion(d.Logger(), func(region string) http.HandlerFunc {
			return rest.ParseMajorVersion(d.Logger(), func(majorVersion uint16) http.HandlerFunc {
				return rest.ParseMinorVersion(d.Logger(), func(minorVersion uint16) http.HandlerFunc {
					return func(w http.ResponseWriter, r *http.Request) {
						cts, err := viewProcessor(d, db).ViewByRegionAndVersionProvider(region, majorVersion, minorVersion)()
						if err != nil {
							d.Logger().WithError(err).Errorf("Unable to get configuration templates.")
							server.WriteErrorResponse(d.Logger())(w)(err)
							return
						}

						query := r.URL.Query()
						queryParams := jsonapi.ParseQueryFields(&query)
						server.MarshalResponse[ViewRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(cts)
					}
				})
			})
		})
	}
}

func handleGetConfigurationTemplates(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			page, err := paginate.ParseParams(r.URL.Query(), paginate.DefaultPageSize, paginate.MaxPageSize)
			if err != nil {
				server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
				return
			}

			paged, err := viewProcessor(d, db).AllViewProvider(page)()
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to get configuration templates.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalPaginatedResponse[[]ViewRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(paged.Items, paginate.EnvelopeFor(paged), r)
		}
	}
}

func handleGetConfigurationTemplateById(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTemplateId(d.Logger(), func(templateId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				cts, err := viewProcessor(d, db).ViewByIdProvider(templateId)()
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to get configuration templates.")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[ViewRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(cts)
			}
		})
	}
}

func handleUpdateConfigurationTemplate(db *gorm.DB) rest.InputHandler[RestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
		return rest.ParseTemplateId(d.Logger(), func(templateId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				p := NewProcessor(d.Logger(), d.Context(), db).
					WithValidator(preset.NewValidator(data.NewProcessor(d.Logger())))
				err := p.UpdateById(templateId, input)
				if err != nil {
					var ve *validationFailureError
					if errors.As(err, &ve) {
						w.Header().Set("Content-Type", "application/vnd.api+json")
						w.WriteHeader(http.StatusBadRequest)
						_ = json.NewEncoder(w).Encode(map[string]any{"errors": ve.AsJSONAPIErrors()})
						return
					}
					d.Logger().WithError(err).Errorf("Unable to update configuration template.")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			}
		})
	}
}

func handleDeleteConfigurationTemplate(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTemplateId(d.Logger(), func(templateId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				err := NewProcessor(d.Logger(), d.Context(), db).DeleteById(templateId)
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to delete configuration template.")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			}
		})
	}
}

// writeJSONAPIError emits the same document shape validationFailureError
// renders, for the statuses server.WriteErrorResponse cannot express (it maps
// everything to 500/503). Keeps the re-seed endpoint's 404 and 409 consistent
// with the existing 400s.
func writeJSONAPIError(w http.ResponseWriter, status int, title string, detail string) {
	w.Header().Set("Content-Type", "application/vnd.api+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"errors": []map[string]any{{
			"status": strconv.Itoa(status),
			"title":  title,
			"detail": detail,
		}},
	})
}

func handleReseedConfigurationTemplate(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTemplateId(d.Logger(), func(templateId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				err := viewProcessor(d, db).ReseedById(templateId)
				if err == nil {
					w.WriteHeader(http.StatusNoContent)
					return
				}

				switch {
				case errors.Is(err, ErrTemplateNotFound):
					writeJSONAPIError(w, http.StatusNotFound, "template not found", "No configuration template exists with id "+templateId.String()+".")
				case errors.Is(err, ErrNoShippedTemplate):
					writeJSONAPIError(w, http.StatusConflict, "no shipped template", "This image ships no seed file for the template's region and version, so there is nothing to reset to.")
				default:
					var ve *validationFailureError
					if errors.As(err, &ve) {
						// A broken seed file: CI-guarded, so this should not
						// occur. Rendered identically to create/update
						// validation failures.
						w.Header().Set("Content-Type", "application/vnd.api+json")
						w.WriteHeader(http.StatusBadRequest)
						_ = json.NewEncoder(w).Encode(map[string]any{"errors": ve.AsJSONAPIErrors()})
						return
					}
					d.Logger().WithError(err).Errorf("Unable to re-seed configuration template.")
					server.WriteErrorResponse(d.Logger())(w)(err)
				}
			}
		})
	}
}
