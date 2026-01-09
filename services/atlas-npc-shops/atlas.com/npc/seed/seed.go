package seed

// SeedResult represents the result of a seed operation
type SeedResult struct {
	DeletedShops       int      `json:"deletedShops"`
	DeletedCommodities int      `json:"deletedCommodities"`
	CreatedShops       int      `json:"createdShops"`
	CreatedCommodities int      `json:"createdCommodities"`
	FailedCount        int      `json:"failedCount"`
	Errors             []string `json:"errors,omitempty"`
}
