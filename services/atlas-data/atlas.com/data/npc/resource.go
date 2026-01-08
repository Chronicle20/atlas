package npc

import (
	"atlas-data/rest"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"net/http"
	"strconv"
)

func InitResource(db *gorm.DB) func(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(si jsonapi.ServerInformation) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(si)

			r := router.PathPrefix("/data/npcs").Subrouter()
			r.HandleFunc("", registerGet("get_npcs", handleGetNpcsRequest(db))).Methods(http.MethodGet)
			r.HandleFunc("/{npcId}", registerGet("get_npc", handleGetNpcRequest(db))).Methods(http.MethodGet)
		}
	}
}

func handleGetNpcsRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			s := NewStorage(d.Logger(), db)

			// Check for storebank filter
			query := r.URL.Query()
			storebankFilter := query.Get("filter[storebank]")

			results, err := s.GetAll(d.Context())
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to retrieve NPCs.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// Apply storebank filter if specified
			if storebankFilter == "true" {
				filtered := make([]RestModel, 0)
				for _, n := range results {
					if n.Storebank {
						filtered = append(filtered, n)
					}
				}
				results = filtered
			}

			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(results)
		}
	}
}

func handleGetNpcRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseNPC(d.Logger(), func(npcId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				s := NewStorage(d.Logger(), db)
				res, err := s.GetById(d.Context())(strconv.Itoa(int(npcId)))
				if err != nil {
					d.Logger().WithError(err).Debugf("Unable to locate NPC %d.", npcId)
					w.WriteHeader(http.StatusNotFound)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
			}
		})
	}
}
