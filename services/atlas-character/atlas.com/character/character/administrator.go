package character

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"strconv"
	"strings"
)

type EntityUpdateFunction func() ([]string, func(e *entity))

func create(db *gorm.DB, tenantId uuid.UUID, accountId uint32, worldId byte, name string, level byte, strength uint16, dexterity uint16, intelligence uint16, luck uint16, maxHP uint16, maxMP uint16, jobId uint16, gender byte, hair uint32, face uint32, skinColor byte, mapId uint32) (Model, error) {
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
		HP:           maxHP,
		MP:           maxMP,
		MaxHP:        maxHP,
		MaxMP:        maxMP,
		JobId:        jobId,
		SkinColor:    skinColor,
		Gender:       gender,
		Hair:         hair,
		Face:         face,
		MapId:        mapId,
		SP:           "0, 0, 0, 0, 0, 0, 0, 0, 0, 0",
	}

	err := db.Create(e).Error
	if err != nil {
		return Model{}, err
	}
	return makeCharacter(*e)
}

func delete(db *gorm.DB, tenantId uuid.UUID, characterId uint32) error {
	return db.Where(&entity{TenantId: tenantId, ID: characterId}).Delete(&entity{}).Error
}

func update(db *gorm.DB, tenantId uuid.UUID, characterId uint32, modifiers ...EntityUpdateFunction) error {
	e := &entity{}

	var columns []string
	for _, modifier := range modifiers {
		c, u := modifier()
		columns = append(columns, c...)
		u(e)
	}
	return db.Model(&entity{TenantId: tenantId, ID: characterId}).Select(columns).Updates(e).Error
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
		return []string{"HP"}, func(e *entity) {
			e.HP = amount
		}
	}
}

func SetMana(amount uint16) EntityUpdateFunction {
	return func() ([]string, func(e *entity)) {
		return []string{"MP"}, func(e *entity) {
			e.MP = amount
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

func SetMaxHP(hp uint16) EntityUpdateFunction {
	return func() ([]string, func(e *entity)) {
		return []string{"MaxHP"}, func(e *entity) {
			e.MaxHP = hp
		}
	}
}

func SetMaxMP(mp uint16) EntityUpdateFunction {
	return func() ([]string, func(e *entity)) {
		return []string{"MaxMP"}, func(e *entity) {
			e.MaxMP = mp
		}
	}
}

func SetMapId(mapId uint32) EntityUpdateFunction {
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

func SetJob(jobId uint16) EntityUpdateFunction {
	return func() ([]string, func(e *entity)) {
		return []string{"JobId"}, func(e *entity) {
			e.JobId = jobId
		}
	}
}
