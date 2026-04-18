package _map

import (
	"context"
	"strconv"
	"strings"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const (
	SearchMaxQueryLen = 128
	SearchMaxLimit    = 50
)

type SearchResult struct {
	Id         uint32
	Name       string
	StreetName string
}

func SearchByQuery(l logrus.FieldLogger, db *gorm.DB) func(ctx context.Context) func(q string, limit int) ([]SearchResult, error) {
	return func(ctx context.Context) func(q string, limit int) ([]SearchResult, error) {
		return func(q string, limit int) ([]SearchResult, error) {
			t := tenant.MustFromContext(ctx)
			bypass := database.WithoutTenantFilter(ctx)

			results, seen, err := runTenantQueries(db, bypass, t.Id(), q, limit)
			if err != nil {
				return nil, err
			}
			if len(results) >= limit {
				return results[:limit], nil
			}

			globalResults, _, err := runTenantQueries(db, bypass, uuid.Nil, q, limit)
			if err != nil {
				return nil, err
			}
			for _, gr := range globalResults {
				if _, ok := seen[gr.Id]; ok {
					continue
				}
				results = append(results, gr)
				if len(results) >= limit {
					break
				}
			}
			return results, nil
		}
	}
}

func runTenantQueries(db *gorm.DB, ctx context.Context, tenantId uuid.UUID, q string, limit int) ([]SearchResult, map[uint32]struct{}, error) {
	seen := make(map[uint32]struct{})
	results := make([]SearchResult, 0, limit)

	if numeric, err := strconv.Atoi(q); err == nil {
		var row SearchIndexEntity
		exactErr := db.WithContext(ctx).
			Where("tenant_id = ? AND map_id = ?", tenantId, uint32(numeric)).
			Take(&row).Error
		if exactErr == nil {
			results = append(results, SearchResult{Id: row.MapId, Name: row.Name, StreetName: row.StreetName})
			seen[row.MapId] = struct{}{}
		}
	}

	if len(results) >= limit {
		return results, seen, nil
	}

	pattern := "%" + strings.ToLower(q) + "%"
	var rows []SearchIndexEntity
	err := db.WithContext(ctx).
		Where("tenant_id = ? AND (LOWER(name) LIKE ? OR LOWER(street_name) LIKE ?)", tenantId, pattern, pattern).
		Order("name ASC, map_id ASC").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, nil, err
	}
	for _, r := range rows {
		if _, ok := seen[r.MapId]; ok {
			continue
		}
		results = append(results, SearchResult{Id: r.MapId, Name: r.Name, StreetName: r.StreetName})
		seen[r.MapId] = struct{}{}
		if len(results) >= limit {
			break
		}
	}
	return results, seen, nil
}

type SearchResultRestModel struct {
	Id         _map.Id `json:"-"`
	Name       string  `json:"name"`
	StreetName string  `json:"streetName"`
}

func (r SearchResultRestModel) GetName() string { return "maps" }
func (r SearchResultRestModel) GetID() string   { return strconv.Itoa(int(r.Id)) }

func (r *SearchResultRestModel) SetID(idStr string) error {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return err
	}
	r.Id = _map.Id(id)
	return nil
}

func (r SearchResultRestModel) GetCustomLinks(url string) jsonapi.Links {
	lnks := make(map[string]jsonapi.Link)
	lnks["self"] = jsonapi.Link{Href: url}
	return lnks
}
