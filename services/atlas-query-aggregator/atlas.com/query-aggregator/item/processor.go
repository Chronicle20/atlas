package item

import (
	"context"
	"sync"
	"time"

	"github.com/Chronicle20/atlas-constants/inventory"
	atlasItem "github.com/Chronicle20/atlas-constants/item"
	"github.com/sirupsen/logrus"
)

// Processor provides operations for querying item data
type Processor interface {
	GetSlotMax(itemId uint32) uint32
}

// cacheEntry represents a cached item data entry with expiration
type cacheEntry struct {
	slotMax   uint32
	expiresAt time.Time
}

// ItemCacheInterface defines the interface for the item cache
type ItemCacheInterface interface {
	Get(itemId uint32) (uint32, bool)
	Put(itemId uint32, slotMax uint32, ttl time.Duration)
}

// ItemCache is a singleton cache for item slot max data
type ItemCache struct {
	mu   sync.RWMutex
	data map[uint32]cacheEntry
}

var itemCache ItemCacheInterface
var cacheOnce sync.Once

// GetItemCache returns the singleton instance of the item cache
func GetItemCache() ItemCacheInterface {
	cacheOnce.Do(func() {
		itemCache = &ItemCache{
			data: make(map[uint32]cacheEntry),
		}
	})
	return itemCache
}

// Get retrieves a slotMax from cache if not expired
func (c *ItemCache) Get(itemId uint32) (uint32, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.data[itemId]
	if !ok {
		return 0, false
	}

	// Check if expired
	if time.Now().After(entry.expiresAt) {
		return 0, false
	}

	return entry.slotMax, true
}

// Put stores a slotMax in cache with expiration
func (c *ItemCache) Put(itemId uint32, slotMax uint32, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[itemId] = cacheEntry{
		slotMax:   slotMax,
		expiresAt: time.Now().Add(ttl),
	}
}

// ProcessorImpl implements the Processor interface with caching
type ProcessorImpl struct {
	l     logrus.FieldLogger
	ctx   context.Context
	cache ItemCacheInterface
}

// NewProcessor creates a new item processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:     l,
		ctx:   ctx,
		cache: GetItemCache(),
	}
}

// GetSlotMax retrieves the maximum stack size for an item
// Returns cached value if available, otherwise fetches from service
// Returns default value on error to allow graceful degradation
func (p *ProcessorImpl) GetSlotMax(itemId uint32) uint32 {
	// Check cache first
	if slotMax, ok := p.cache.Get(itemId); ok {
		p.l.Debugf("Cache hit for item [%d], slotMax=%d", itemId, slotMax)
		return slotMax
	}

	// Determine item type to know which endpoint to call
	invType, ok := inventory.TypeFromItemId(atlasItem.Id(itemId))
	if !ok {
		p.l.Warnf("Invalid item ID [%d], using default slotMax", itemId)
		return GetDefaultSlotMax(itemId)
	}

	var slotMax uint32

	// Fetch from appropriate service based on item type
	switch invType {
	case inventory.TypeValueEquip:
		// Equipment never stacks
		slotMax = 1
		p.cache.Put(itemId, slotMax, time.Hour)
		return slotMax

	case inventory.TypeValueUse:
		// Consumable items
		resp, fetchErr := requestConsumable(itemId)(p.l, p.ctx)
		if fetchErr != nil {
			p.l.WithError(fetchErr).Warnf("Failed to fetch consumable [%d], using default", itemId)
			slotMax = GetDefaultSlotMax(itemId)
		} else {
			slotMax = resp.SlotMax
		}

	case inventory.TypeValueSetup:
		// Setup items
		resp, fetchErr := requestSetup(itemId)(p.l, p.ctx)
		if fetchErr != nil {
			p.l.WithError(fetchErr).Warnf("Failed to fetch setup [%d], using default", itemId)
			slotMax = GetDefaultSlotMax(itemId)
		} else {
			slotMax = resp.SlotMax
		}

	case inventory.TypeValueETC:
		// ETC items
		resp, fetchErr := requestEtc(itemId)(p.l, p.ctx)
		if fetchErr != nil {
			p.l.WithError(fetchErr).Warnf("Failed to fetch etc [%d], using default", itemId)
			slotMax = GetDefaultSlotMax(itemId)
		} else {
			slotMax = resp.SlotMax
		}

	case inventory.TypeValueCash:
		// Cash items - treat similar to consumables for now
		resp, fetchErr := requestConsumable(itemId)(p.l, p.ctx)
		if fetchErr != nil {
			p.l.WithError(fetchErr).Warnf("Failed to fetch cash item [%d], using default", itemId)
			slotMax = GetDefaultSlotMax(itemId)
		} else {
			slotMax = resp.SlotMax
		}

	default:
		p.l.Warnf("Unknown inventory type for item [%d], using default slotMax", itemId)
		slotMax = GetDefaultSlotMax(itemId)
	}

	// Handle invalid slotMax (0 or suspiciously high values)
	if slotMax == 0 {
		p.l.Warnf("Item [%d] has slotMax=0, using default", itemId)
		slotMax = GetDefaultSlotMax(itemId)
	} else if slotMax > 1000 {
		p.l.Warnf("Item [%d] has suspicious slotMax=%d, capping at 1000", itemId, slotMax)
		slotMax = 1000
	}

	// Cache the result
	p.cache.Put(itemId, slotMax, time.Hour)
	p.l.Debugf("Fetched slotMax=%d for item [%d]", slotMax, itemId)

	return slotMax
}
