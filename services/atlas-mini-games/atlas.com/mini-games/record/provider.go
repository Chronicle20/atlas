package record

import (
	"context"
	"errors"

	"gorm.io/gorm"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// GetOrZero returns the game_records row for (characterId, gameType) in the
// context's tenant. When no row exists yet it returns a zero-valued Model
// (CharacterId/GameType populated, Wins/Ties/Losses all 0) and a nil error
// rather than gorm.ErrRecordNotFound — callers should not have to
// special-case "never played this game type yet".
//
// Tenancy is context-driven (DOM-11): db must be scoped with WithContext(ctx)
// so the tenant callbacks add the tenant_id filter, and ctx supplies the
// tenant id stamped onto the absent-row zero Model.
func GetOrZero(ctx context.Context, db *gorm.DB, characterId uint32, gameType GameType) (Model, error) {
	var e Entity
	err := db.Where("character_id = ? AND game_type = ?", characterId, string(gameType)).First(&e).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Zero-filled absent-row Model (Id stays uuid.Nil = "never played").
			ten := tenant.MustFromContext(ctx)
			return NewBuilder(ten.Id(), characterId, gameType).Build()
		}
		return Model{}, err
	}
	return Make(e)
}

// GetByCharacter returns one Model per GameType in AllGameTypes for the
// given character, zero-filled for any game type with no rows yet.
func GetByCharacter(ctx context.Context, db *gorm.DB, characterId uint32) ([]Model, error) {
	results := make([]Model, 0, len(AllGameTypes))
	for _, gameType := range AllGameTypes {
		m, err := GetOrZero(ctx, db, characterId, gameType)
		if err != nil {
			return nil, err
		}
		results = append(results, m)
	}
	return results, nil
}
