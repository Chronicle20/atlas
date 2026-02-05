package projection

import (
	"atlas-storage/asset"
	"atlas-storage/storage"
	"context"

	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// BuildProjection creates a new projection from storage data.
// All compartment slices are initialized with ALL assets from storage.
func BuildProjection(l logrus.FieldLogger, db *gorm.DB, tenantId uuid.UUID, ctx context.Context) func(characterId uint32, accountId uint32, worldId world.Id, npcId uint32) (Model, error) {
	return func(characterId uint32, accountId uint32, worldId world.Id, npcId uint32) (Model, error) {
		// Get storage with decorated assets
		s, err := storage.GetByWorldAndAccountId(l, db, tenantId, ctx)(worldId, accountId)
		if err != nil {
			return Model{}, err
		}

		// Initialize all compartments with ALL assets
		compartments := make(map[asset.InventoryType][]asset.Model[any])
		for _, invType := range AllCompartmentTypes() {
			// Each compartment starts with a copy of ALL assets
			assets := make([]asset.Model[any], len(s.Assets()))
			copy(assets, s.Assets())
			compartments[invType] = assets
		}

		return NewBuilder().
			SetCharacterId(characterId).
			SetAccountId(accountId).
			SetWorldId(worldId).
			SetStorageId(s.Id()).
			SetCapacity(s.Capacity()).
			SetMesos(s.Mesos()).
			SetNpcId(npcId).
			SetCompartments(compartments).
			MustBuild(), nil
	}
}
