package seed

// JSONModel represents the JSON structure for a transport route configuration
type JSONModel struct {
	Name                           string `json:"name"`
	StartMapId                     uint32 `json:"startMapId"`
	StagingMapId                   uint32 `json:"stagingMapId"`
	EnRouteMapIds                  []uint32 `json:"enRouteMapIds"`
	DestinationMapId               uint32 `json:"destinationMapId"`
	ObservationMapId               uint32 `json:"observationMapId"`
	BoardingWindowDurationMinutes  int    `json:"boardingWindowDurationMinutes"`
	PreDepartureDurationMinutes    int    `json:"preDepartureDurationMinutes"`
	TravelDurationMinutes          int    `json:"travelDurationMinutes"`
	CycleIntervalMinutes           int    `json:"cycleIntervalMinutes"`
}
