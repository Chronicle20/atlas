package data

import (
	"atlas-data/canonical"
	"atlas-data/document"
	"atlas-data/rest"
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/jtumidanski/api2go/jsonapi"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type StatusRestModel struct {
	Id            string  `json:"-"`
	DocumentCount int64   `json:"documentCount"`
	UpdatedAt     *string `json:"updatedAt"`
}

func (r StatusRestModel) GetName() string {
	return "dataStatus"
}

func (r StatusRestModel) GetID() string {
	return r.Id
}

func (r *StatusRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func (r StatusRestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{}
}

func (r StatusRestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	return []jsonapi.ReferenceID{}
}

func (r StatusRestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	return []jsonapi.MarshalIdentifier{}
}

func (r *StatusRestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *StatusRestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

func (r *StatusRestModel) SetReferencedStructs(_ map[string]map[string]jsonapi.Data) error {
	return nil
}

func queryStatus(ctx context.Context, db *gorm.DB, tenantId string) (int64, *time.Time, error) {
	var count int64
	if err := db.WithContext(ctx).
		Model(&document.Entity{}).
		Where("tenant_id = ?", tenantId).
		Count(&count).Error; err != nil {
		return 0, nil, err
	}
	if count == 0 {
		return 0, nil, nil
	}
	row := db.WithContext(ctx).
		Model(&document.Entity{}).
		Where("tenant_id = ?", tenantId).
		Select("MAX(updated_at)").
		Row()
	var raw sql.NullString
	if err := row.Scan(&raw); err != nil {
		return 0, nil, err
	}
	if !raw.Valid || raw.String == "" {
		return count, nil, nil
	}
	t, err := parseDBTime(raw.String)
	if err != nil {
		return 0, nil, err
	}
	if t.IsZero() {
		return count, nil, nil
	}
	return count, &t, nil
}

func parseDBTime(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999 -0700 MST",
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, nil
}

// resolveStatusTenantId maps the ?scope= query parameter to the tenant_id the
// ingest wrote documents under, mirroring workers.tenantFromParams: the default
// (empty or "tenant") scope reads the active tenant's rows, while "shared" reads
// the canonical baseline rows anchored to the version-scoped canonical id
// (canonical.TenantId(region, major, minor)). The shared scope is gated behind
// operator credentials, matching wzinput.ResolveScope. On an invalid request it
// writes the HTTP error and returns ok=false.
func resolveStatusTenantId(w http.ResponseWriter, r *http.Request, t tenant.Model) (string, bool) {
	switch r.URL.Query().Get("scope") {
	case "", "tenant":
		return t.Id().String(), true
	case "shared":
		if r.Header.Get("X-Atlas-Operator") != "1" {
			http.Error(w, "operator required", http.StatusForbidden)
			return "", false
		}
		return canonical.TenantId(t.Region(), t.MajorVersion(), t.MinorVersion()).String(), true
	default:
		http.Error(w, "invalid scope", http.StatusBadRequest)
		return "", false
	}
}

func handleGetStatus(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			t := tenant.MustFromContext(d.Context())
			tenantId, ok := resolveStatusTenantId(w, r, t)
			if !ok {
				return
			}
			// queryStatus filters tenant_id explicitly to the resolved scope,
			// so bypass the automatic context-tenant filter — otherwise a
			// scope=shared read would be AND-ed with the caller's tenant_id and
			// never see the canonical baseline rows.
			ctx := database.WithoutTenantFilter(d.Context())
			count, maxUpdated, err := queryStatus(ctx, db, tenantId)
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to read data status.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			res := StatusRestModel{
				Id:            tenantId,
				DocumentCount: count,
			}
			if maxUpdated != nil {
				s := maxUpdated.UTC().Format(time.RFC3339)
				res.UpdatedAt = &s
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[StatusRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
		}
	}
}
