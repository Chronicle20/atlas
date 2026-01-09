package seed

// SeedResult represents the result of a seed operation
type SeedResult struct {
	DeletedCount int      `json:"deletedCount"`
	CreatedCount int      `json:"createdCount"`
	FailedCount  int      `json:"failedCount"`
	Errors       []string `json:"errors,omitempty"`
}

// CombinedSeedResult represents the combined results of seeding monster and continent drops
type CombinedSeedResult struct {
	MonsterDrops   SeedResult `json:"monsterDrops"`
	ContinentDrops SeedResult `json:"continentDrops"`
}
