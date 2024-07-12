package account

import (
	"atlas-account/rest"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/manyminds/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"net/http"
)

const (
	createAccount    = "create_account"
	getAccountByName = "get_account_by_name"
	getAccountById   = "get_account"
	updateAccount    = "update_account"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			register := rest.RegisterHandler(l)(db)(si)
			registerInput := rest.RegisterInputHandler[RestModel](l)(db)(si)

			r := router.PathPrefix("/accounts").Subrouter()
			r.HandleFunc("/", registerInput(createAccount, handleCreateAccount)).Methods(http.MethodPost)
			r.HandleFunc("/", register(getAccountByName, handleGetAccountByName)).Queries("name", "{name}").Methods(http.MethodGet)
			r.HandleFunc("/{accountId}", register(getAccountById, handleGetAccountById)).Methods(http.MethodGet)
			r.HandleFunc("/{accountId}", registerInput(updateAccount, handleUpdateAccount)).Methods(http.MethodPatch)
		}
	}
}

func handleUpdateAccount(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
	return rest.ParseAccountId(d.Logger(), func(accountId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			im, err := model.Transform(input, Extract)
			if err != nil {
				d.Logger().WithError(err).Errorf("Invalid input.")
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			a, err := Update(d.Logger(), d.DB(), c.Tenant())(accountId, im)
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to update account [%d].", accountId)
				w.WriteHeader(http.StatusNotFound)
				return
			}

			res, err := model.Transform(a, Transform)
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			server.Marshal[RestModel](d.Logger())(w)(c.ServerInformation())(res)
		}
	})
}

func handleCreateAccount(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		emitCreateCommand(d.Logger(), d.Span(), c.Tenant())(input.Name, input.Password)
		w.WriteHeader(http.StatusAccepted)
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

func handleGetAccountByName(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return parseName(d.Logger(), func(name string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			a, err := GetByName(d.Logger(), d.DB(), c.Tenant())(name)
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to locate account [%s].", name)
				w.WriteHeader(http.StatusNotFound)
				return
			}

			res, err := model.Transform(a, Transform)
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			server.Marshal[RestModel](d.Logger())(w)(c.ServerInformation())(res)
		}
	})
}

func handleGetAccountById(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseAccountId(d.Logger(), func(id uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			a, err := GetById(d.Logger(), d.DB(), c.Tenant())(id)
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to locate account [%d].", id)
				w.WriteHeader(http.StatusNotFound)
				return
			}

			res, err := model.Transform(a, Transform)
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			server.Marshal[RestModel](d.Logger())(w)(c.ServerInformation())(res)
		}
	})
}
