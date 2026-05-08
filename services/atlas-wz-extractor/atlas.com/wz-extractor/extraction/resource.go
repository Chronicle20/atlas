package extraction

import (
	"atlas-wz-extractor/extraction/job"
	"atlas-wz-extractor/extraction/lock"
	"atlas-wz-extractor/rest"
	"net/http"
	"sync"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
)

type Dirs struct {
	InputDir     string
	OutputXmlDir string
	OutputImgDir string
}

// InitResource wires the WZ extraction REST surface. The wg parameter is kept
// for API compatibility with the previous synchronous-goroutine handler;
// under the Kafka-backed model it is effectively a no-op (the unit work runs
// in consumer pods, not on a goroutine here). Removal is planned as a
// follow-up — see design.md §11.
func InitResource(p Processor, store job.Store, tl *lock.TenantLock, prod producerProvider, wg *sync.WaitGroup, dirs Dirs) func(si jsonapi.ServerInformation) server.RouteInitializer {
	u := &uploadDeps{inputDir: dirs.InputDir, tl: tl}
	s := &statusDeps{inputDir: dirs.InputDir, outputXmlDir: dirs.OutputXmlDir}
	_ = wg
	return func(si jsonapi.ServerInformation) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			register := rest.RegisterHandler(l)(si)

			ext := router.PathPrefix("/wz/extractions").Subrouter()
			ext.HandleFunc("", register("create_extraction", handleExtract(p, store, tl, prod, dirs))).Methods(http.MethodPost)
			ext.HandleFunc("", register("get_extraction_status", s.handleExtractionStatus())).Methods(http.MethodGet)
			ext.HandleFunc("/jobs/{jobId}", register("get_extraction_job", handleJobStatus(store))).Methods(http.MethodGet)

			in := router.PathPrefix("/wz/input").Subrouter()
			in.HandleFunc("", register("upload_wz", u.handleUploadBridge())).Methods(http.MethodPatch)
			in.HandleFunc("", register("get_input_status", s.handleInputStatus())).Methods(http.MethodGet)
		}
	}
}

func (u *uploadDeps) handleUploadBridge() rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return u.handleUpload(d.Logger(), d.Context())
	}
}

func (s *statusDeps) handleInputStatus() rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return s.renderInputStatus(d.Logger(), d.Context())
	}
}

func (s *statusDeps) handleExtractionStatus() rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return s.renderExtractionStatus(d.Logger(), d.Context())
	}
}
