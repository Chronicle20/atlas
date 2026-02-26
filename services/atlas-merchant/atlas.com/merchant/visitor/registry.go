package visitor

import (
	"context"
	"strconv"

	atlas "github.com/Chronicle20/atlas-redis"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

type Registry struct {
	shopVisitors  *atlas.Index // shopId → set of characterIds
	characterShop *atlas.Index // characterId → shopId (reverse lookup)
}

var registry *Registry

func InitRegistry(client *goredis.Client) {
	registry = &Registry{
		shopVisitors:  atlas.NewIndex(client, "merchant", "shop-visitors"),
		characterShop: atlas.NewIndex(client, "merchant", "character-shop"),
	}
}

func GetRegistry() *Registry {
	return registry
}

func (r *Registry) AddVisitor(ctx context.Context, t tenant.Model, shopId uuid.UUID, characterId uint32) error {
	cidStr := strconv.FormatUint(uint64(characterId), 10)
	if err := r.shopVisitors.Add(ctx, t, shopId.String(), cidStr); err != nil {
		return err
	}
	return r.characterShop.Add(ctx, t, cidStr, shopId.String())
}

func (r *Registry) RemoveVisitor(ctx context.Context, t tenant.Model, shopId uuid.UUID, characterId uint32) error {
	cidStr := strconv.FormatUint(uint64(characterId), 10)
	if err := r.shopVisitors.Remove(ctx, t, shopId.String(), cidStr); err != nil {
		return err
	}
	return r.characterShop.Remove(ctx, t, cidStr, shopId.String())
}

func (r *Registry) GetVisitors(ctx context.Context, t tenant.Model, shopId uuid.UUID) ([]uint32, error) {
	members, err := r.shopVisitors.Lookup(ctx, t, shopId.String())
	if err != nil {
		return nil, err
	}
	result := make([]uint32, 0, len(members))
	for _, m := range members {
		id, err := strconv.ParseUint(m, 10, 32)
		if err != nil {
			continue
		}
		result = append(result, uint32(id))
	}
	return result, nil
}

func (r *Registry) GetVisitorCount(ctx context.Context, t tenant.Model, shopId uuid.UUID) (int, error) {
	members, err := r.shopVisitors.Lookup(ctx, t, shopId.String())
	if err != nil {
		return 0, err
	}
	return len(members), nil
}

func (r *Registry) RemoveAllVisitors(ctx context.Context, t tenant.Model, shopId uuid.UUID) ([]uint32, error) {
	visitors, err := r.GetVisitors(ctx, t, shopId)
	if err != nil {
		return nil, err
	}
	for _, characterId := range visitors {
		cidStr := strconv.FormatUint(uint64(characterId), 10)
		_ = r.characterShop.Remove(ctx, t, cidStr, shopId.String())
	}
	_ = r.shopVisitors.RemoveAll(ctx, t, shopId.String())
	return visitors, nil
}

func (r *Registry) GetShopForCharacter(ctx context.Context, t tenant.Model, characterId uint32) (uuid.UUID, error) {
	cidStr := strconv.FormatUint(uint64(characterId), 10)
	shopIdStr, err := r.characterShop.LookupOne(ctx, t, cidStr)
	if err != nil {
		return uuid.Nil, err
	}
	return uuid.Parse(shopIdStr)
}
