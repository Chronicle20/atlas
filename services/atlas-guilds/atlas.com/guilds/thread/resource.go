package thread

import (
	"atlas-guilds/rest"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/manyminds/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"net/http"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			r := router.PathPrefix("/guilds/{guildId}/threads").Subrouter()
			r.HandleFunc("", rest.RegisterHandler(l)(si)("get_guild_threads", handleGetGuildThreads(db))).Methods(http.MethodGet)
			r.HandleFunc("/{threadId}", rest.RegisterHandler(l)(si)("get_guild_thread", handleGetGuildThread(db))).Methods(http.MethodGet)
		}
	}
}

func handleGetGuildThreads(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseGuildId(d.Logger(), func(guildId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				thrs, err := GetAll(d.Context())(db)(guildId)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				res, err := model.SliceMap(Transform)(model.FixedProvider(thrs))(model.ParallelMap())()
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST model.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				server.Marshal[[]RestModel](d.Logger())(w)(c.ServerInformation())(res)
			}
		})
	}
}

func handleGetGuildThread(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseGuildId(d.Logger(), func(guildId uint32) http.HandlerFunc {
			return rest.ParseThreadId(d.Logger(), func(threadId uint32) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					thr, err := GetById(d.Context())(db)(guildId, threadId)
					if err != nil {
						w.WriteHeader(http.StatusInternalServerError)
						return
					}

					res, err := model.Map(Transform)(model.FixedProvider(thr))()
					if err != nil {
						d.Logger().WithError(err).Errorf("Creating REST model.")
						w.WriteHeader(http.StatusInternalServerError)
						return
					}

					server.Marshal[RestModel](d.Logger())(w)(c.ServerInformation())(res)
				}
			})
		})
	}
}
