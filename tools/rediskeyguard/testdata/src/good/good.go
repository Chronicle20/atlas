package good

import (
	"context"

	goredis "github.com/redis/go-redis/v9"
)

// Passing the client as a value is allowed.
func wire(client *goredis.Client) *goredis.Client {
	return client
}

// Non-keyed commands and pipeline construction are allowed.
func allowed(client *goredis.Client) {
	ctx := context.Background()
	_ = client.Ping(ctx)
	_ = client.Pipeline()
}
