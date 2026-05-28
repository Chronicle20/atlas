// Package v9 is a minimal stub of github.com/redis/go-redis/v9 for analysistest.
package v9

import "context"

// Result types
type IntCmd struct{}
type StringCmd struct{}
type StatusCmd struct{}
type ScanCmd struct{}
type PipelinedFunc func(Pipeliner) error

func (c *StringCmd) Result() (string, error) { return "", nil }

// Pipeliner is the pipeline interface.
type Pipeliner interface {
	cmdable
}

// cmdable holds all key-based commands.
type cmdable interface {
	SAdd(ctx context.Context, key string, members ...interface{}) *IntCmd
	HSet(ctx context.Context, key string, values ...interface{}) *IntCmd
	Scan(ctx context.Context, cursor uint64, match string, count int64) *ScanCmd
	Get(ctx context.Context, key string) *StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration interface{}) *StatusCmd
	Del(ctx context.Context, keys ...string) *IntCmd
	Ping(ctx context.Context) *StatusCmd
}

// Client is a Redis client.
type Client struct{}

func (c *Client) SAdd(ctx context.Context, key string, members ...interface{}) *IntCmd {
	return nil
}
func (c *Client) HSet(ctx context.Context, key string, values ...interface{}) *IntCmd {
	return nil
}
func (c *Client) Scan(ctx context.Context, cursor uint64, match string, count int64) *ScanCmd {
	return nil
}
func (c *Client) Get(ctx context.Context, key string) *StringCmd {
	return nil
}
func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration interface{}) *StatusCmd {
	return nil
}
func (c *Client) Del(ctx context.Context, keys ...string) *IntCmd {
	return nil
}
func (c *Client) Ping(ctx context.Context) *StatusCmd {
	return nil
}
func (c *Client) Pipeline() Pipeliner {
	return nil
}
