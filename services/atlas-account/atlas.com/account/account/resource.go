package account

import (
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
	"strconv"
)

const (
	getAccountByName = "get_account_by_name"
	getAccountById   = "get_account"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			r := router.PathPrefix("/accounts").Subrouter()
			r.HandleFunc("/", registerCreateAccount(si)(l, db)).Methods(http.MethodPost)
			r.HandleFunc("/", registerGetAccountByName(si)(l, db)).Queries("name", "{name}").Methods(http.MethodGet)
			r.HandleFunc("/{accountId}", registerGetAccountById(si)(l, db)).Methods(http.MethodGet)
		}
	}
}

func registerCreateAccount(si jsonapi.ServerInformation) func(l logrus.FieldLogger, db *gorm.DB) http.HandlerFunc {
	return func(l logrus.FieldLogger, db *gorm.DB) http.HandlerFunc {
		return rest.RetrieveSpan(l, getAccountByName, func(sl logrus.FieldLogger, span opentracing.Span) http.HandlerFunc {
			return rest.ParseTenant(sl, func(tenant tenant.Model) http.HandlerFunc {
				return parseCreateInput(sl, func(input CreateRestModel) http.HandlerFunc {
					return handleCreateAccount(si)(sl, db)(span)(tenant)(input)
				})
			})
		})
	}
}

type createInputHandler func(container CreateRestModel) http.HandlerFunc

func parseCreateInput(l logrus.FieldLogger, next createInputHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var model CreateRestModel

		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		err = jsonapi.Unmarshal(body, &model)
		if err != nil {
			l.WithError(err).Errorln("Deserializing input", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(model)(w, r)
	}
}

func handleCreateAccount(_ jsonapi.ServerInformation) func(l logrus.FieldLogger, db *gorm.DB) func(span opentracing.Span) func(model tenant.Model) func(input CreateRestModel) http.HandlerFunc {
	return func(l logrus.FieldLogger, db *gorm.DB) func(span opentracing.Span) func(model tenant.Model) func(input CreateRestModel) http.HandlerFunc {
		return func(span opentracing.Span) func(model tenant.Model) func(input CreateRestModel) http.HandlerFunc {
			return func(model tenant.Model) func(input CreateRestModel) http.HandlerFunc {
				return func(input CreateRestModel) http.HandlerFunc {
					return func(w http.ResponseWriter, r *http.Request) {
						emitCreateCommand(l, span, model)(input.Name, input.Password)
						w.WriteHeader(http.StatusAccepted)
					}
				}
			}
		}
	}
}

func registerGetAccountByName(si jsonapi.ServerInformation) func(l logrus.FieldLogger, db *gorm.DB) http.HandlerFunc {
	return func(l logrus.FieldLogger, db *gorm.DB) http.HandlerFunc {
		return rest.RetrieveSpan(l, getAccountByName, func(sl logrus.FieldLogger, span opentracing.Span) http.HandlerFunc {
			return rest.ParseTenant(sl, func(tenant tenant.Model) http.HandlerFunc {
				return parseName(sl, func(name string) http.HandlerFunc {
					return handleGetAccountByName(si)(sl, db)(span)(tenant)(name)
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

func ParseId(l logrus.FieldLogger, next idHandler) http.HandlerFunc {
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
		return rest.RetrieveSpan(l, getAccountById, func(sl logrus.FieldLogger, span opentracing.Span) http.HandlerFunc {
			return rest.ParseTenant(sl, func(tenant tenant.Model) http.HandlerFunc {
				return ParseId(sl, func(id uint32) http.HandlerFunc {
					return handleGetAccountById(si)(sl, db)(span)(tenant)(id)
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
