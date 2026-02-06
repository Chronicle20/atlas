package tenants

import (
	"atlas-configurations/rest"
	"net/http"

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
				err := NewProcessor(d.Logger(), d.Context(), db).UpdateById(tenantId, input)
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to update configuration tenant.")
					w.WriteHeader(http.StatusInternalServerError)
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
