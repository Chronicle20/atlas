package account

import (
	"atlas-account/rest"
	"atlas-account/tenant"
	"github.com/gorilla/mux"
	"github.com/manyminds/api2go/jsonapi"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"net/http"
	"strconv"
)

const (
	getAccountByName = "get_account_by_name"
	getAccountById   = "get_account"
)

func InitResource(si jsonapi.ServerInformation) func(router *mux.Router, l logrus.FieldLogger, db *gorm.DB) {
	return func(router *mux.Router, l logrus.FieldLogger, db *gorm.DB) {
		r := router.PathPrefix("/accounts").Subrouter()
		r.HandleFunc("/", registerGetAccountByName(si)(l, db)).Queries("name", "{name}").Methods(http.MethodGet)
		r.HandleFunc("/{accountId}", registerGetAccountById(si)(l, db)).Methods(http.MethodGet)
	}
}

func registerGetAccountByName(si jsonapi.ServerInformation) func(l logrus.FieldLogger, db *gorm.DB) http.HandlerFunc {
	return func(l logrus.FieldLogger, db *gorm.DB) http.HandlerFunc {
		return rest.RetrieveSpan(getAccountByName, func(span opentracing.Span) http.HandlerFunc {
			return tenant.ParseTenant(l, func(tenant tenant.Model) http.HandlerFunc {
				return parseName(l, func(name string) http.HandlerFunc {
					return handleGetAccountByName(si)(l, db)(span)(tenant)(name)
				})
			})
		})
	}
}

type nameHandler func(name string) http.HandlerFunc

func parseName(l logrus.FieldLogger, next nameHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if val, ok := mux.Vars(r)["name"]; ok {
			next(val)(w, r)
		} else {
			l.Errorf("Missing name parameter.")
			w.WriteHeader(http.StatusBadRequest)
		}
	}
}

func handleGetAccountByName(si jsonapi.ServerInformation) func(l logrus.FieldLogger, db *gorm.DB) func(span opentracing.Span) func(tenant tenant.Model) func(name string) http.HandlerFunc {
	return func(l logrus.FieldLogger, db *gorm.DB) func(span opentracing.Span) func(tenant tenant.Model) func(name string) http.HandlerFunc {
		return func(span opentracing.Span) func(tenant tenant.Model) func(name string) http.HandlerFunc {
			return func(tenant tenant.Model) func(name string) http.HandlerFunc {
				return func(name string) http.HandlerFunc {
					return func(w http.ResponseWriter, r *http.Request) {
						a, err := GetByName(l, db, tenant)(name)
						if err != nil {
							l.WithError(err).Errorf("Unable to locate account [%s].", name)
							w.WriteHeader(http.StatusNotFound)
							return
						}

						res, err := jsonapi.MarshalWithURLs(Transform(a), si)
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

type idHandler func(id uint32) http.HandlerFunc

func parseId(l logrus.FieldLogger, next idHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		value, err := strconv.Atoi(vars["accountId"])
		if err != nil {
			l.WithError(err).Errorln("Error parsing id as uint32")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(uint32(value))(w, r)
	}
}

func registerGetAccountById(si jsonapi.ServerInformation) func(l logrus.FieldLogger, db *gorm.DB) http.HandlerFunc {
	return func(l logrus.FieldLogger, db *gorm.DB) http.HandlerFunc {
		return rest.RetrieveSpan(getAccountById, func(span opentracing.Span) http.HandlerFunc {
			return tenant.ParseTenant(l, func(tenant tenant.Model) http.HandlerFunc {
				return parseId(l, func(id uint32) http.HandlerFunc {
					return handleGetAccountById(si)(l, db)(span)(tenant)(id)
				})
			})
		})
	}
}

func handleGetAccountById(si jsonapi.ServerInformation) func(l logrus.FieldLogger, db *gorm.DB) func(span opentracing.Span) func(tenant tenant.Model) func(id uint32) http.HandlerFunc {
	return func(l logrus.FieldLogger, db *gorm.DB) func(span opentracing.Span) func(tenant tenant.Model) func(id uint32) http.HandlerFunc {
		return func(span opentracing.Span) func(tenant tenant.Model) func(id uint32) http.HandlerFunc {
			return func(tenant tenant.Model) func(id uint32) http.HandlerFunc {
				return func(id uint32) http.HandlerFunc {
					return func(w http.ResponseWriter, r *http.Request) {
						a, err := GetById(l, db, tenant)(id)
						if err != nil {
							l.WithError(err).Errorf("Unable to locate account [%d].", id)
							w.WriteHeader(http.StatusNotFound)
							return
						}

						res, err := jsonapi.MarshalWithURLs(Transform(a), si)
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
