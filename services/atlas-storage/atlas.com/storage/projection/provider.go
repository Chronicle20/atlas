package projection

import (
	"atlas-storage/asset"
	"atlas-storage/storage"

	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// BuildProjection creates a new projection from storage data.
// Assets are grouped by their inventory type into compartments.
func BuildProjection(l logrus.FieldLogger, db *gorm.DB, tenantId uuid.UUID) func(characterId uint32, accountId uint32, worldId world.Id, npcId uint32) (Model, error) {
	return func(characterId uint32, accountId uint32, worldId world.Id, npcId uint32) (Model, error) {
		s, err := storage.GetByWorldAndAccountId(l, db, tenantId)(worldId, accountId)
		if err != nil {
			return Model{}, err
		}

		// Group assets by inventory type
		compartments := make(map[inventory.Type][]asset.Model)
		for _, a := range s.Assets() {
			t := a.InventoryType()
			compartments[t] = append(compartments[t], a)
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
