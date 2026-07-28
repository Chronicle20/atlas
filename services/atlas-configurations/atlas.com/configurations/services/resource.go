package services

import (
	"atlas-configurations/rest"
	"atlas-configurations/services/service"
	"net/http"

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
			r := router.PathPrefix("/configurations/services").Subrouter()
			r.HandleFunc("", rest.RegisterHandler(l)(si)("get_service_configurations", handleGetServiceConfigurations(db))).Methods(http.MethodGet)
			r.HandleFunc("", rest.RegisterInputHandler[service.InputRestModel](l)(si)("create_service_configuration", handleCreateServiceConfiguration(db))).Methods(http.MethodPost)
			r.HandleFunc("/{serviceId}", rest.RegisterHandler(l)(si)("get_service_configuration", handleGetServiceConfiguration(db))).Methods(http.MethodGet)
			r.HandleFunc("/{serviceId}", rest.RegisterInputHandler[service.InputRestModel](l)(si)("update_service_configuration", handleUpdateServiceConfiguration(db))).Methods(http.MethodPatch)
			r.HandleFunc("/{serviceId}", rest.RegisterHandler(l)(si)("delete_service_configuration", handleDeleteServiceConfiguration(db))).Methods(http.MethodDelete)
		}
	}
}

func handleGetServiceConfigurations(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			page, err := paginate.ParseParams(r.URL.Query(), paginate.DefaultPageSize, paginate.MaxPageSize)
			if err != nil {
				server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
				return
			}

			paged, err := NewProcessor(d.Logger(), d.Context(), db).AllProvider(page)()
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to get service configurations.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalPaginatedResponse[[]interface{}](d.Logger())(w)(c.ServerInformation())(queryParams)(paged.Items, paginate.EnvelopeFor(paged), r)
		}
	}
}

func handleGetServiceConfiguration(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseServiceId(d.Logger(), func(serviceId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				cts, err := NewProcessor(d.Logger(), d.Context(), db).GetById(serviceId)
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to get service configuration.")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[interface{}](d.Logger())(w)(c.ServerInformation())(queryParams)(cts)
			}
		})
	}
}

func handleCreateServiceConfiguration(db *gorm.DB) rest.InputHandler[service.InputRestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, input service.InputRestModel) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if !IsValidServiceType(input.Type) {
				d.Logger().Errorf("Invalid service type: %s", input.Type)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			serviceId, err := NewProcessor(d.Logger(), d.Context(), db).Create(input)
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to create service configuration.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			// Set the Location header to the URL of the newly created resource
			w.Header().Set("Location", "/configurations/services/"+serviceId.String())

			// Fetch the created service to return it
			svc, err := NewProcessor(d.Logger(), d.Context(), db).GetById(serviceId)
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to get created service configuration.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			w.WriteHeader(http.StatusCreated)
			server.MarshalResponse[interface{}](d.Logger())(w)(c.ServerInformation())(queryParams)(svc)
		}
	}
}

func handleUpdateServiceConfiguration(db *gorm.DB) rest.InputHandler[service.InputRestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, input service.InputRestModel) http.HandlerFunc {
		return rest.ParseServiceId(d.Logger(), func(serviceId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				if !IsValidServiceType(input.Type) {
					d.Logger().Errorf("Invalid service type: %s", input.Type)
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				err := NewProcessor(d.Logger(), d.Context(), db).UpdateById(serviceId, input)
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to update service configuration.")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				// Fetch the updated service to return it
				svc, err := NewProcessor(d.Logger(), d.Context(), db).GetById(serviceId)
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to get updated service configuration.")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[interface{}](d.Logger())(w)(c.ServerInformation())(queryParams)(svc)
			}
		})
	}
}

func handleDeleteServiceConfiguration(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseServiceId(d.Logger(), func(serviceId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				err := NewProcessor(d.Logger(), d.Context(), db).DeleteById(serviceId)
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to delete service configuration.")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			}
		})
	}
}
