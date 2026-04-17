package visitor

import (
	"context"
	"fmt"
	"strconv"
	"time"

	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

type Registry struct {
	client        *goredis.Client
	characterShop *atlas.Index // characterId → shopId (reverse lookup)
}

var registry *Registry

func InitRegistry(client *goredis.Client) {
	registry = &Registry{
		client:        client,
		characterShop: atlas.NewIndex(client, "merchant", "character-shop"),
	}
}

func GetRegistry() *Registry {
	return registry
}

// shopVisitorKey returns the Redis key for the sorted set of visitors in a shop.
func shopVisitorKey(t tenant.Model, shopId uuid.UUID) string {
	return fmt.Sprintf("%s:merchant:shop-visitors:%s", t.String(), shopId.String())
}

func (r *Registry) AddVisitor(ctx context.Context, t tenant.Model, shopId uuid.UUID, characterId uint32) error {
	cidStr := strconv.FormatUint(uint64(characterId), 10)
	key := shopVisitorKey(t, shopId)
	score := float64(time.Now().UnixNano())
	if err := r.client.ZAdd(ctx, key, goredis.Z{Score: score, Member: cidStr}).Err(); err != nil {
		return err
	}
	return r.characterShop.Add(ctx, t, cidStr, shopId.String())
}

func (r *Registry) RemoveVisitor(ctx context.Context, t tenant.Model, shopId uuid.UUID, characterId uint32) error {
	cidStr := strconv.FormatUint(uint64(characterId), 10)
	key := shopVisitorKey(t, shopId)
	if err := r.client.ZRem(ctx, key, cidStr).Err(); err != nil {
		return err
	}
	return r.characterShop.Remove(ctx, t, cidStr, shopId.String())
}

func (r *Registry) GetVisitors(ctx context.Context, t tenant.Model, shopId uuid.UUID) ([]uint32, error) {
	key := shopVisitorKey(t, shopId)
	members, err := r.client.ZRangeByScore(ctx, key, &goredis.ZRangeBy{
		Min: "-inf",
		Max: "+inf",
	}).Result()
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
	key := shopVisitorKey(t, shopId)
	count, err := r.client.ZCard(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	return int(count), nil
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
	key := shopVisitorKey(t, shopId)
	_ = r.client.Del(ctx, key).Err()
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
