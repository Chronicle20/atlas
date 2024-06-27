package session

import (
	"atlas-account/account"
	"atlas-account/rest"
	"atlas-account/tenant"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/manyminds/api2go/jsonapi"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"io"
	"net/http"
)

const (
	createSession = "create_session"
	deleteSession = "delete_session"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			r := router.PathPrefix("/accounts/{accountId}/sessions").Subrouter()
			r.HandleFunc("/", registerCreateSession(si)(l, db)).Methods(http.MethodPost)
			r.HandleFunc("/", registerDeleteSession(si)(l, db)).Methods(http.MethodDelete)
		}
	}
}

func registerCreateSession(si jsonapi.ServerInformation) func(l logrus.FieldLogger, db *gorm.DB) http.HandlerFunc {
	return func(l logrus.FieldLogger, db *gorm.DB) http.HandlerFunc {
		return rest.RetrieveSpan(createSession, func(span opentracing.Span) http.HandlerFunc {
			return rest.ParseTenant(l, func(tenant tenant.Model) http.HandlerFunc {
				return parseInput(l, func(container InputRestModel) http.HandlerFunc {
					return handleCreateSession(si)(l, db)(span)(tenant)(container)
				})
			})
		})
	}
}

func registerDeleteSession(si jsonapi.ServerInformation) func(l logrus.FieldLogger, db *gorm.DB) http.HandlerFunc {
	return func(l logrus.FieldLogger, db *gorm.DB) http.HandlerFunc {
		return rest.RetrieveSpan(deleteSession, func(span opentracing.Span) http.HandlerFunc {
			return rest.ParseTenant(l, func(tenant tenant.Model) http.HandlerFunc {
				return account.ParseId(l, func(id uint32) http.HandlerFunc {
					return handleDeleteSession(si)(l, db)(span)(tenant)(id)
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

func handleCreateSession(si jsonapi.ServerInformation) func(l logrus.FieldLogger, db *gorm.DB) func(span opentracing.Span) func(tenant tenant.Model) func(container InputRestModel) http.HandlerFunc {
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

func handleDeleteSession(_ jsonapi.ServerInformation) func(l logrus.FieldLogger, db *gorm.DB) func(span opentracing.Span) func(tenant tenant.Model) func(accountId uint32) http.HandlerFunc {
	return func(l logrus.FieldLogger, db *gorm.DB) func(span opentracing.Span) func(tenant tenant.Model) func(accountId uint32) http.HandlerFunc {
		return func(span opentracing.Span) func(tenant tenant.Model) func(accountId uint32) http.HandlerFunc {
			return func(tenant tenant.Model) func(accountId uint32) http.HandlerFunc {
				return func(accountId uint32) http.HandlerFunc {
					return func(w http.ResponseWriter, r *http.Request) {
						emitLogoutCommand(l, span, tenant)(accountId)
						w.WriteHeader(http.StatusAccepted)
					}
				}
			}
		}
	}
}
