package _map

import (
	"context"
	"sync"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/sirupsen/logrus"
)

// Processor provides operations for querying map player counts
type Processor interface {
	GetPlayerCountInField(f field.Model) (int, error)
	GetPlayerCountsInMaps(worldId world.Id, channelId channel.Id, mapIds []_map.Id) (map[_map.Id]int, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewProcessor creates a new map processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

// GetPlayerCountInField retrieves the player count for a single field
// Returns 0 on error to allow graceful degradation
func (p *ProcessorImpl) GetPlayerCountInField(f field.Model) (int, error) {
	resp, err := requestCharactersInMap(f)(p.l, p.ctx)
	if err != nil {
		p.l.WithError(err).Warnf("Failed to get characters in map [%d], using count 0", f.MapId())
		return 0, nil
	}
	count := len(resp)
	p.l.Debugf("Map [%d] has %d players", f.MapId(), count)
	return count, nil
}

// GetPlayerCountsInMaps retrieves player counts for multiple maps in parallel
// Returns a map of mapId -> playerCount
// Uses graceful degradation - returns 0 for maps that fail to query
// Note: Uses uuid.Nil for instance when querying arbitrary maps
func (p *ProcessorImpl) GetPlayerCountsInMaps(worldId world.Id, channelId channel.Id, mapIds []_map.Id) (map[_map.Id]int, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	counts := make(map[_map.Id]int)

	for _, mapId := range mapIds {
		wg.Add(1)
		go func(id _map.Id) {
			defer wg.Done()
			f := field.NewBuilder(worldId, channelId, id).Build()
			count, _ := p.GetPlayerCountInField(f)
			mu.Lock()
			counts[id] = count
			mu.Unlock()
		}(mapId)
	}
	wg.Wait()

	p.l.Debugf("Retrieved player counts for %d maps", len(counts))
	return counts, nil
}
