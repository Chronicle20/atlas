package record

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GetOrZero returns the game_records row for (tenantId, characterId,
// gameType). When no row exists yet it returns a zero-valued Model
// (CharacterId/GameType populated, Wins/Ties/Losses all 0) and a nil error
// rather than gorm.ErrRecordNotFound — callers should not have to
// special-case "never played this game type yet".
func GetOrZero(db *gorm.DB, tenantId uuid.UUID, characterId uint32, gameType GameType) (Model, error) {
	var e Entity
	err := db.Where("tenant_id = ? AND character_id = ? AND game_type = ?", tenantId, characterId, string(gameType)).First(&e).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Zero-filled absent-row Model (Id stays uuid.Nil = "never played").
			return NewBuilder(tenantId, characterId, gameType).Build()
		}
		return Model{}, err
	}
	return Make(e)
}

// GetByCharacter returns one Model per GameType in AllGameTypes for the
// given character, zero-filled for any game type with no rows yet.
func GetByCharacter(db *gorm.DB, tenantId uuid.UUID, characterId uint32) ([]Model, error) {
	results := make([]Model, 0, len(AllGameTypes))
	for _, gameType := range AllGameTypes {
		m, err := GetOrZero(db, tenantId, characterId, gameType)
		if err != nil {
			return nil, err
		}
		results = append(results, m)
	}
	return results, nil
}
