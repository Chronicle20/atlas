package environments

import (
	"atlas-configurations/rest"
	"errors"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
)

// InitResource registers /configurations/environments. Name (not a
// generated id) is the resource's addressable identity in the URL, since
// name is the wire identity everywhere else it appears - the outbox key,
// env.Record.Name, and what the sparse overlay's bootstrap Job knows about
// itself.
func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			r := router.PathPrefix("/configurations/environments").Subrouter()
			r.HandleFunc("", rest.RegisterHandler(l)(si)("get_configuration_environments", handleGetConfigurationEnvironments(db))).Methods(http.MethodGet)
			r.HandleFunc("", rest.RegisterInputHandler[RestModel](l)(si)("create_configuration_environment", handleCreateConfigurationEnvironment(db))).Methods(http.MethodPost)
			r.HandleFunc("/{name}", rest.RegisterHandler(l)(si)("get_configuration_environment", handleGetConfigurationEnvironment(db))).Methods(http.MethodGet)
			r.HandleFunc("/{name}", rest.RegisterInputHandler[RestModel](l)(si)("update_configuration_environment", handleUpdateConfigurationEnvironment(db))).Methods(http.MethodPatch)
		}
	}
}

func parseName(l logrus.FieldLogger, next func(string) http.HandlerFunc) http.HandlerFunc {
	return server.ParseStringId(l, "name", next)
}

// isValidationError reports whether err is one of the processor's 400-worthy
// input errors, as opposed to a persistence/lookup failure.
func isValidationError(err error) bool {
	return errors.Is(err, ErrNameRequired) || errors.Is(err, ErrInvalidName) ||
		errors.Is(err, ErrInvalidPhase) || errors.Is(err, ErrIllegalPhaseTransition)
}

func handleGetConfigurationEnvironments(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			page, err := paginate.ParseParams(r.URL.Query(), paginate.DefaultPageSize, paginate.MaxPageSize)
			if err != nil {
				server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
				return
			}

			paged, err := NewProcessor(d.Logger(), d.Context(), db).AllProvider(page)()
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to get configuration environments.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalPaginatedResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(paged.Items, paginate.EnvelopeFor(paged), r)
		}
	}
}

func handleGetConfigurationEnvironment(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return parseName(d.Logger(), func(name string) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				rm, err := NewProcessor(d.Logger(), d.Context(), db).GetByName(name)
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to get configuration environment.")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
			}
		})
	}
}

func handleCreateConfigurationEnvironment(db *gorm.DB) rest.InputHandler[RestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			rm, err := NewProcessor(d.Logger(), d.Context(), db).Create(input)
			if err != nil {
				if isValidationError(err) {
					server.WriteBadRequest(d.Logger(), w, err.Error())
					return
				}
				d.Logger().WithError(err).Errorf("Unable to create configuration environment.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			w.Header().Set("Location", "/configurations/environments/"+rm.Name)

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			w.WriteHeader(http.StatusCreated)
			server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
		}
	}
}

func handleUpdateConfigurationEnvironment(db *gorm.DB) rest.InputHandler[RestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
		return parseName(d.Logger(), func(name string) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				rm, err := NewProcessor(d.Logger(), d.Context(), db).UpdateByName(name, input)
				if err != nil {
					if isValidationError(err) {
						server.WriteBadRequest(d.Logger(), w, err.Error())
						return
					}
					d.Logger().WithError(err).Errorf("Unable to update configuration environment.")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
			}
		})
	}
}
