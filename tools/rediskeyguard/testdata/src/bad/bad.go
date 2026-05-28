package bad

import (
	"context"

	goredis "github.com/redis/go-redis/v9"
)

func useRaw(client *goredis.Client) {
	ctx := context.Background()
	client.SAdd(ctx, "drops:all", "x")   // want `rediskeyguard: SAdd called on raw go-redis client`
	client.HSet(ctx, "h", "f", "v")      // want `rediskeyguard: HSet called on raw go-redis client`
	client.Scan(ctx, 0, "pat:*", 100)   // want `rediskeyguard: Scan called on raw go-redis client`
	_, _ = client.Get(ctx, "k").Result() // want `rediskeyguard: Get called on raw go-redis client`
}
