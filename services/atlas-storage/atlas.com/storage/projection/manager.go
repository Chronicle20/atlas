package projection

import (
	"context"
	"strconv"

	atlas "github.com/Chronicle20/atlas-redis"
	"github.com/Chronicle20/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

// ManagerInterface defines the interface for projection management
type ManagerInterface interface {
	Get(ctx context.Context, characterId uint32) (Model, bool)
	Create(ctx context.Context, characterId uint32, projection Model)
	Delete(ctx context.Context, characterId uint32)
	Update(ctx context.Context, characterId uint32, updateFn func(Model) Model) bool
}

// Manager is the Redis-backed projection manager
type Manager struct {
	projections *atlas.TenantRegistry[uint32, Model]
}

var manager ManagerInterface

// GetManager returns the singleton instance of the projection manager
func GetManager() ManagerInterface {
	return manager
}

// InitManager initializes the projection manager with a Redis client
func InitManager(client *goredis.Client) {
	manager = &Manager{
		projections: atlas.NewTenantRegistry[uint32, Model](client, "storage-projection", func(k uint32) string {
			return strconv.FormatUint(uint64(k), 10)
		}),
	}
}

// Get retrieves a projection for a character
func (m *Manager) Get(ctx context.Context, characterId uint32) (Model, bool) {
	t := tenant.MustFromContext(ctx)
	proj, err := m.projections.Get(ctx, t, characterId)
	if err != nil {
		return Model{}, false
	}
	return proj, true
}

// Create stores a projection for a character
func (m *Manager) Create(ctx context.Context, characterId uint32, projection Model) {
	t := tenant.MustFromContext(ctx)
	_ = m.projections.Put(ctx, t, characterId, projection)
}

// Delete removes a projection for a character
func (m *Manager) Delete(ctx context.Context, characterId uint32) {
	t := tenant.MustFromContext(ctx)
	_ = m.projections.Remove(ctx, t, characterId)
}

// Update atomically updates a projection using the provided function.
// Returns true if the projection existed and was updated, false otherwise.
func (m *Manager) Update(ctx context.Context, characterId uint32, updateFn func(Model) Model) bool {
	t := tenant.MustFromContext(ctx)
	proj, err := m.projections.Get(ctx, t, characterId)
	if err != nil {
		return false
	}
	updated := updateFn(proj)
	_ = m.projections.Put(ctx, t, characterId, updated)
	return true
}
