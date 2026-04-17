package shop

import (
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

type Registry struct {
	activeShops   *atlas.TenantRegistry[uint32, ActiveShopEntry]
	mapPlacement  *atlas.Index // mapId → set of shopIds
	client        *goredis.Client
}

type ActiveShopEntry struct {
	ShopId     uuid.UUID  `json:"shopId"`
	ShopType   ShopType   `json:"shopType"`
	WorldId    world.Id   `json:"worldId"`
	ChannelId  channel.Id `json:"channelId"`
	MapId      uint32     `json:"mapId"`
	InstanceId uuid.UUID  `json:"instanceId"`
}

var registry *Registry

func InitRegistry(client *goredis.Client) {
	registry = &Registry{
		activeShops: atlas.NewTenantRegistry[uint32, ActiveShopEntry](client, "merchant-active", func(k uint32) string {
			return strconv.FormatUint(uint64(k), 10)
		}),
		mapPlacement: atlas.NewIndex(client, "merchant", "map-shops"),
		client:       client,
	}
}

func GetRegistry() *Registry {
	return registry
}
