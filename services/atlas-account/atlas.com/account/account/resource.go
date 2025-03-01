package account

import (
	"atlas-account/kafka/producer"
	"atlas-account/rest"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"net/http"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			register := rest.RegisterHandler(l)(db)(si)
			registerInput := rest.RegisterInputHandler[RestModel](l)(db)(si)

			r := router.PathPrefix("/accounts").Subrouter()
			r.HandleFunc("/", registerInput("create_account", handleCreateAccount)).Methods(http.MethodPost)
			r.HandleFunc("/", register("get_account_by_name", handleGetAccountByName)).Queries("name", "{name}").Methods(http.MethodGet)
			r.HandleFunc("/", register("get_accounts", handleGetAccounts)).Methods(http.MethodGet)
			r.HandleFunc("/{accountId}", register("get_account", handleGetAccountById)).Methods(http.MethodGet)
			r.HandleFunc("/{accountId}", registerInput("update_account", handleUpdateAccount)).Methods(http.MethodPatch)
		}
	}
}

func handleUpdateAccount(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
	return rest.ParseAccountId(d.Logger(), func(accountId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			im, err := model.Map(Extract)(model.FixedProvider(input))()
			if err != nil {
				d.Logger().WithError(err).Errorf("Invalid input.")
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			a, err := Update(d.Logger(), d.DB(), d.Context())(accountId, im)
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to update account [%d].", accountId)
				w.WriteHeader(http.StatusNotFound)
				return
			}

			res, err := model.Map(Transform)(model.FixedProvider(a))()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
		}
	})
}

func handleCreateAccount(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_ = producer.ProviderImpl(d.Logger())(d.Context())(EnvCreateAccountCommandTopic)(createCommandProvider(input.Name, input.Password))
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
			res, err := model.Map(Transform)(ByNameProvider(d.DB())(d.Context())(name))()
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to retrieve account by name [%s].", name)
				w.WriteHeader(http.StatusNotFound)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
		}
	})
}

func handleGetAccounts(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		as, err := GetByTenant(d.DB())(d.Context())
		if err != nil {
			d.Logger().WithError(err).Errorf("Unable to locate accounts.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		res, err := model.SliceMap(Transform)(model.FixedProvider(as))(model.ParallelMap())()
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating REST model.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		query := r.URL.Query()
		queryParams := jsonapi.ParseQueryFields(&query)
		server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
	}
}

func handleGetAccountById(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseAccountId(d.Logger(), func(id uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			res, err := model.Map(Transform)(ByIdProvider(d.DB())(d.Context())(id))()
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to locate account [%d].", id)
				w.WriteHeader(http.StatusNotFound)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
		}
	})
}
