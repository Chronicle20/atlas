package seed

import (
	"atlas-gachapons/gachapon"
	"atlas-gachapons/global"
	"atlas-gachapons/item"
	"encoding/json"
	"fmt"
	"os"
)

type SeedResult struct {
	DeletedCount int      `json:"deletedCount"`
	CreatedCount int      `json:"createdCount"`
	FailedCount  int      `json:"failedCount"`
	Errors       []string `json:"errors,omitempty"`
}

type CombinedSeedResult struct {
	Gachapons   SeedResult `json:"gachapons"`
	Items       SeedResult `json:"items"`
	GlobalItems SeedResult `json:"globalItems"`
}

const defaultGachaponsPath = "/gachapons/data/gachapons.json"
const defaultItemsPath = "/gachapons/data/gachapon_items.json"
const defaultGlobalItemsPath = "/gachapons/data/global_gachapon_items.json"

func getGachaponsPath() string {
	if path := os.Getenv("GACHAPONS_DATA_PATH"); path != "" {
		return path
	}
	return defaultGachaponsPath
}

func getItemsPath() string {
	if path := os.Getenv("GACHAPON_ITEMS_DATA_PATH"); path != "" {
		return path
	}
	return defaultItemsPath
}

func getGlobalItemsPath() string {
	if path := os.Getenv("GLOBAL_ITEMS_DATA_PATH"); path != "" {
		return path
	}
	return defaultGlobalItemsPath
}

func LoadGachapons() ([]gachapon.JSONModel, error) {
	data, err := os.ReadFile(getGachaponsPath())
	if err != nil {
		return nil, fmt.Errorf("failed to read gachapons file: %w", err)
	}
	var models []gachapon.JSONModel
	if err := json.Unmarshal(data, &models); err != nil {
		return nil, fmt.Errorf("failed to parse gachapons JSON: %w", err)
	}
	return models, nil
}

func LoadItems() ([]item.JSONModel, error) {
	data, err := os.ReadFile(getItemsPath())
	if err != nil {
		return nil, fmt.Errorf("failed to read gachapon items file: %w", err)
	}
	var models []item.JSONModel
	if err := json.Unmarshal(data, &models); err != nil {
		return nil, fmt.Errorf("failed to parse gachapon items JSON: %w", err)
	}
	return models, nil
}

func LoadGlobalItems() ([]global.JSONModel, error) {
	data, err := os.ReadFile(getGlobalItemsPath())
	if err != nil {
		return nil, fmt.Errorf("failed to read global items file: %w", err)
	}
	var models []global.JSONModel
	if err := json.Unmarshal(data, &models); err != nil {
		return nil, fmt.Errorf("failed to parse global items JSON: %w", err)
	}
	return models, nil
}
