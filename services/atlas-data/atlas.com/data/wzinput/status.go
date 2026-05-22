package wzinput

import (
	"fmt"
	"net/http"

	"atlas-data/rest"
	minio "atlas-data/storage/minio"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/jtumidanski/api2go/jsonapi"
)

// Status represents the aggregate status of WZ uploads in a scope.
type Status struct {
	FileCount  int    `json:"fileCount"`
	TotalBytes int64  `json:"totalBytes"`
	UpdatedAt  string `json:"updatedAt,omitempty"`
}

// GetName returns the JSON:API resource type. Matches the pre-F14 wire
// shape "wzInputStatus".
func (s Status) GetName() string { return "wzInputStatus" }

// GetID returns the JSON:API resource id. The status endpoint exposes a
// single per-scope singleton resource, "current".
func (s Status) GetID() string { return "current" }

// SetID satisfies the JSON:API UnmarshalIdentifier interface; the id is
// fixed at "current" so incoming values are intentionally ignored.
func (s *Status) SetID(string) error { return nil }

func (s Status) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{}
}

func (s Status) GetReferencedIDs() []jsonapi.ReferenceID {
	return []jsonapi.ReferenceID{}
}

func (s Status) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	return []jsonapi.MarshalIdentifier{}
}

func (s *Status) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (s *Status) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

func (s *Status) SetReferencedStructs(_ map[string]map[string]jsonapi.Data) error {
	return nil
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
			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			w.Header().Set("Content-Type", "application/vnd.api+json")
			server.MarshalResponse[Status](d.Logger())(w)(c.ServerInformation())(queryParams)(
				Status{FileCount: s.Count, TotalBytes: s.Size, UpdatedAt: s.UpdatedAt},
			)
		}
	}
}
