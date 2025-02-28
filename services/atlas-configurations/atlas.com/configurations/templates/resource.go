package templates

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
			r := router.PathPrefix("/configurations/templates").Subrouter()
			r.HandleFunc("", rest.RegisterInputHandler[RestModel](l)(si)("create_configuration_template", handleCreateConfigurationTemplate(db))).Methods(http.MethodPost)
			r.HandleFunc("", rest.RegisterHandler(l)(si)("get_configuration_template", handleGetConfigurationTemplate(db))).Methods(http.MethodGet).Queries("region", "{region}", "majorVersion", "{majorVersion}", "minorVersion", "{minorVersion}")
			r.HandleFunc("", rest.RegisterHandler(l)(si)("get_configuration_templates", handleGetConfigurationTemplates(db))).Methods(http.MethodGet)
			r.HandleFunc("/{tenantId}", rest.RegisterInputHandler[RestModel](l)(si)("update_configuration_template", handleUpdateConfigurationTemplate(db))).Methods(http.MethodPatch)
		}
	}
}

func handleCreateConfigurationTemplate(db *gorm.DB) rest.InputHandler[RestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			err := Create(d.Logger())(d.Context())(db)(input)
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to create configuration tenant.")
				w.WriteHeader(http.StatusInternalServerError)
			}
		}
	}
}

func handleGetConfigurationTemplate(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseRegion(d.Logger(), func(region string) http.HandlerFunc {
			return rest.ParseMajorVersion(d.Logger(), func(majorVersion uint16) http.HandlerFunc {
				return rest.ParseMinorVersion(d.Logger(), func(minorVersion uint16) http.HandlerFunc {
					return func(w http.ResponseWriter, r *http.Request) {
						cts, err := GetByRegionAndVersion(d.Logger())(d.Context())(db)(region, majorVersion, minorVersion)
						if err != nil {
							d.Logger().WithError(err).Errorf("Unable to get configuration templates.")
							w.WriteHeader(http.StatusInternalServerError)
							return
						}

						server.Marshal[RestModel](d.Logger())(w)(c.ServerInformation())(cts)
					}
				})
			})
		})
	}
}

func handleGetConfigurationTemplates(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			cts, err := GetAll(d.Logger())(d.Context())(db)()
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to get configuration templates.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			server.Marshal[[]RestModel](d.Logger())(w)(c.ServerInformation())(cts)
		}
	}
}

func handleUpdateConfigurationTemplate(db *gorm.DB) rest.InputHandler[RestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
		return rest.ParseTemplateId(d.Logger(), func(templateId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				err := UpdateById(d.Logger())(d.Context())(db)(templateId, input)
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to update configuration tenant.")
					w.WriteHeader(http.StatusInternalServerError)
				}
			}
		})
	}
}
