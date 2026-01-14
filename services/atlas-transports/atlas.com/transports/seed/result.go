package seed

// SeedResult represents the result of a seed operation
type SeedResult struct {
	DeletedRoutes int      `json:"deletedRoutes"`
	CreatedRoutes int      `json:"createdRoutes"`
	FailedCount   int      `json:"failedCount"`
	Errors        []string `json:"errors,omitempty"`
}
