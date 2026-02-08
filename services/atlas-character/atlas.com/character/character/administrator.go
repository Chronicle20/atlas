package character

import (
	"strconv"
	"strings"

	"github.com/Chronicle20/atlas-constants/job"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EntityUpdateFunction func() ([]string, func(e *entity))

func create(db *gorm.DB, tenantId uuid.UUID, accountId uint32, worldId world.Id, name string, level byte, strength uint16, dexterity uint16, intelligence uint16, luck uint16, maxHP uint16, maxMP uint16, jobId job.Id, gender byte, hair uint32, face uint32, skinColor byte, mapId _map.Id, gm int) (Model, error) {
	e := &entity{
		TenantId:     tenantId,
		AccountId:    accountId,
		World:        worldId,
		Name:         name,
		Level:        level,
		Strength:     strength,
		Dexterity:    dexterity,
		Intelligence: intelligence,
		Luck:         luck,
		Hp:           maxHP,
		Mp:           maxMP,
		MaxHp:        maxHP,
		MaxMp:        maxMP,
		JobId:        jobId,
		SkinColor:    skinColor,
		Gender:       gender,
		Hair:         hair,
		Face:         face,
		MapId:        mapId,
		SP:           "0, 0, 0, 0, 0, 0, 0, 0, 0, 0",
		GM:           gm,
	}

	err := db.Create(e).Error
	if err != nil {
		return Model{}, err
	}
	return modelFromEntity(*e)
}

func delete(db *gorm.DB, tenantId uuid.UUID, characterId uint32) error {
	return db.Where(&entity{TenantId: tenantId, ID: characterId}).Delete(&entity{}).Error
}

// Returns a function which accepts a character model,and updates the persisted state of the character given a set of
// modifying functions.
func dynamicUpdate(db *gorm.DB) func(modifiers ...EntityUpdateFunction) func(tenantId uuid.UUID) model.Operator[Model] {
	return func(modifiers ...EntityUpdateFunction) func(tenantId uuid.UUID) model.Operator[Model] {
		return func(tenantId uuid.UUID) model.Operator[Model] {
			return func(c Model) error {
				if len(modifiers) > 0 {
					err := update(db, tenantId, c.Id(), modifiers...)
					if err != nil {
						return err
					}
				}
				return nil
			}
		}
	}
}

func update(db *gorm.DB, tenantId uuid.UUID, characterId uint32, modifiers ...EntityUpdateFunction) error {
	// Build a map of column->value updates instead of using a struct
	// This avoids GORM including zero values from unset fields
	updates := make(map[string]interface{})

	for _, modifier := range modifiers {
		columns, updateFunc := modifier()

		// Create a temporary entity to capture the update
		tempEntity := &entity{}
		updateFunc(tempEntity)

		// Extract the specific field values that were set
		for _, column := range columns {
			switch column {
			case "MapId":
				updates[column] = tempEntity.MapId
			case "Level":
				updates[column] = tempEntity.Level
			case "Experience":
				updates[column] = tempEntity.Experience
			case "GachaponExperience":
				updates[column] = tempEntity.GachaponExperience
			case "Strength":
				updates[column] = tempEntity.Strength
			case "Dexterity":
				updates[column] = tempEntity.Dexterity
			case "Intelligence":
				updates[column] = tempEntity.Intelligence
			case "Luck":
				updates[column] = tempEntity.Luck
			case "Hp":
				updates[column] = tempEntity.Hp
			case "Mp":
				updates[column] = tempEntity.Mp
			case "MaxHp":
				updates[column] = tempEntity.MaxHp
			case "MaxMp":
				updates[column] = tempEntity.MaxMp
			case "Meso":
				updates[column] = tempEntity.Meso
			case "HpMpUsed":
				updates[column] = tempEntity.HpMpUsed
			case "JobId":
				updates[column] = tempEntity.JobId
			case "SkinColor":
				updates[column] = tempEntity.SkinColor
			case "Gender":
				updates[column] = tempEntity.Gender
			case "Fame":
				updates[column] = tempEntity.Fame
			case "Hair":
				updates[column] = tempEntity.Hair
			case "Face":
				updates[column] = tempEntity.Face
			case "AP":
				updates[column] = tempEntity.AP
			case "SP":
				updates[column] = tempEntity.SP
			case "SpawnPoint":
				updates[column] = tempEntity.SpawnPoint
			case "GM":
				updates[column] = tempEntity.GM
			case "Name":
				updates[column] = tempEntity.Name
			case "X":
				updates[column] = tempEntity.X
			case "Y":
				updates[column] = tempEntity.Y
			case "Stance":
				updates[column] = tempEntity.Stance
			}
		}
	}

	if len(updates) == 0 {
		return nil
	}

	return db.Model(&entity{TenantId: tenantId, ID: characterId}).Updates(updates).Error
}

func SetLevel(level byte) EntityUpdateFunction {
	return func() ([]string, func(e *entity)) {
		return []string{"Level"}, func(e *entity) {
			e.Level = level
		}
	}
}

func SetMeso(amount uint32) EntityUpdateFunction {
	return func() ([]string, func(e *entity)) {
		return []string{"Meso"}, func(e *entity) {
			e.Meso = amount
		}
	}
}

func SetHealth(amount uint16) EntityUpdateFunction {
	return func() ([]string, func(e *entity)) {
		return []string{"Hp"}, func(e *entity) {
			e.Hp = amount
		}
	}
}

func SetMana(amount uint16) EntityUpdateFunction {
	return func() ([]string, func(e *entity)) {
		return []string{"Mp"}, func(e *entity) {
			e.Mp = amount
		}
	}
}

func SetAP(amount uint16) EntityUpdateFunction {
	return func() ([]string, func(e *entity)) {
		return []string{"AP"}, func(e *entity) {
			e.AP = amount
		}
	}
}

func SetStrength(amount uint16) EntityUpdateFunction {
	return func() ([]string, func(e *entity)) {
		return []string{"Strength"}, func(e *entity) {
			e.Strength = amount
		}
	}
}

func SetDexterity(amount uint16) EntityUpdateFunction {
	return func() ([]string, func(e *entity)) {
		return []string{"Dexterity"}, func(e *entity) {
			e.Dexterity = amount
		}
	}
}

func SetIntelligence(amount uint16) EntityUpdateFunction {
	return func() ([]string, func(e *entity)) {
		return []string{"Intelligence"}, func(e *entity) {
			e.Intelligence = amount
		}
	}
}

func SetLuck(amount uint16) EntityUpdateFunction {
	return func() ([]string, func(e *entity)) {
		return []string{"Luck"}, func(e *entity) {
			e.Luck = amount
		}
	}
}

func SpendOnStrength(strength uint16, ap uint16) []EntityUpdateFunction {
	return []EntityUpdateFunction{SetStrength(strength), SetAP(ap)}
}

func SpendOnDexterity(dexterity uint16, ap uint16) []EntityUpdateFunction {
	return []EntityUpdateFunction{SetDexterity(dexterity), SetAP(ap)}
}

func SpendOnIntelligence(intelligence uint16, ap uint16) []EntityUpdateFunction {
	return []EntityUpdateFunction{SetIntelligence(intelligence), SetAP(ap)}
}

func SpendOnLuck(luck uint16, ap uint16) []EntityUpdateFunction {
	return []EntityUpdateFunction{SetLuck(luck), SetAP(ap)}
}

func SetMaxHp(hp uint16) EntityUpdateFunction {
	return func() ([]string, func(e *entity)) {
		return []string{"MaxHp"}, func(e *entity) {
			e.MaxHp = hp
		}
	}
}

func SetMaxMp(mp uint16) EntityUpdateFunction {
	return func() ([]string, func(e *entity)) {
		return []string{"MaxMp"}, func(e *entity) {
			e.MaxMp = mp
		}
	}
}

func SetHpMpUsed(value int) EntityUpdateFunction {
	return func() ([]string, func(e *entity)) {
		return []string{"HpMpUsed"}, func(e *entity) {
			e.HpMpUsed = value
		}
	}
}

func SetMapId(mapId _map.Id) EntityUpdateFunction {
	return func() ([]string, func(e *entity)) {
		return []string{"MapId"}, func(e *entity) {
			e.MapId = mapId
		}
	}
}

func SetExperience(experience uint32) EntityUpdateFunction {
	return func() ([]string, func(e *entity)) {
		return []string{"Experience"}, func(e *entity) {
			e.Experience = experience
		}
	}
}

func UpdateSpawnPoint(spawnPoint uint32) EntityUpdateFunction {
	return func() ([]string, func(e *entity)) {
		return []string{"SpawnPoint"}, func(e *entity) {
			e.SpawnPoint = spawnPoint
		}
	}
}

func SetSP(amount uint32, bookId uint32) EntityUpdateFunction {
	return func() ([]string, func(e *entity)) {
		return []string{"SP"}, func(e *entity) {
			sps := strings.Split(e.SP, ",")
			sps[bookId] = strconv.Itoa(int(amount))
			e.SP = strings.Join(sps, ",")
		}
	}
}

func SetJob(jobId job.Id) EntityUpdateFunction {
	return func() ([]string, func(e *entity)) {
		return []string{"JobId"}, func(e *entity) {
			e.JobId = jobId
		}
	}
}

func SetFame(amount int16) EntityUpdateFunction {
	return func() ([]string, func(e *entity)) {
		return []string{"Fame"}, func(e *entity) {
			e.Fame = amount
		}
	}
}

func SetName(name string) EntityUpdateFunction {
	return func() ([]string, func(e *entity)) {
		return []string{"Name"}, func(e *entity) {
			e.Name = name
		}
	}
}

func SetHair(hair uint32) EntityUpdateFunction {
	return func() ([]string, func(e *entity)) {
		return []string{"Hair"}, func(e *entity) {
			e.Hair = hair
		}
	}
}

func SetFace(face uint32) EntityUpdateFunction {
	return func() ([]string, func(e *entity)) {
		return []string{"Face"}, func(e *entity) {
			e.Face = face
		}
	}
}

func SetGender(gender byte) EntityUpdateFunction {
	return func() ([]string, func(e *entity)) {
		return []string{"Gender"}, func(e *entity) {
			e.Gender = gender
		}
	}
}

func SetSkinColor(skinColor byte) EntityUpdateFunction {
	return func() ([]string, func(e *entity)) {
		return []string{"SkinColor"}, func(e *entity) {
			e.SkinColor = skinColor
		}
	}
}

func SetGm(gm int) EntityUpdateFunction {
	return func() ([]string, func(e *entity)) {
		return []string{"GM"}, func(e *entity) {
			e.GM = gm
		}
	}
}
