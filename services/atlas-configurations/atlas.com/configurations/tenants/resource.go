package tenants

import (
	"atlas-configurations/data"
	"atlas-configurations/drift"
	"atlas-configurations/rest"
	"atlas-configurations/scope"
	"atlas-configurations/templates"
	"atlas-configurations/tenants/characters/preset"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
	tenantlib "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			r := router.PathPrefix("/configurations/tenants").Subrouter()
			r.HandleFunc("", rest.RegisterHandler(l)(si)("get_configuration_tenants", handleGetConfigurationTenants(db))).Methods(http.MethodGet)
			r.HandleFunc("", rest.RegisterInputHandler[RestModel](l)(si)("create_configuration_tenant", handleCreateConfigurationTenant(db))).Methods(http.MethodPost)
			r.HandleFunc("/{tenantId}", rest.RegisterHandler(l)(si)("get_configuration_tenant", handleGetConfigurationTenant(db))).Methods(http.MethodGet)
			r.HandleFunc("/{tenantId}", rest.RegisterInputHandler[RestModel](l)(si)("update_configuration_tenant", handleUpdateConfigurationTenant(db))).Methods(http.MethodPatch)
			r.HandleFunc("/{tenantId}", rest.RegisterHandler(l)(si)("delete_configuration_tenant", handleDeleteConfigurationTenant(db))).Methods(http.MethodDelete)
			r.HandleFunc("/{tenantId}/reset", rest.RegisterHandler(l)(si)("reset_configuration_tenant", handleResetConfigurationTenant(db))).Methods(http.MethodPost)
		}
	}
}

// viewProcessor is the read/reset processor: the ordinary processor with
// the templates processor attached, so a baseline is resolvable. The
// write paths (Create, UpdateById, DeleteById) deliberately do NOT need
// it -- mirrors templates/resource.go:37-42.
func viewProcessor(d *rest.HandlerDependency, db *gorm.DB) Processor {
	return NewProcessor(d.Logger(), d.Context(), db).
		WithTemplates(templates.NewProcessor(d.Logger(), d.Context(), db))
}

// writeJSONAPIError emits the same document shape validationFailureError
// renders, for the statuses server.WriteErrorResponse cannot express (it
// maps everything to 500/503). Copied from templates/resource.go:188-202.
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

// resetRequest is the reset endpoint's optional body. An absent body,
// `{}`, an absent `sections` key and `sections: []` are all equivalent
// and mean "every comparable section" (FR-4.2).
type resetRequest struct {
	Data struct {
		Attributes struct {
			Sections []string `json:"sections"`
		} `json:"attributes"`
	} `json:"data"`
}

func parseResetSections(r *http.Request) ([]string, error) {
	var body resetRequest
	err := json.NewDecoder(r.Body).Decode(&body)
	if errors.Is(err, io.EOF) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return body.Data.Attributes.Sections, nil
}

func handleGetConfigurationTenants(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			page, err := paginate.ParseParams(r.URL.Query(), paginate.DefaultPageSize, paginate.MaxPageSize)
			if err != nil {
				server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
				return
			}

			paged, err := viewProcessor(d, db).AllViewProvider(page)()
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to get configuration tenants.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalPaginatedResponse[[]ViewRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(paged.Items, paginate.EnvelopeFor(paged), r)
		}
	}
}

func handleGetConfigurationTenant(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				cts, err := viewProcessor(d, db).ViewByIdProvider(tenantId)()
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to get configuration tenants.")
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

func handleUpdateConfigurationTenant(db *gorm.DB) rest.InputHandler[RestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				// atlas-configurations is a bootstrap-tier service that does
				// not require tenant headers on incoming requests, so the
				// request ctx has no tenant. The validator's atlas-data calls
				// are tenant-scoped; for the tenant PATCH path we synthesize
				// a tenant context from {URL tenantId, body region/major/minor}
				// so the validator can fully run R-6..R-12. The template path
				// has no equivalent identity and skips those rules instead.
				ctx := d.Context()
				if t, terr := tenantlib.Create(tenantId, input.Region, input.MajorVersion, input.MinorVersion); terr == nil {
					ctx = tenantlib.WithContext(ctx, t)
				} else {
					d.Logger().WithError(terr).Warn("Unable to construct tenant model from PATCH input; preset validation will skip atlas-data lookups.")
				}
				p := NewProcessor(d.Logger(), ctx, db).
					WithValidator(preset.NewValidator(data.NewProcessor(d.Logger())))
				err := p.UpdateById(tenantId, input)
				if err != nil {
					var ve *validationFailureError
					if errors.As(err, &ve) {
						w.Header().Set("Content-Type", "application/vnd.api+json")
						w.WriteHeader(http.StatusBadRequest)
						_ = json.NewEncoder(w).Encode(map[string]any{"errors": ve.AsJSONAPIErrors()})
						return
					}
					d.Logger().WithError(err).Errorf("Unable to update configuration tenant.")
					rest.WriteErrorResponse(d.Logger())(w)(err)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			}
		})
	}
}

