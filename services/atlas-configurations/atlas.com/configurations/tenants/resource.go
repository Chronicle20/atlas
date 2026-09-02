package tenants

import (
	"atlas-configurations/data"
	"atlas-configurations/drift"
	"atlas-configurations/rest"
	"atlas-configurations/scope"
	"atlas-configurations/templates"
	"atlas-configurations/tenants/characters/preset"
	"bytes"
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
			r.HandleFunc("/{tenantId}/reset", normalizeResetBody(rest.RegisterInputHandler[ResetRestModel](l)(si)("reset_configuration_tenant", handleResetConfigurationTenant(db)))).Methods(http.MethodPost)
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

// normalizeResetBody rewrites an absent or `{}` reset request body into the
// canonical empty `tenants` envelope before rest.RegisterInputHandler's
// jsonapi.Unmarshal decode step. jsonapi.Unmarshal treats a zero-length body
// and `{}` as decode errors, but FR-4.2 requires both (and an absent
// `sections` key, and `sections: []`) to mean "reset every comparable
// section" -- so they must reach the handler as a successfully decoded,
// empty ResetRestModel rather than fail at the framework's generic decode
// step, which writes a bare, non-JSON:API-shaped 400.
//
// A body that is present but not valid JSON is rejected here, in the same
// errors-array shape every other error response on this endpoint uses, for
// the same reason: jsonapi.Unmarshal's own decode-failure path can't.
//
// A body that is valid JSON but is not a JSON:API envelope -- missing a
// top-level "data" object or a "data.type" -- is rejected here too, for the
// same reason. FR-4.2 only defines "reset everything" for an absent or `{}`
// body; it does not extend that meaning to an arbitrary envelope-less JSON
// object, so this case stays a 400 rather than being folded into the
// "reset everything" normalization above. Left unhandled, it would fall
// through to jsonapi.Unmarshal's own "Source JSON is empty and has no
// \"attributes\" payload object" error, which the framework surfaces as a
// bare, content-type-less 400 -- the one input class on this endpoint that
// would escape the vnd.api+json errors-array shape every other error uses.
func normalizeResetBody(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		_ = r.Body.Close()
		if err != nil {
			w.Header().Set("Content-Type", "application/vnd.api+json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"errors": []map[string]any{{
				"status": strconv.Itoa(http.StatusBadRequest),
				"title":  "malformed request body",
				"detail": "The reset request body could not be read: " + err.Error(),
			}}})
			return
		}

		trimmed := bytes.TrimSpace(raw)
		switch {
		case len(trimmed) == 0, string(trimmed) == "{}":
			trimmed = []byte(`{"data":{"type":"tenants","attributes":{}}}`)
		case !json.Valid(trimmed):
			w.Header().Set("Content-Type", "application/vnd.api+json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"errors": []map[string]any{{
				"status": strconv.Itoa(http.StatusBadRequest),
				"title":  "malformed request body",
				"detail": "The reset request body could not be decoded.",
			}}})
			return
		case !hasResetEnvelope(trimmed):
			w.Header().Set("Content-Type", "application/vnd.api+json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"errors": []map[string]any{{
				"status": strconv.Itoa(http.StatusBadRequest),
				"title":  "malformed request body",
				"detail": "The reset request body must be a JSON:API document with a top-level \"data\" object naming a \"type\".",
			}}})
			return
		}

		r.Body = io.NopCloser(bytes.NewReader(trimmed))
		r.ContentLength = int64(len(trimmed))
		next(w, r)
	}
}

// hasResetEnvelope reports whether raw is a JSON:API document with a
// top-level "data" object naming a non-empty "type" -- the minimum
// jsonapi.Unmarshal needs to reach setDataIntoTarget instead of failing at
// its own "Source JSON is empty" check. It does not validate the type name
// or attributes; RegisterInputHandler's decode step and preset validation
// still do that work.
func hasResetEnvelope(raw []byte) bool {
	var doc struct {
		Data *struct {
			Type string `json:"type"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return false
	}
	return doc.Data != nil && doc.Data.Type != ""
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

func handleResetConfigurationTenant(db *gorm.DB) rest.InputHandler[ResetRestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, input ResetRestModel) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				// The validator's atlas-data calls are tenant-scoped and
				// atlas-configurations takes no tenant headers, so
				// synthesize a tenant context exactly as the PATCH path
				// does (resource.go:152-157) -- from the URL tenant id and
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

				view, err := p.ResetById(tenantId, input.Sections)
				if err == nil {
					query := r.URL.Query()
					queryParams := jsonapi.ParseQueryFields(&query)
					server.MarshalResponse[ViewRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(view)
					return
				}

				switch {
				case errors.Is(err, drift.ErrUnknownSection):
					w.Header().Set("Content-Type", "application/vnd.api+json")
					w.WriteHeader(http.StatusBadRequest)
					_ = json.NewEncoder(w).Encode(map[string]any{"errors": []map[string]any{{
						"status": strconv.Itoa(http.StatusBadRequest),
						"title":  "unknown section",
						"detail": err.Error(),
					}}})
				case errors.Is(err, scope.ErrCrossEnvironmentWrite):
					w.Header().Set("Content-Type", "application/vnd.api+json")
					w.WriteHeader(http.StatusForbidden)
					_ = json.NewEncoder(w).Encode(map[string]any{"errors": []map[string]any{{
						"status": strconv.Itoa(http.StatusForbidden),
						"title":  "cross-environment write",
						"detail": err.Error(),
					}}})
				case errors.Is(err, ErrTenantNotFound):
					w.Header().Set("Content-Type", "application/vnd.api+json")
					w.WriteHeader(http.StatusNotFound)
					_ = json.NewEncoder(w).Encode(map[string]any{"errors": []map[string]any{{
						"status": strconv.Itoa(http.StatusNotFound),
						"title":  "tenant not found",
						"detail": "No configuration tenant exists with id " + tenantId.String() + ".",
					}}})
				case errors.Is(err, ErrNoBaselineTemplate):
					w.Header().Set("Content-Type", "application/vnd.api+json")
					w.WriteHeader(http.StatusConflict)
					_ = json.NewEncoder(w).Encode(map[string]any{"errors": []map[string]any{{
						"status": strconv.Itoa(http.StatusConflict),
						"title":  "no baseline template",
						"detail": "No configuration template resolves for this tenant's region and version, so there is nothing to reset to.",
					}}})
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
