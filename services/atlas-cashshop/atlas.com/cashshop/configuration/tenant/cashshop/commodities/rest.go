package commodities

type RestModel struct {
	HourlyExpirations []HourlyExpiration `json:"hourlyExpirations,omitempty"`
}

type HourlyExpiration struct {
	TemplateId uint32 `json:"templateId"`
	Hours      uint32 `json:"hours"`
}
