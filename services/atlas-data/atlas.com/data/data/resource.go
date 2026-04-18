package data

import (
	"atlas-data/document"
	_map "atlas-data/map"
	"atlas-data/rest"
	"net/http"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitResource(db *gorm.DB) func(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(si jsonapi.ServerInformation) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			r := router.PathPrefix("/data").Subrouter()
			r.HandleFunc("/process", rest.RegisterHandler(l)(si)("process", processData(db))).Methods(http.MethodPost)
			r.HandleFunc("/status", rest.RegisterHandler(l)(si)("get_status", handleGetStatus(db))).Methods(http.MethodGet)
		}
	}
}

func processData(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			t := tenant.MustFromContext(d.Context())
			d.Logger().Debugf("Processing data for tenant [%s], region [%s], version [%d.%d].", t.Id().String(), t.Region(), t.MajorVersion(), t.MinorVersion())

			err := database.ExecuteTransaction(db.WithContext(d.Context()), func(tx *gorm.DB) error {
				if err := document.DeleteAll(d.Context())(tx); err != nil {
					return err
				}
				return _map.DeleteAllSearchIndex(d.Context())(tx)
			})
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to delete existing documents.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			err = ProcessData(d.Logger())(d.Context())
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to process data.")
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			w.WriteHeader(http.StatusAccepted)
		}
	}
}
