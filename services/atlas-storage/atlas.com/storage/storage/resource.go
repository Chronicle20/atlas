package storage

import (
	"atlas-storage/rest"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"net/http"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(si)
			registerPost := rest.RegisterHandler(l)(si)

			r := router.PathPrefix("/storage/accounts").Subrouter()
			r.HandleFunc("/{accountId}", registerGet("get_storage", handleGetStorageRequest(db))).Methods(http.MethodGet)
			r.HandleFunc("/{accountId}", registerPost("create_storage", handleCreateStorageRequest(db))).Methods(http.MethodPost)
		}
	}
}

func handleGetStorageRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseAccountId(d.Logger(), func(accountId uint32) http.HandlerFunc {
			return rest.ParseWorldId(d.Logger(), func(worldId byte) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					t := tenant.MustFromContext(d.Context())

					s, err := GetByWorldAndAccountId(d.Logger(), db, t.Id())(worldId, accountId)
					if err != nil {
						d.Logger().WithError(err).Debugf("Unable to locate storage for world %d account %d.", worldId, accountId)
						w.WriteHeader(http.StatusNotFound)
						return
					}

					query := r.URL.Query()
					queryParams := jsonapi.ParseQueryFields(&query)
					server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(Transform(s))
				}
			})
		})
	}
}

func handleCreateStorageRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseAccountId(d.Logger(), func(accountId uint32) http.HandlerFunc {
			return rest.ParseWorldId(d.Logger(), func(worldId byte) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					t := tenant.MustFromContext(d.Context())

					// Check if storage already exists
					_, err := GetByWorldAndAccountId(d.Logger(), db, t.Id())(worldId, accountId)
					if err == nil {
						// Storage already exists
						d.Logger().Debugf("Storage already exists for world %d account %d.", worldId, accountId)
						w.WriteHeader(http.StatusConflict)
						return
					}

					// Create new storage
					s, err := Create(d.Logger(), db, t.Id())(worldId, accountId)
					if err != nil {
						d.Logger().WithError(err).Errorf("Unable to create storage for world %d account %d.", worldId, accountId)
						w.WriteHeader(http.StatusInternalServerError)
						return
					}

					query := r.URL.Query()
					queryParams := jsonapi.ParseQueryFields(&query)
					w.WriteHeader(http.StatusCreated)
					server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(Transform(s))
				}
			})
		})
	}
}
