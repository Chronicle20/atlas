package login

import (
	"atlas-account/rest"
	"atlas-account/tenant"
	"github.com/gorilla/mux"
	"github.com/manyminds/api2go/jsonapi"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"io"
	"net/http"
)

const (
	createLogin = "create_login"
)

func InitResource(si jsonapi.ServerInformation) func(router *mux.Router, l logrus.FieldLogger, db *gorm.DB) {
	return func(router *mux.Router, l logrus.FieldLogger, db *gorm.DB) {
		r := router.PathPrefix("/logins").Subrouter()
		r.HandleFunc("/", registerCreateLogin(si)(l, db)).Methods(http.MethodPost)
	}
}

func registerCreateLogin(si jsonapi.ServerInformation) func(l logrus.FieldLogger, db *gorm.DB) http.HandlerFunc {
	return func(l logrus.FieldLogger, db *gorm.DB) http.HandlerFunc {
		return rest.RetrieveSpan(createLogin, func(span opentracing.Span) http.HandlerFunc {
			return tenant.ParseTenant(l, func(tenant tenant.Model) http.HandlerFunc {
				return parseInput(l, func(container InputRestModel) http.HandlerFunc {
					return handleCreateLogin(si)(l, db)(span)(tenant)(container)
				})
			})
		})
	}
}

type inputHandler func(container InputRestModel) http.HandlerFunc

func parseInput(l logrus.FieldLogger, next inputHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var login InputRestModel

		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		err = jsonapi.Unmarshal(body, &login)
		if err != nil {
			l.WithError(err).Errorln("Deserializing input", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(login)(w, r)
	}
}

func handleCreateLogin(si jsonapi.ServerInformation) func(l logrus.FieldLogger, db *gorm.DB) func(span opentracing.Span) func(tenant tenant.Model) func(container InputRestModel) http.HandlerFunc {
	return func(l logrus.FieldLogger, db *gorm.DB) func(span opentracing.Span) func(tenant tenant.Model) func(container InputRestModel) http.HandlerFunc {
		return func(span opentracing.Span) func(tenant tenant.Model) func(container InputRestModel) http.HandlerFunc {
			return func(tenant tenant.Model) func(container InputRestModel) http.HandlerFunc {
				return func(container InputRestModel) http.HandlerFunc {
					return func(w http.ResponseWriter, r *http.Request) {
						resp := AttemptLogin(l, db, span, tenant)(container.SessionId, container.Name, container.Password)
						res, err := jsonapi.MarshalWithURLs(Transform(resp), si)
						if err != nil {
							l.WithError(err).Errorf("Unable to marshal models.")
							w.WriteHeader(http.StatusInternalServerError)
							return
						}

						_, err = w.Write(res)
						if err != nil {
							l.WithError(err).Errorf("Unable to write response.")
							w.WriteHeader(http.StatusInternalServerError)
							return
						}
					}
				}
			}
		}
	}
}
