package wzinput

import (
	"encoding/json"
	"fmt"
	"net/http"

	"atlas-data/rest"
	minio "atlas-data/storage/minio"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Status represents the aggregate status of WZ uploads in a scope.
type Status struct {
	FileCount  int    `json:"fileCount"`
	TotalBytes int64  `json:"totalBytes"`
	UpdatedAt  string `json:"updatedAt,omitempty"`
}

func statusHandler(mc *minio.Client) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if mc == nil {
				http.Error(w, "minio unavailable", http.StatusServiceUnavailable)
				return
			}
			t := tenant.MustFromContext(d.Context())
			scope, err := ResolveScope(r, t)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			prefix := fmt.Sprintf("%s/regions/%s/versions/%d.%d/", scope.Key, t.Region(), t.MajorVersion(), t.MinorVersion())
			s, err := mc.PrefixStats(r.Context(), mc.Cfg().BucketWZ, prefix)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/vnd.api+json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"type":       "wzInputStatus",
					"id":         "current",
					"attributes": Status{FileCount: s.Count, TotalBytes: s.Size, UpdatedAt: s.UpdatedAt},
				},
			})
		}
	}
}
