package _map

import (
	"context"
	"github.com/sirupsen/logrus"
	"sync"
)

// Processor provides operations for querying map player counts
type Processor interface {
	GetPlayerCountInMap(worldId byte, channelId byte, mapId uint32) (int, error)
	GetPlayerCountsInMaps(worldId byte, channelId byte, mapIds []uint32) (map[uint32]int, error)
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

// GetPlayerCountInMap retrieves the player count for a single map
// Returns 0 on error to allow graceful degradation
func (p *ProcessorImpl) GetPlayerCountInMap(worldId byte, channelId byte, mapId uint32) (int, error) {
	resp, err := requestCharactersInMap(worldId, channelId, mapId)(p.l, p.ctx)
	if err != nil {
		p.l.WithError(err).Warnf("Failed to get characters in map [%d], using count 0", mapId)
		return 0, nil
	}
	count := len(resp)
	p.l.Debugf("Map [%d] has %d players", mapId, count)
	return count, nil
}

// GetPlayerCountsInMaps retrieves player counts for multiple maps in parallel
// Returns a map of mapId -> playerCount
// Uses graceful degradation - returns 0 for maps that fail to query
func (p *ProcessorImpl) GetPlayerCountsInMaps(worldId byte, channelId byte, mapIds []uint32) (map[uint32]int, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	counts := make(map[uint32]int)

	for _, mapId := range mapIds {
		wg.Add(1)
		go func(id uint32) {
			defer wg.Done()
			count, _ := p.GetPlayerCountInMap(worldId, channelId, id)
			mu.Lock()
			counts[id] = count
			mu.Unlock()
		}(mapId)
	}
	wg.Wait()

	p.l.Debugf("Retrieved player counts for %d maps", len(counts))
	return counts, nil
}
