package chat

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	goredis "github.com/redis/go-redis/v9"

	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type Registry struct {
	lines *atlas.TenantKeyedSortedSet[uint32]
}

var registry *Registry

func InitRegistry(client *goredis.Client) {
	registry = &Registry{
		lines: atlas.NewTenantKeyedSortedSet[uint32](client, "chat:recent", func(characterId uint32) string {
			return strconv.FormatUint(uint64(characterId), 10)
		}),
	}
}

func GetRegistry() *Registry {
	return registry
}

func (r *Registry) Append(ctx context.Context, t tenant.Model, line Line) error {
	loadConfig()
	member, err := json.Marshal(line)
	if err != nil {
		return err
	}
	score := float64(line.Timestamp)
	minScore := score - float64(retentionSeconds)*1000
	ttl := time.Duration(retentionSeconds) * time.Second
	return r.lines.AddBounded(ctx, t, line.SenderId, string(member), score, minScore, int64(maxLines), ttl)
}

func (r *Registry) RecentBySender(ctx context.Context, t tenant.Model, characterId uint32) ([]Line, error) {
	members, err := r.lines.Range(ctx, t, characterId)
	if err != nil {
		return nil, err
	}
	result := make([]Line, 0, len(members))
	for _, m := range members {
		var line Line
		if err := json.Unmarshal([]byte(m), &line); err != nil {
			continue
		}
		result = append(result, line)
	}
	return result, nil
}
