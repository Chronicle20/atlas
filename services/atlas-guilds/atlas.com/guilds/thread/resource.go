package thread

import (
	"atlas-guilds/rest"
	"net/http"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
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
				page, err := paginate.ParseParams(r.URL.Query(), paginate.DefaultPageSize, paginate.MaxPageSize)
				if err != nil {
					server.WriteBadRequest(d.Logger(), w, err.Error())
					return
				}

				paged, err := NewProcessor(d.Logger(), d.Context(), db).AllProvider(guildId, page)()
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to locate threads for guild [%d].", guildId)
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				res, err := model.SliceMap(Transform)(model.FixedProvider(paged.Items))(model.ParallelMap())()
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST model.")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				// Marshal response
				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalPaginatedResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res, paginate.EnvelopeFor(paged), r)
			}
		})
	}
}

func handleGetGuildThread(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseGuildId(d.Logger(), func(guildId uint32) http.HandlerFunc {
			return rest.ParseThreadId(d.Logger(), func(threadId uint32) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					thr, err := NewProcessor(d.Logger(), d.Context(), db).GetById(guildId, threadId)
					if err != nil {
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}

					res, err := model.Map(Transform)(model.FixedProvider(thr))()
					if err != nil {
						d.Logger().WithError(err).Errorf("Creating REST model.")
						server.WriteErrorResponse(d.Logger())(w)(err)
						return
					}

					// Marshal response
					query := r.URL.Query()
					queryParams := jsonapi.ParseQueryFields(&query)
					server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
				}
			})
		})
	}
}
