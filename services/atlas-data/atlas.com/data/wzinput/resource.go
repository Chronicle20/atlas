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
			// PATCH /data/wz streams the WZ multipart body directly to MinIO via
			// uploadHandler. It deliberately uses rest.RegisterHandler (not
			// RegisterInputHandler[T]) because the request body is binary multipart
			// content, not a JSON:API envelope — there is no input model to decode.
			// RegisterInputHandler[T] would consume the body as JSON and fail; the
			// byte-stream path is the only correct shape for very large WZ uploads.
			r.HandleFunc("/wz", rest.RegisterHandler(l)(si)("wz_upload", uploadHandler(mc))).Methods(http.MethodPatch)
			r.HandleFunc("/wz", rest.RegisterHandler(l)(si)("wz_status", statusHandler(mc))).Methods(http.MethodGet)
		}
	}
}
