package services

import (
	"atlas-configurations/rest"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/manyminds/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"net/http"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			r := router.PathPrefix("/configurations/services").Subrouter()
			r.HandleFunc("", rest.RegisterHandler(l)(si)("get_service_configurations", handleGetServiceConfigurations(db))).Methods(http.MethodGet)
			r.HandleFunc("/{serviceId}", rest.RegisterHandler(l)(si)("get_service_configuration", handleGetServiceConfiguration(db))).Methods(http.MethodGet)
		}
	}
}

func handleGetServiceConfigurations(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			cts, err := GetAll(d.Logger())(d.Context())(db)()
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to get configuration tenants.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			server.Marshal[[]interface{}](d.Logger())(w)(c.ServerInformation())(cts)
		}
	}
}

func handleGetServiceConfiguration(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseServiceId(d.Logger(), func(serviceId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				cts, err := GetById(d.Logger())(d.Context())(db)(serviceId)
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to get configuration tenants.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				server.Marshal[interface{}](d.Logger())(w)(c.ServerInformation())(cts)
			}
		})
	}
}
