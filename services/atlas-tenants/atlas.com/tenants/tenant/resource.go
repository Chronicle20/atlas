package tenant

import (
	"atlas-tenants/rest"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"net/http"
)

// GetAllTenantsHandler handles GET /tenants
func GetAllTenantsHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			p := NewProcessor(d.Logger(), d.Context(), db)
			tenants, err := p.GetAll()
			if err != nil {
				d.Logger().WithError(err).Error("Failed to get all tenants")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			restModels := make([]RestModel, len(tenants))
			for i, tenant := range tenants {
				rm, err := Transform(tenant)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to transform tenant")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				restModels[i] = rm
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(restModels)
		}
	}
}

// GetTenantByIdHandler handles GET /tenants/{tenantId}
func GetTenantByIdHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				p := NewProcessor(d.Logger(), d.Context(), db)
				tenant, err := p.GetById(tenantId)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to get tenant")
					w.WriteHeader(http.StatusNotFound)
					return
				}

				rm, err := Transform(tenant)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to transform tenant")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
			}
		})
	}
}

// CreateTenantHandler handles POST /tenants
func CreateTenantHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext, model RestModel) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, model RestModel) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			name, region, majorVersion, minorVersion, err := Extract(model)
			if err != nil {
				d.Logger().WithError(err).Error("Failed to extract tenant data")
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			p := NewProcessor(d.Logger(), d.Context(), db)
			tenant, err := p.Create(name, region, majorVersion, minorVersion)
			if err != nil {
				d.Logger().WithError(err).Error("Failed to create tenant")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			rm, err := Transform(tenant)
			if err != nil {
				d.Logger().WithError(err).Error("Failed to transform tenant")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			w.WriteHeader(http.StatusCreated)
			server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
		}
	}
}

// UpdateTenantHandler handles PATCH /tenants/{tenantId}
func UpdateTenantHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext, model RestModel) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, model RestModel) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				name, region, majorVersion, minorVersion, err := Extract(model)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to extract tenant data")
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				p := NewProcessor(d.Logger(), d.Context(), db)
				tenant, err := p.Update(tenantId, name, region, majorVersion, minorVersion)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to update tenant")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				rm, err := Transform(tenant)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to transform tenant")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
			}
		})
	}
}

// DeleteTenantHandler handles DELETE /tenants/{tenantId}
func DeleteTenantHandler(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				p := NewProcessor(d.Logger(), d.Context(), db)
				err := p.Delete(tenantId)
				if err != nil {
					d.Logger().WithError(err).Error("Failed to delete tenant")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				w.WriteHeader(http.StatusNoContent)
			}
		})
	}
}

// RegisterRoutes registers the tenant routes
func RegisterRoutes(db *gorm.DB) func(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(si jsonapi.ServerInformation) server.RouteInitializer {
		return func(r *mux.Router, l logrus.FieldLogger) {
			registerHandler := rest.RegisterHandler(l)(si)
			registerInputHandler := rest.RegisterInputHandler[RestModel](l)(si)

			r.HandleFunc("/tenants", registerHandler("get_all_tenants", GetAllTenantsHandler(db))).Methods(http.MethodGet)
			r.HandleFunc("/tenants/{tenantId}", registerHandler("get_tenant_by_id", GetTenantByIdHandler(db))).Methods(http.MethodGet)
			r.HandleFunc("/tenants", registerInputHandler("create_tenant", CreateTenantHandler(db))).Methods(http.MethodPost)
			r.HandleFunc("/tenants/{tenantId}", registerInputHandler("update_tenant", UpdateTenantHandler(db))).Methods(http.MethodPatch)
			r.HandleFunc("/tenants/{tenantId}", registerHandler("delete_tenant", DeleteTenantHandler(db))).Methods(http.MethodDelete)
		}
	}
}
