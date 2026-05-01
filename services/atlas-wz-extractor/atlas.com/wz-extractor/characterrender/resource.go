package characterrender

import (
	"atlas-wz-extractor/rest"
	"net/http"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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
		// ParseTenant puts the tenant on `d.Context()`, but the http.HandlerFunc
		// is invoked with the original `r` whose `r.Context()` carries the mux
		// route variables. Replacing r.Context() with d.Context() would clobber
		// the mux vars — instead, copy the tenant values onto the request's
		// existing context so both the mux vars and the tenant are available.
		return func(w http.ResponseWriter, r *http.Request) {
			t, err := tenant.FromContext(d.Context())()
			if err != nil {
				// ParseTenant ran successfully so this should not be reachable;
				// fall through to the inner handler which will reject via its
				// own tenant check.
				inner(w, r)
				return
			}
			inner(w, r.WithContext(tenant.WithContext(r.Context(), t)))
		}
	}
}
