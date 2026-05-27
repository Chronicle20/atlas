package seeder

import "time"

type SubdomainCounts struct {
	Deleted int64    `json:"deleted"`
	Created int64    `json:"created"`
	Failed  int64    `json:"failed"`
	Errors  []string `json:"errors,omitempty"`
}

type Result struct {
	GroupName       string                     `json:"groupName"`
	CatalogRevision string                     `json:"catalogRevision"`
	Subdomains      map[string]SubdomainCounts `json:"subdomains"`
	StartedAt       time.Time                  `json:"startedAt"`
	CompletedAt     time.Time                  `json:"completedAt"`
}

type SubdomainStatus struct {
	Count     int64      `json:"count"`
	UpdatedAt *time.Time `json:"updatedAt"`
}

type Status struct {
	GroupName            string                     `json:"groupName"`
	Subdomains           map[string]SubdomainStatus `json:"subdomains"`
	UpdatedAt            *time.Time                 `json:"updatedAt"`
	CatalogRevision      string                     `json:"catalogRevision"`
	TenantSeededRevision *string                    `json:"tenantSeededRevision"`
	TenantSeededAt       *time.Time                 `json:"tenantSeededAt"`
}

const MaxErrors = 100
