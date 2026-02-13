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

func InitResource(p Processor, wg *sync.WaitGroup) func(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(si jsonapi.ServerInformation) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			register := rest.RegisterHandler(l)(si)
			r := router.PathPrefix("/wz/extractions").Subrouter()
			r.HandleFunc("", register("create_extraction", handleExtract(p, wg))).Methods(http.MethodPost)
		}
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
