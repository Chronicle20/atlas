package configuration

import (
	"atlas-configurations/configuration/service/channel"
	"atlas-configurations/configuration/service/characterfactory"
	"atlas-configurations/configuration/service/login"
	"atlas-configurations/configuration/service/npcconversation"
	"atlas-configurations/configuration/service/world"
	"atlas-configurations/rest"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/manyminds/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"net/http"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			r := router.PathPrefix("/configurations/{type}").Subrouter()
			r.HandleFunc("", rest.RegisterHandler(l)(si)("get_configurations", handleGetConfigurations(db))).Queries("id", "{id}").Methods(http.MethodGet)
			r.HandleFunc("", rest.RegisterHandler(l)(si)("get_configurations", handleGetConfigurations(db))).Methods(http.MethodGet)
		}
	}
}

func handleGetConfigurations(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseConfigurationType(d.Logger(), func(theType string) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				var serviceId uuid.UUID
				var err error
				if val, ok := mux.Vars(r)["id"]; ok {
					serviceId, err = uuid.Parse(val)
					if err != nil {
						w.WriteHeader(http.StatusBadRequest)
						return
					}
				} else {
					serviceId = uuid.Nil
				}

				if theType == TypeChannelService {
					var rm channel.RestModel
					rm, err = GetChannelServiceConfiguration(d.Context())(db)(serviceId)
					if err != nil {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					server.Marshal[channel.RestModel](d.Logger())(w)(c.ServerInformation())(rm)
					return
				}
				if theType == TypeCharacterFactory {
					var rm characterfactory.RestModel
					rm, err = GetCharacterFactoryConfiguration(d.Context())(db)(serviceId)
					if err != nil {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					server.Marshal[characterfactory.RestModel](d.Logger())(w)(c.ServerInformation())(rm)
					return
				}
				if theType == TypeLoginService {
					var rm login.RestModel
					rm, err = GetLoginServiceConfiguration(d.Context())(db)(serviceId)
					if err != nil {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					server.Marshal[login.RestModel](d.Logger())(w)(c.ServerInformation())(rm)
					return
				}
				if theType == TypeNPCConversation {
					var rm npcconversation.RestModel
					rm, err = GetNPCConversationConfiguration(d.Context())(db)(serviceId)
					if err != nil {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					server.Marshal[npcconversation.RestModel](d.Logger())(w)(c.ServerInformation())(rm)
					return
				}
				if theType == TypeWorldService {
					var rm world.RestModel
					rm, err = GetWorldServiceConfiguration(d.Context())(db)(serviceId)
					if err != nil {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					server.Marshal[world.RestModel](d.Logger())(w)(c.ServerInformation())(rm)
					return
				}

				w.WriteHeader(http.StatusBadRequest)
			}
		})
	}
}
