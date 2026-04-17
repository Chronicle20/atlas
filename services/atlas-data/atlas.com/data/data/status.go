package data

import (
	"atlas-data/document"
	"atlas-data/rest"
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/Chronicle20/atlas-rest/server"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/jtumidanski/api2go/jsonapi"
	"gorm.io/gorm"
)

type StatusRestModel struct {
	Id             string  `json:"-"`
	DocumentCount  int64   `json:"documentCount"`
	UpdatedAt      *string `json:"updatedAt"`
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

func handleGetStatus(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			t := tenant.MustFromContext(d.Context())
			count, maxUpdated, err := queryStatus(d.Context(), db, t.Id().String())
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to read data status.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			res := StatusRestModel{
				Id:            t.Id().String(),
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

