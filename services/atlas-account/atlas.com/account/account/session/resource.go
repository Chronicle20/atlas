package session

import (
	"atlas-account/account"
	"atlas-account/kafka/producer"
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
	createSession = "create_session"
	updateSession = "update_session"
	deleteSession = "delete_session"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			r := router.PathPrefix("/accounts/{accountId}/sessions").Subrouter()
			r.HandleFunc("/", rest.RegisterInputHandler[InputRestModel](l)(db)(si)(createSession, handleCreateSession)).Methods(http.MethodPost)
			r.HandleFunc("/", rest.RegisterInputHandler[InputRestModel](l)(db)(si)(updateSession, handleUpdateSession)).Methods(http.MethodPatch)
			r.HandleFunc("/", rest.RegisterHandler(l)(db)(si)(deleteSession, handleDeleteSession)).Methods(http.MethodDelete)
		}
	}
}

func handleCreateSession(d *rest.HandlerDependency, c *rest.HandlerContext, input InputRestModel) http.HandlerFunc {
	return rest.ParseAccountId(d.Logger(), func(accountId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			resp := AttemptLogin(d.Logger(), d.DB(), d.Span(), c.Tenant())(input.SessionId, input.Name, input.Password)
			res, err := model.Map(model.FixedProvider(resp), Transform)()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			server.Marshal[OutputRestModel](d.Logger())(w)(c.ServerInformation())(res)
		}
	})
}

func handleUpdateSession(d *rest.HandlerDependency, c *rest.HandlerContext, input InputRestModel) http.HandlerFunc {
	return rest.ParseAccountId(d.Logger(), func(accountId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			resp := ProgressState(d.Logger(), d.DB(), d.Span(), c.Tenant())(input.SessionId, accountId, account.State(input.State))
			res, err := model.Map(model.FixedProvider(resp), Transform)()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			server.Marshal[OutputRestModel](d.Logger())(w)(c.ServerInformation())(res)
		}
	})
}

func handleDeleteSession(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseAccountId(d.Logger(), func(accountId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			_ = producer.ProviderImpl(d.Logger())(d.Span())(EnvCommandTopicAccountLogout)(logoutCommandProvider(c.Tenant(), accountId))
			w.WriteHeader(http.StatusAccepted)
		}
	})
}
