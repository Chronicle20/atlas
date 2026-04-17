package extraction

import (
	"atlas-wz-extractor/rest"
	"context"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/Chronicle20/atlas-rest/server"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
)

type Dirs struct {
	InputDir     string
	OutputXmlDir string
}

func InitResource(p Processor, wg *sync.WaitGroup, dirs Dirs) func(si jsonapi.ServerInformation) server.RouteInitializer {
	u := &uploadDeps{inputDir: dirs.InputDir}
	s := &statusDeps{inputDir: dirs.InputDir, outputXmlDir: dirs.OutputXmlDir}
	return func(si jsonapi.ServerInformation) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			register := rest.RegisterHandler(l)(si)

			ext := router.PathPrefix("/wz/extractions").Subrouter()
			ext.HandleFunc("", register("create_extraction", handleExtract(p, wg))).Methods(http.MethodPost)
			ext.HandleFunc("", register("get_extraction_status", s.handleExtractionStatus())).Methods(http.MethodGet)

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

func handleExtract(p Processor, wg *sync.WaitGroup) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			t := tenant.MustFromContext(d.Context())
			xmlOnly := r.URL.Query().Get("xmlOnly") == "true"
			imagesOnly := r.URL.Query().Get("imagesOnly") == "true"

			d.Logger().Infof("Starting extraction for tenant [%s], region [%s], version [%d.%d].",
				t.Id().String(), t.Region(), t.MajorVersion(), t.MinorVersion())

			asyncCtx := tenant.WithContext(context.Background(), t)

			wg.Add(1)
			go func() {
				defer wg.Done()
				key := TenantKey(t)
				m := Acquire(key)
				defer Release(m)
				if err := p.Extract(d.Logger(), asyncCtx, xmlOnly, imagesOnly); err != nil {
					d.Logger().WithError(err).Errorf("Extraction failed.")
				} else {
					d.Logger().Infof("Extraction completed successfully.")
				}
			}()

			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "started"})
		}
	}
}
