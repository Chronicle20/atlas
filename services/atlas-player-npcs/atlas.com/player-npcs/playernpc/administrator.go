package playernpc

import (
	"database/sql"
	"encoding/binary"
	"hash/fnv"

	"github.com/google/uuid"
	"gorm.io/gorm"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// createPlayerNpc inserts the root row and its equipment rows in a single
// transaction (design §8: the whole deploy is one atomic write). A unique
// constraint violation on script id, (map, name), or (map, object id) --
// PRD §6 -- aborts the transaction and is returned to the caller as-is.
func createPlayerNpc(db *gorm.DB, tenantId uuid.UUID, m Model) (Model, error) {
	entity := MakeEntity(tenantId, m)
	entity.Id = uuid.Nil

	err := database.ExecuteTransaction(db, func(tx *gorm.DB) error {
		if err := tx.Create(&entity).Error; err != nil {
			return err
		}

		equipment := MakeEquipmentEntities(tenantId, entity.Id, m)
		for i := range equipment {
			equipment[i].Id = uuid.Nil
			if err := tx.Create(&equipment[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return Model{}, err
	}

	return getPlayerNpcModel(db, entity.Id)
}

// deletePlayerNpc removes the root row. The `player_npc_equipment` foreign
// key is declared ON DELETE CASCADE (entity.go), so its rows are removed by
// the database, not by this function.
func deletePlayerNpc(db *gorm.DB, id uuid.UUID) error {
	return database.ExecuteTransaction(db, func(tx *gorm.DB) error {
		return tx.Where("id = ?", id).Delete(&Entity{}).Error
	})
}

// getPlayerNpcModel hydrates a full Model (root + equipment) for id.
func getPlayerNpcModel(db *gorm.DB, id uuid.UUID) (Model, error) {
	entity, err := getByIdEntityProvider(id)(db)()
	if err != nil {
		return Model{}, err
	}
	equipmentEntities, err := getEquipmentByPlayerNpcIdProvider(id)(db)()
	if err != nil {
		return Model{}, err
	}
	return Make(entity, equipmentEntities)
}

// playerNpcsByMap hydrates every deployed Player NPC for (worldId, mapId)
// -- the map-enter read path (design §6).
func playerNpcsByMap(db *gorm.DB, worldId byte, mapId uint32, page model.Page) ([]Model, error) {
	paged, err := getByMapPagedProvider(worldId, mapId, page)(db)()
	if err != nil {
		return nil, err
	}

	models := make([]Model, 0, len(paged.Items))
	for _, e := range paged.Items {
		m, err := getPlayerNpcModel(db, e.Id)
		if err != nil {
			return nil, err
		}
		models = append(models, m)
	}
	return models, nil
}

// advisoryLockKey derives a stable int64 advisory-lock key from
// (tenant, world, map) (design §8.1/D-6). fnv-1a over the tenant id bytes
// plus the world/map ids gives a key that is stable for the lifetime of
// the process without needing a registry -- collisions only cost an
// unrelated map briefly sharing the lock, never correctness, since the
// lock only serializes, it never gates on identity.
func advisoryLockKey(tenantId uuid.UUID, worldId byte, mapId uint32) int64 {
	h := fnv.New64a()
	_, _ = h.Write(tenantId[:])
	_, _ = h.Write([]byte{worldId})
	var mb [4]byte
	binary.BigEndian.PutUint32(mb[:], mapId)
	_, _ = h.Write(mb[:])
	return int64(h.Sum64())
}

// advisoryLock takes a transaction-scoped Postgres advisory lock keyed by
// (tenant, world, map) (design §8.1/D-6). It releases on commit or
// rollback with no cleanup path, so there is no unlock to call. On a
// non-Postgres dialect (the sqlite unit-test database) locking is a no-op,
// matching services/atlas-data/atlas.com/data/baseline/restore.go's
// dialect-aware convention -- a single-threaded test process needs no real
// mutual exclusion.
func advisoryLock(tx *gorm.DB, tenantId uuid.UUID, worldId byte, mapId uint32) error {
	if tx.Name() != "postgres" {
		return nil
	}
	return tx.Exec("SELECT pg_advisory_xact_lock(?)", advisoryLockKey(tenantId, worldId, mapId)).Error
}

// countByName counts the Player NPCs already deployed for (world, map,
// name) -- the duplicate check (PRD §6's unique index, eligibility's
// existingCount input).
func countByName(tx *gorm.DB, worldId byte, mapId uint32, name string) (int, error) {
	var count int64
	err := tx.Model(&Entity{}).Where("world_id = ? AND map_id = ? AND name = ?", worldId, mapId, name).Count(&count).Error
	return int(count), err
}

// inUseScriptIds reads every script id already allocated in (world) --
// allocation.Allocate's `inUse` input (design §8.1).
func inUseScriptIds(tx *gorm.DB, worldId byte) (map[uint32]bool, error) {
	var ids []uint32
	if err := tx.Model(&Entity{}).Where("world_id = ?", worldId).Pluck("script_id", &ids).Error; err != nil {
		return nil, err
	}
	inUse := make(map[uint32]bool, len(ids))
	for _, id := range ids {
		inUse[id] = true
	}
	return inUse, nil
}

// mapEntities reads every Player NPC on (world, map), ordered ascending by
// script id -- deployment history order (design §5.4 step 1).
func mapEntities(tx *gorm.DB, worldId byte, mapId uint32) ([]Entity, error) {
	var entities []Entity
	err := tx.Where("world_id = ? AND map_id = ?", worldId, mapId).Order("script_id ASC").Find(&entities).Error
	return entities, err
}

// entitiesByCharacter reads every Player NPC belonging to characterId,
// optionally scoped to mapId (nil matches every map) -- the REMOVE command's
// (design §9.2, FR-8.2) and the REST bulk-delete filter's (PRD §5) shared
// lookup.
func entitiesByCharacter(db *gorm.DB, characterId uint32, mapId *_map.Id) ([]Entity, error) {
	q := db.Where("character_id = ?", characterId)
	if mapId != nil {
		q = q.Where("map_id = ?", uint32(*mapId))
	}
	var entities []Entity
	err := q.Find(&entities).Error
	return entities, err
}

// currentStepForMap returns the positioner step every Player NPC on
// (world, map) currently shares (design §5.4 rewrites the whole map's step
// atomically, so any one row's value is the map's value), or 0 when the
// map has no rows yet.
func currentStepForMap(tx *gorm.DB, worldId byte, mapId uint32) (byte, error) {
	var steps []byte
	err := tx.Model(&Entity{}).Where("world_id = ? AND map_id = ?", worldId, mapId).Limit(1).Pluck("step", &steps).Error
	if err != nil {
		return 0, err
	}
	if len(steps) == 0 {
		return 0, nil
	}
	return steps[0], nil
}

// nextWorldJobRank returns MAX(world_job_rank)+1 over (world, job
// category) -- the deployment ordinal counter (design §6.3), read under
// the same lock as allocation so two simultaneous deploys cannot collide
// on it.
func nextWorldJobRank(tx *gorm.DB, worldId byte, jobCategory uint16) (uint32, error) {
	return nextRank(tx, "world_job_rank", "world_id = ? AND job_id = ?", worldId, jobCategory)
}

// nextOverallJobRank returns MAX(overall_job_rank)+1 over (job category)
// -- the tenant-wide deployment ordinal counter (design §6.3), independent
// of world.
func nextOverallJobRank(tx *gorm.DB, jobCategory uint16) (uint32, error) {
	return nextRank(tx, "overall_job_rank", "job_id = ?", jobCategory)
}

func nextRank(tx *gorm.DB, column string, where string, args ...interface{}) (uint32, error) {
	row := tx.Model(&Entity{}).Where(where, args...).Select("MAX(" + column + ")").Row()
	var max sql.NullInt64
	if err := row.Scan(&max); err != nil {
		return 0, err
	}
	if !max.Valid {
		return 1, nil
	}
	return uint32(max.Int64) + 1, nil
}

// updatePosition persists a reorganized or freshly resolved position/step
// for an existing row (design §5.4 step 3), leaving every other column
// untouched.
func updatePosition(tx *gorm.DB, id uuid.UUID, x, cy int16, fh uint16, rx0, rx1 int16, step byte) error {
	return tx.Model(&Entity{}).Where("id = ?", id).Updates(map[string]interface{}{
		"x":    x,
		"cy":   cy,
		"fh":   fh,
		"rx0":  rx0,
		"rx1":  rx1,
		"step": step,
	}).Error
}

// updateAppearanceAndRank persists a re-deploy's refreshed appearance and
// current-standing ranks (design §6.2), leaving script id, object id,
// position and the frozen deployment-ordinal ranks untouched.
func updateAppearanceAndRank(tx *gorm.DB, id uuid.UUID, gender byte, skin byte, face uint32, hair uint32, jobId uint16, worldRank uint32, overallRank uint32) error {
	return tx.Model(&Entity{}).Where("id = ?", id).Updates(map[string]interface{}{
		"gender":       gender,
		"skin":         skin,
		"face":         face,
		"hair":         hair,
		"job_id":       jobId,
		"world_rank":   worldRank,
		"overall_rank": overallRank,
	}).Error
}

// replaceEquipment deletes every equipment row for playerNpcId and inserts
// equipment in its place, inside tx -- used by re-deploy (design §6.2) to
// refresh frozen equipment without touching the root row's own update
// path.
func replaceEquipment(tx *gorm.DB, tenantId uuid.UUID, playerNpcId uuid.UUID, equipment []EquipmentModel) error {
	if err := tx.Where("player_npc_id = ?", playerNpcId).Delete(&EquipmentEntity{}).Error; err != nil {
		return err
	}
	entities := make([]EquipmentEntity, 0, len(equipment))
	for _, em := range equipment {
		entities = append(entities, EquipmentEntity{
			TenantId:    tenantId,
			PlayerNpcId: playerNpcId,
			Slot:        em.Slot(),
			ItemId:      em.ItemId(),
		})
	}
	for i := range entities {
		if err := tx.Create(&entities[i]).Error; err != nil {
			return err
		}
	}
	return nil
}
