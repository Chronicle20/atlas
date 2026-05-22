package wzinput

import (
	"net/http"

	"atlas-data/rest"
	minio "atlas-data/storage/minio"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
)

// InitResource installs PATCH and GET /data/wz routes backed by MinIO.
func InitResource(mc *minio.Client) func(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(si jsonapi.ServerInformation) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			r := router.PathPrefix("/data").Subrouter()
			r.HandleFunc("/wz", rest.RegisterHandler(l)(si)("wz_upload", uploadHandler(mc))).Methods(http.MethodPatch)
			r.HandleFunc("/wz", rest.RegisterHandler(l)(si)("wz_status", statusHandler(mc))).Methods(http.MethodGet)
		}
	}
}
