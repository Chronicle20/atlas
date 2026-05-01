package characterrender

import (
	"atlas-wz-extractor/rest"
	"net/http"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
)

// InitResource registers the render route.
func InitResource(h *Handler) func(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(si jsonapi.ServerInformation) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			register := rest.RegisterHandler(l)(si)
			ren := router.PathPrefix("/wz/character").Subrouter()
			ren.HandleFunc(
				"/render/{tenant}/{region}/{version}/{hash}.png",
				register("render_character", h.handleRenderBridge()),
			).Methods(http.MethodGet)
		}
	}
}

func (h *Handler) handleRenderBridge() rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		inner := h.HandleRender(d.Logger())
		// ParseTenant only puts the tenant on `d.Context()`; the inner handler
		// reads `r.Context()` to populate observability fields and to verify the
		// path tenant against the header tenant. Inject the tenant-aware ctx
		// into the request before delegating.
		return func(w http.ResponseWriter, r *http.Request) {
			inner(w, r.WithContext(d.Context()))
		}
	}
}
