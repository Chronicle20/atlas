package services

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
			r := router.PathPrefix("/configurations/services").Subrouter()
			r.HandleFunc("", rest.RegisterHandler(l)(si)("get_service_configurations", handleGetServiceConfigurations(db))).Methods(http.MethodGet)
			r.HandleFunc("/{serviceId}", rest.RegisterHandler(l)(si)("get_service_configuration", handleGetServiceConfiguration(db))).Methods(http.MethodGet)
		}
	}
}

func handleGetServiceConfigurations(db *gorm.DB) rest.GetHandler {
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
			server.MarshalResponse[[]interface{}](d.Logger())(w)(c.ServerInformation())(queryParams)(cts)
		}
	}
}

func handleGetServiceConfiguration(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseServiceId(d.Logger(), func(serviceId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				cts, err := NewProcessor(d.Logger(), d.Context(), db).GetById(serviceId)
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to get configuration tenants.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[interface{}](d.Logger())(w)(c.ServerInformation())(queryParams)(cts)
			}
		})
	}
}
