package data

import (
	"atlas-data/rest"
	"net/http"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitResource(db *gorm.DB) func(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(si jsonapi.ServerInformation) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			r := router.PathPrefix("/data").Subrouter()
			// POST /data/process is registered by runtime/rest.InitResource — that
			// handler creates a k8s ingest Job. The legacy in-process processData
			// was removed in task-076 F13.
			r.HandleFunc("/status", rest.RegisterHandler(l)(si)("get_status", handleGetStatus(db))).Methods(http.MethodGet)
		}
	}
}
