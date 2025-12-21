package templates

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
			r := router.PathPrefix("/configurations/templates").Subrouter()
			r.HandleFunc("", rest.RegisterInputHandler[RestModel](l)(si)("create_configuration_template", handleCreateConfigurationTemplate(db))).Methods(http.MethodPost)
			r.HandleFunc("", rest.RegisterHandler(l)(si)("get_configuration_template", handleGetConfigurationTemplate(db))).Methods(http.MethodGet).Queries("region", "{region}", "majorVersion", "{majorVersion}", "minorVersion", "{minorVersion}")
			r.HandleFunc("", rest.RegisterHandler(l)(si)("get_configuration_templates", handleGetConfigurationTemplates(db))).Methods(http.MethodGet)
			r.HandleFunc("/{templateId}", rest.RegisterHandler(l)(si)("get_configuration_template", handleGetConfigurationTemplateById(db))).Methods(http.MethodGet)
			r.HandleFunc("/{templateId}", rest.RegisterInputHandler[RestModel](l)(si)("update_configuration_template", handleUpdateConfigurationTemplate(db))).Methods(http.MethodPatch)
			r.HandleFunc("/{templateId}", rest.RegisterHandler(l)(si)("delete_configuration_template", handleDeleteConfigurationTemplate(db))).Methods(http.MethodDelete)
		}
	}
}

func handleCreateConfigurationTemplate(db *gorm.DB) rest.InputHandler[RestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			templateId, err := NewProcessor(d.Logger(), d.Context(), db).Create(input)
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to create configuration template.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// Set the Location header to the URL of the newly created resource
			w.Header().Set("Location", "/configurations/templates/"+templateId.String())

			// Get the created resource
			input.Id = templateId.String()

			// Return the created resource
			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			w.WriteHeader(http.StatusCreated)
			server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(input)
		}
	}
}

func handleGetConfigurationTemplate(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseRegion(d.Logger(), func(region string) http.HandlerFunc {
			return rest.ParseMajorVersion(d.Logger(), func(majorVersion uint16) http.HandlerFunc {
				return rest.ParseMinorVersion(d.Logger(), func(minorVersion uint16) http.HandlerFunc {
					return func(w http.ResponseWriter, r *http.Request) {
						cts, err := NewProcessor(d.Logger(), d.Context(), db).GetByRegionAndVersion(region, majorVersion, minorVersion)
						if err != nil {
							d.Logger().WithError(err).Errorf("Unable to get configuration templates.")
							w.WriteHeader(http.StatusInternalServerError)
							return
						}

						query := r.URL.Query()
						queryParams := jsonapi.ParseQueryFields(&query)
						server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(cts)
					}
				})
			})
		})
	}
}

func handleGetConfigurationTemplates(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			cts, err := NewProcessor(d.Logger(), d.Context(), db).GetAll()
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to get configuration templates.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(cts)
		}
	}
}

func handleGetConfigurationTemplateById(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseTemplateId(d.Logger(), func(templateId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				cts, err := NewProcessor(d.Logger(), d.Context(), db).GetById(templateId)
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to get configuration templates.")
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

func handleUpdateConfigurationTemplate(db *gorm.DB) rest.InputHandler[RestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
		return rest.ParseTemplateId(d.Logger(), func(templateId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				err := NewProcessor(d.Logger(), d.Context(), db).UpdateById(templateId, input)
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to update configuration template.")
					w.WriteHeader(http.StatusInternalServerError)
				}
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
					w.WriteHeader(http.StatusInternalServerError)
				}
			}
		})
	}
}