func handleCreateConfigurationTenant(db *gorm.DB) rest.InputHandler[RestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			tenantId, err := NewProcessor(d.Logger(), d.Context(), db).Create(input)
			if err != nil {
				var ve *validationFailureError
				if errors.As(err, &ve) {
					w.Header().Set("Content-Type", "application/vnd.api+json")
					w.WriteHeader(http.StatusBadRequest)
					_ = json.NewEncoder(w).Encode(map[string]any{"errors": ve.AsJSONAPIErrors()})
					return
				}
				d.Logger().WithError(err).Errorf("Unable to create configuration tenant.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			// Read back through the view provider so POST returns exactly
			// what a subsequent GET returns -- same attributes, same
			// computed drift. It also means the onboarding flow can
			// assert FR-5.2 from the create response.
			view, err := viewProcessor(d, db).ViewByIdProvider(tenantId)()
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to read back created configuration tenant.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			// Set the Location header to the URL of the newly created resource
			w.Header().Set("Location", "/configurations/tenants/"+tenantId.String())

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			w.WriteHeader(http.StatusCreated)
			server.MarshalResponse[ViewRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(view)
		}
	}
}

func handleDeleteConfigurationTenant(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				err := NewProcessor(d.Logger(), d.Context(), db).DeleteById(tenantId)
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to delete configuration tenant.")
					rest.WriteErrorResponse(d.Logger())(w)(err)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			}
		})
	}
}

func handleResetConfigurationTenant(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				sections, err := parseResetSections(r)
				if err != nil {
					writeJSONAPIError(w, http.StatusBadRequest, "malformed request body", "The reset request body could not be decoded: "+err.Error())
					return
				}

				// The validator's atlas-data calls are tenant-scoped and
				// atlas-configurations takes no tenant headers, so
				// synthesize a tenant context exactly as the PATCH path
				// does (resource.go:81-93) -- from the URL tenant id and
				// the STORED row's region/version, which the processor
				// re-reads. Without it the atlas-data-backed preset rules
				// silently skip.
				ctx := d.Context()
				stored, gErr := NewProcessor(d.Logger(), ctx, db).GetById(tenantId)
				if gErr == nil {
					if t, terr := tenantlib.Create(tenantId, stored.Region, stored.MajorVersion, stored.MinorVersion); terr == nil {
						ctx = tenantlib.WithContext(ctx, t)
					} else {
						d.Logger().WithError(terr).Warn("Unable to construct tenant model for reset; preset validation will skip atlas-data lookups.")
					}
				}

				p := NewProcessor(d.Logger(), ctx, db).
					WithTemplates(templates.NewProcessor(d.Logger(), ctx, db)).
					WithValidator(preset.NewValidator(data.NewProcessor(d.Logger())))

				view, err := p.ResetById(tenantId, sections)
				if err == nil {
					query := r.URL.Query()
					queryParams := jsonapi.ParseQueryFields(&query)
					server.MarshalResponse[ViewRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(view)
					return
				}

				switch {
				case errors.Is(err, drift.ErrUnknownSection):
					writeJSONAPIError(w, http.StatusBadRequest, "unknown section", err.Error())
				case errors.Is(err, scope.ErrCrossEnvironmentWrite):
					writeJSONAPIError(w, http.StatusForbidden, "cross-environment write", err.Error())
				case errors.Is(err, ErrTenantNotFound):
					writeJSONAPIError(w, http.StatusNotFound, "tenant not found", "No configuration tenant exists with id "+tenantId.String()+".")
				case errors.Is(err, ErrNoBaselineTemplate):
					writeJSONAPIError(w, http.StatusConflict, "no baseline template", "No configuration template resolves for this tenant's region and version, so there is nothing to reset to.")
				default:
					var ve *validationFailureError
					if errors.As(err, &ve) {
						// 422, not the PATCH path's 400: a validation
						// failure here means "the server's own baseline
						// is unprocessable", not "your body is bad". The
						// request was fine.
						w.Header().Set("Content-Type", "application/vnd.api+json")
						w.WriteHeader(http.StatusUnprocessableEntity)
						_ = json.NewEncoder(w).Encode(map[string]any{"errors": ve.AsJSONAPIErrors()})
						return
					}
					d.Logger().WithError(err).Errorf("Unable to reset configuration tenant.")
					server.WriteErrorResponse(d.Logger())(w)(err)
				}
			}
		})
	}
}
