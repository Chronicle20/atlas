package data

import (
	"atlas-data/document"
	"atlas-data/rest"
	"net/http"

	"github.com/Chronicle20/atlas-rest/server"
	"github.com/Chronicle20/atlas-tenant"
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
		}
	}
}

func processData(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			t := tenant.MustFromContext(d.Context())
			d.Logger().Debugf("Processing data for tenant [%s], region [%s], version [%d.%d].", t.Id().String(), t.Region(), t.MajorVersion(), t.MinorVersion())

			err := document.DeleteAll(d.Context())(db)
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
