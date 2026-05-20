package seeder

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// backgroundSeeds tracks outstanding postSeed goroutines.  Tests call
// backgroundSeeds.Wait() before resetting metrics to avoid data races.
var backgroundSeeds sync.WaitGroup

// RegisterRoutes registers the POST /seed and GET /seed/status endpoints for
// the given Group onto the provided router.
func RegisterRoutes(
	router *mux.Router,
	db *gorm.DB,
	logger logrus.FieldLogger,
	src CatalogSource,
	g Group,
) {
	router.HandleFunc(g.URLPrefix+"/seed", postSeed(logger, db, src, g)).Methods(http.MethodPost)
	router.HandleFunc(g.URLPrefix+"/seed/status", getStatus(logger, db, src, g)).Methods(http.MethodGet)
}

func postSeed(l logrus.FieldLogger, db *gorm.DB, src CatalogSource, g Group) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t := tenant.MustFromContext(r.Context())
		backgroundSeeds.Add(1)
		go func() {
			defer backgroundSeeds.Done()
			bgCtx := tenant.WithContext(context.Background(), t)
			res, err := Seed(bgCtx, db, src, g)
			if err != nil {
				l.WithError(err).WithFields(logrus.Fields{
					"tenant_id":  t.Id(),
					"group_name": g.Name,
				}).Error("Seed failed")
				return
			}
			l.WithFields(logrus.Fields{
				"tenant_id":        t.Id(),
				"group_name":       g.Name,
				"catalog_revision": res.CatalogRevision,
				"subdomains":       summarize(res.Subdomains),
			}).Info("Seed complete")
		}()
		w.WriteHeader(http.StatusAccepted)
	}
}

func getStatus(l logrus.FieldLogger, db *gorm.DB, src CatalogSource, g Group) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		st, err := ReadStatus(r.Context(), db, src, g)
		if err != nil {
			l.WithError(err).WithField("group_name", g.Name).Error("ReadStatus failed")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if st.TenantSeededRevision != nil && st.CatalogRevision != "" && st.CatalogRevision != *st.TenantSeededRevision {
			l.WithFields(logrus.Fields{
				"group_name":             g.Name,
				"catalog_revision":       st.CatalogRevision,
				"tenant_seeded_revision": *st.TenantSeededRevision,
			}).Warn("seed catalog drift detected")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(st)
	}
}

func summarize(m map[string]SubdomainCounts) map[string]int64 {
	out := make(map[string]int64, len(m))
	for k, v := range m {
		out[k] = v.Created
	}
	return out
}
