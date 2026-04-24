package seed

import (
	"atlas-gachapons/gachapon"
	"atlas-gachapons/global"
	"atlas-gachapons/item"
	"atlas-gachapons/rest"
	"net/http"
	"sync"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/jtumidanski/api2go/jsonapi"
	"golang.org/x/sync/errgroup"
)

type GachaponsSeedStatusRestModel struct {
	Id              string  `json:"-"`
	GachaponCount   int64   `json:"gachaponCount"`
	ItemCount       int64   `json:"itemCount"`
	GlobalItemCount int64   `json:"globalItemCount"`
	UpdatedAt       *string `json:"updatedAt"`
}

func (r GachaponsSeedStatusRestModel) GetName() string        { return "gachaponsSeedStatus" }
func (r GachaponsSeedStatusRestModel) GetID() string          { return r.Id }
func (r *GachaponsSeedStatusRestModel) SetID(id string) error { r.Id = id; return nil }
func (r GachaponsSeedStatusRestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{}
}
func (r GachaponsSeedStatusRestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	return []jsonapi.ReferenceID{}
}
func (r GachaponsSeedStatusRestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	return []jsonapi.MarshalIdentifier{}
}
func (r *GachaponsSeedStatusRestModel) SetToOneReferenceID(_, _ string) error { return nil }
func (r *GachaponsSeedStatusRestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}
func (r *GachaponsSeedStatusRestModel) SetReferencedStructs(_ map[string]map[string]jsonapi.Data) error {
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
		var gachapons, items, globals subcount

		g, gctx := errgroup.WithContext(d.Context())
		g.Go(func() error {
			count, updated, err := gachapon.NewProcessor(l, gctx, db).Count()
			if err != nil {
				return err
			}
			mu.Lock()
			gachapons = subcount{count: count, updatedAt: updated}
			mu.Unlock()
			return nil
		})
		g.Go(func() error {
			count, updated, err := item.NewProcessor(l, gctx, db).Count()
			if err != nil {
				return err
			}
			mu.Lock()
			items = subcount{count: count, updatedAt: updated}
			mu.Unlock()
			return nil
		})
		g.Go(func() error {
			count, updated, err := global.NewProcessor(l, gctx, db).Count()
			if err != nil {
				return err
			}
			mu.Lock()
			globals = subcount{count: count, updatedAt: updated}
			mu.Unlock()
			return nil
		})

		if err := g.Wait(); err != nil {
			l.WithError(err).Errorf("Unable to read gachapons seed status.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		res := GachaponsSeedStatusRestModel{
			Id:              t.Id().String(),
			GachaponCount:   gachapons.count,
			ItemCount:       items.count,
			GlobalItemCount: globals.count,
			UpdatedAt:       maxUpdatedAtRFC3339(gachapons.updatedAt, items.updatedAt, globals.updatedAt),
		}

		query := r.URL.Query()
		queryParams := jsonapi.ParseQueryFields(&query)
		server.MarshalResponse[GachaponsSeedStatusRestModel](l)(w)(c.ServerInformation())(queryParams)(res)
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
