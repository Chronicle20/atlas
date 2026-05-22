package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"atlas-renders/character"
	"atlas-renders/mapr"
	"atlas-renders/storage"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

func main() {
	l := newLogger()
	s, err := storage.New(l, storage.ConfigFromEnv())
	if err != nil {
		l.WithError(err).Warn("storage init failed; render handlers will 503")
		s = nil
	}
	r := mux.NewRouter()
	r.Use(tenantMiddleware(l))
	r.HandleFunc("/api/wz/character/render/{tenant}/{region}/{version}/{hash}.png", character.Handler(l, s)).Methods(http.MethodGet)
	r.HandleFunc("/api/wz/map/render/{tenant}/{region}/{version}/{mapId}/{kind}.png", mapr.Handler(l, s)).Methods(http.MethodGet)
	r.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintln(w, "ok")
	})
	port := os.Getenv("REST_PORT")
	if port == "" {
		port = "8080"
	}
	l.Infof("atlas-renders listening on :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		l.WithError(err).Fatal("server exited")
	}
}

// tenantMiddleware parses the four tenant headers (TENANT_ID, REGION,
// MAJOR_VERSION, MINOR_VERSION) and injects a tenant.Model into the request
// context so downstream handlers can call tenant.MustFromContext(ctx). The
// /healthz endpoint bypasses the check so liveness probes don't need
// tenant headers.
func tenantMiddleware(l logrus.FieldLogger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/healthz" {
				next.ServeHTTP(w, r)
				return
			}
			ctx, err := contextFromHeaders(r)
			if err != nil {
				l.WithError(err).Debug("rejecting request with invalid tenant headers")
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func contextFromHeaders(r *http.Request) (context.Context, error) {
	tenantID := r.Header.Get(tenant.ID)
	if tenantID == "" {
		return nil, fmt.Errorf("missing %s header", tenant.ID)
	}
	id, err := uuid.Parse(tenantID)
	if err != nil {
		return nil, fmt.Errorf("invalid %s: %w", tenant.ID, err)
	}
	region := r.Header.Get(tenant.Region)
	if region == "" {
		return nil, fmt.Errorf("missing %s header", tenant.Region)
	}
	major, err := strconv.ParseUint(r.Header.Get(tenant.MajorVersion), 10, 16)
	if err != nil {
		return nil, fmt.Errorf("invalid %s: %w", tenant.MajorVersion, err)
	}
	minor, err := strconv.ParseUint(r.Header.Get(tenant.MinorVersion), 10, 16)
	if err != nil {
		return nil, fmt.Errorf("invalid %s: %w", tenant.MinorVersion, err)
	}
	m, err := tenant.Create(id, region, uint16(major), uint16(minor))
	if err != nil {
		return nil, fmt.Errorf("invalid tenant headers: %w", err)
	}
	return tenant.WithContext(r.Context(), m), nil
}
