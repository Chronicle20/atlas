package tenants

import (
	"atlas-configurations/data"
	"atlas-configurations/rest"
	"atlas-configurations/tenants/characters/preset"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	tenantlib "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
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
		}
	}
}

func handleGetConfigurationTenants(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			cts, err := NewProcessor(d.Logger(), d.Context(), db).GetAll()
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to get configuration tenants.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(cts)
		}
	}
}

func handleGetConfigurationTenant(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				cts, err := NewProcessor(d.Logger(), d.Context(), db).GetById(tenantId)
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to get configuration tenants.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(cts)
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
					WithValidator(preset.NewValidator(data.NewClient(d.Logger())))
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
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			}
		})
	}
}

func handleCreateConfigurationTenant(db *gorm.DB) rest.InputHandler[RestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			tenantId, err := NewProcessor(d.Logger(), d.Context(), db).Create(input)
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to create configuration tenant.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// Set the Location header to the URL of the newly created resource
			w.Header().Set("Location", "/configurations/tenants/"+tenantId.String())

			// Set the ID of the input model to the created tenant ID
			input.Id = tenantId.String()

			// Return the created resource
			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			w.WriteHeader(http.StatusCreated)
			server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(input)
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
					w.WriteHeader(http.StatusInternalServerError)
				}
			}
		})
	}
}
