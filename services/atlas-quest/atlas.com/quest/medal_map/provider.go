package medal_map

import (
	"gorm.io/gorm"
)

// countByCharacterAndQuest returns the distinct-map count for a character's
// quest -- the value Cosmic writes as quest progress under the quest's
// infoNumber (MapScriptMethods.java:130).
func countByCharacterAndQuest(db *gorm.DB, characterId uint32, questId uint32) (uint32, error) {
	var count int64
	err := db.Model(&entity{}).Where("character_id = ? AND quest_id = ?", characterId, questId).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return uint32(count), nil
}
