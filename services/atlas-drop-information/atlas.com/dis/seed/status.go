package seed

import (
	continentdrop "atlas-drops-information/continent/drop"
	monsterdrop "atlas-drops-information/monster/drop"
	reactordrop "atlas-drops-information/reactor/drop"
	"atlas-drops-information/rest"
	"net/http"
	"sync"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/jtumidanski/api2go/jsonapi"
	"golang.org/x/sync/errgroup"
)

type DropsSeedStatusRestModel struct {
	Id                 string  `json:"-"`
	MonsterDropCount   int64   `json:"monsterDropCount"`
	ContinentDropCount int64   `json:"continentDropCount"`
	ReactorDropCount   int64   `json:"reactorDropCount"`
	UpdatedAt          *string `json:"updatedAt"`
}

func (r DropsSeedStatusRestModel) GetName() string        { return "dropsSeedStatus" }
func (r DropsSeedStatusRestModel) GetID() string          { return r.Id }
func (r *DropsSeedStatusRestModel) SetID(id string) error { r.Id = id; return nil }
func (r DropsSeedStatusRestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{}
}
func (r DropsSeedStatusRestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	return []jsonapi.ReferenceID{}
}
func (r DropsSeedStatusRestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	return []jsonapi.MarshalIdentifier{}
}
func (r *DropsSeedStatusRestModel) SetToOneReferenceID(_, _ string) error { return nil }
func (r *DropsSeedStatusRestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}
func (r *DropsSeedStatusRestModel) SetReferencedStructs(_ map[string]map[string]jsonapi.Data) error {
	return nil
}

type subcount struct {
	count     int64
	updatedAt *time.Time
}

func handleGetSeedStatus(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := d.Logger()
		db := d.DB()
		t := tenant.MustFromContext(d.Context())

		var mu sync.Mutex
		var monster, continent, reactor subcount

		g, gctx := errgroup.WithContext(d.Context())
		g.Go(func() error {
			count, updated, err := monsterdrop.NewProcessor(l, gctx, db).Count()
			if err != nil {
				return err
			}
			mu.Lock()
			monster = subcount{count: count, updatedAt: updated}
			mu.Unlock()
			return nil
		})
		g.Go(func() error {
			count, updated, err := continentdrop.NewProcessor(l, gctx, db).Count()
			if err != nil {
				return err
			}
			mu.Lock()
			continent = subcount{count: count, updatedAt: updated}
			mu.Unlock()
			return nil
		})
		g.Go(func() error {
			count, updated, err := reactordrop.NewProcessor(l, gctx, db).Count()
			if err != nil {
				return err
			}
			mu.Lock()
			reactor = subcount{count: count, updatedAt: updated}
			mu.Unlock()
			return nil
		})

		if err := g.Wait(); err != nil {
			l.WithError(err).Errorf("Unable to read drops seed status.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		res := DropsSeedStatusRestModel{
			Id:                 t.Id().String(),
			MonsterDropCount:   monster.count,
			ContinentDropCount: continent.count,
			ReactorDropCount:   reactor.count,
			UpdatedAt:          maxUpdatedAtRFC3339(monster.updatedAt, continent.updatedAt, reactor.updatedAt),
		}

		query := r.URL.Query()
		queryParams := jsonapi.ParseQueryFields(&query)
		server.MarshalResponse[DropsSeedStatusRestModel](l)(w)(c.ServerInformation())(queryParams)(res)
	}
}

func maxUpdatedAtRFC3339(parts ...*time.Time) *string {
	var max *time.Time
	for _, p := range parts {
		if p == nil {
			continue
		}
		if max == nil || p.After(*max) {
			max = p
		}
	}
	if max == nil {
		return nil
	}
	s := max.UTC().Format(time.RFC3339)
	return &s
}
