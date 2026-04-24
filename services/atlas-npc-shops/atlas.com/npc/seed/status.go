package seed

import (
	"atlas-npc/commodities"
	"atlas-npc/rest"
	"atlas-npc/shops"
	"net/http"
	"sync"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/jtumidanski/api2go/jsonapi"
	"golang.org/x/sync/errgroup"
)

type NpcShopsSeedStatusRestModel struct {
	Id             string  `json:"-"`
	ShopCount      int64   `json:"shopCount"`
	CommodityCount int64   `json:"commodityCount"`
	UpdatedAt      *string `json:"updatedAt"`
}

func (r NpcShopsSeedStatusRestModel) GetName() string        { return "npcShopsSeedStatus" }
func (r NpcShopsSeedStatusRestModel) GetID() string          { return r.Id }
func (r *NpcShopsSeedStatusRestModel) SetID(id string) error { r.Id = id; return nil }
func (r NpcShopsSeedStatusRestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{}
}
func (r NpcShopsSeedStatusRestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	return []jsonapi.ReferenceID{}
}
func (r NpcShopsSeedStatusRestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	return []jsonapi.MarshalIdentifier{}
}
func (r *NpcShopsSeedStatusRestModel) SetToOneReferenceID(_, _ string) error { return nil }
func (r *NpcShopsSeedStatusRestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}
func (r *NpcShopsSeedStatusRestModel) SetReferencedStructs(_ map[string]map[string]jsonapi.Data) error {
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
		var shopsSub, commoditiesSub subcount

		g, gctx := errgroup.WithContext(d.Context())
		g.Go(func() error {
			count, updated, err := shops.NewProcessor(l, gctx, db).Count()
			if err != nil {
				return err
			}
			mu.Lock()
			shopsSub = subcount{count: count, updatedAt: updated}
			mu.Unlock()
			return nil
		})
		g.Go(func() error {
			count, updated, err := commodities.NewProcessor(l, gctx, db).Count()
			if err != nil {
				return err
			}
			mu.Lock()
			commoditiesSub = subcount{count: count, updatedAt: updated}
			mu.Unlock()
			return nil
		})

		if err := g.Wait(); err != nil {
			l.WithError(err).Errorf("Unable to read npc shops seed status.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		res := NpcShopsSeedStatusRestModel{
			Id:             t.Id().String(),
			ShopCount:      shopsSub.count,
			CommodityCount: commoditiesSub.count,
			UpdatedAt:      maxUpdatedAtRFC3339(shopsSub.updatedAt, commoditiesSub.updatedAt),
		}

		query := r.URL.Query()
		queryParams := jsonapi.ParseQueryFields(&query)
		server.MarshalResponse[NpcShopsSeedStatusRestModel](l)(w)(c.ServerInformation())(queryParams)(res)
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
