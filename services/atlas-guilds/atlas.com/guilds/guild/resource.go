package guild

import (
	"atlas-guilds/rest"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"net/http"
	"strconv"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(si)
			r := router.PathPrefix("/guilds").Subrouter()
			r.HandleFunc("", registerGet("get_guilds_by_member_id", handleGetGuilds(db))).Queries("filter[members.id]", "{memberId}").Methods(http.MethodGet)
			r.HandleFunc("", registerGet("get_guilds", handleGetGuilds(db))).Methods(http.MethodGet)
			r.HandleFunc("/{guildId}", registerGet("get_guild", handleGetGuild(db))).Methods(http.MethodGet)
		}
	}
}

func handleGetGuilds(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			var filters = make([]model.Filter[Model], 0)
			if memberFilter, ok := mux.Vars(r)["memberId"]; ok {
				memberId, err := strconv.Atoi(memberFilter)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				filters = append(filters, MemberFilter(uint32(memberId)))
			}

			gs, err := NewProcessor(d.Logger(), d.Context(), db).GetSlice(filters...)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			res, err := model.SliceMap(Transform)(model.FixedProvider(gs))(model.ParallelMap())()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// Marshal response
			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
		}
	}
}

func handleGetGuild(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseGuildId(d.Logger(), func(guildId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				g, err := NewProcessor(d.Logger(), d.Context(), db).GetById(guildId)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				res, err := model.Map(Transform)(model.FixedProvider(g))()
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST model.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				
				// Marshal response
				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
			}
		})
	}
}
