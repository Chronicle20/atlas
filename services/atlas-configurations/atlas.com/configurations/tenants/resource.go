package tenants

import (
	"atlas-configurations/rest"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"net/http"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			r := router.PathPrefix("/configurations/tenants").Subrouter()
			r.HandleFunc("", rest.RegisterHandler(l)(si)("get_configuration_tenants", handleGetConfigurationTenants(db))).Methods(http.MethodGet)
			r.HandleFunc("/{tenantId}", rest.RegisterHandler(l)(si)("get_configuration_tenant", handleGetConfigurationTenant(db))).Methods(http.MethodGet)
			r.HandleFunc("/{tenantId}", rest.RegisterInputHandler[RestModel](l)(si)("update_configuration_tenant", handleUpdateConfigurationTenant(db))).Methods(http.MethodPatch)
		}
	}
}

func handleGetConfigurationTenants(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			cts, err := GetAll(d.Logger())(d.Context())(db)()
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to get configuration tenants.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			server.Marshal[[]RestModel](d.Logger())(w)(c.ServerInformation())(cts)
		}
	}
}

func handleGetConfigurationTenant(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				cts, err := GetById(d.Logger())(d.Context())(db)(tenantId)
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to get configuration tenants.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				server.Marshal[RestModel](d.Logger())(w)(c.ServerInformation())(cts)
			}
		})
	}
}

func handleUpdateConfigurationTenant(db *gorm.DB) rest.InputHandler[RestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
		return rest.ParseTenantId(d.Logger(), func(tenantId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				err := UpdateById(d.Logger())(d.Context())(db)(tenantId, input)
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to update configuration tenant.")
					w.WriteHeader(http.StatusInternalServerError)
				}
			}
		})
	}
}
