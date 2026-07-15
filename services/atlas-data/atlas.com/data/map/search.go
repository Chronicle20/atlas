package _map

import (
	"context"
	"strconv"

	"atlas-data/searchindex"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const (
	SearchMaxQueryLen = searchindex.MaxQueryLen
	SearchMaxLimit    = searchindex.MaxLimit
)

type SearchResult struct {
	Id         uint32
	Name       string
	StreetName string
}

func mapSearchSpec() searchindex.QuerySpec[SearchIndexEntity] {
	return searchindex.QuerySpec[SearchIndexEntity]{
		EntityIdColumn: "map_id",
		NameColumns:    []string{"name", "street_name"},
		Order:          "name ASC, map_id ASC",
	}
}

func SearchByQuery(_ logrus.FieldLogger, db *gorm.DB) func(ctx context.Context) func(q string, offset, limit int) ([]SearchResult, error) {
	return func(ctx context.Context) func(q string, offset, limit int) ([]SearchResult, error) {
		return func(q string, offset, limit int) ([]SearchResult, error) {
			spec := mapSearchSpec()
			tenantId, err := searchindex.ResolveTenantId(db, ctx, spec)
			if err != nil {
				return nil, err
			}
			rows, err := searchindex.Search[SearchIndexEntity](db, ctx, tenantId, q, offset, limit, spec)
			if err != nil {
				return nil, err
			}
			out := make([]SearchResult, 0, len(rows))
			for _, r := range rows {
				out = append(out, SearchResult{Id: r.MapId, Name: r.Name, StreetName: r.StreetName})
			}
			return out, nil
		}
	}
}

// CountByQuery returns the total row count matching q for the map search
// index, mirroring SearchByQuery's tenant resolution so the pagination
// envelope's meta.total agrees with the paged result set.
func CountByQuery(db *gorm.DB) func(ctx context.Context) func(q string) (int, error) {
	return func(ctx context.Context) func(q string) (int, error) {
		return func(q string) (int, error) {
			spec := mapSearchSpec()
			tenantId, err := searchindex.ResolveTenantId(db, ctx, spec)
			if err != nil {
				return 0, err
			}
			return searchindex.Count(db, ctx, tenantId, q, spec)
		}
	}
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
