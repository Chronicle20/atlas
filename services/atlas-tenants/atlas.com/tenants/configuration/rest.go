package configuration

import (
	"encoding/json"
)

// RouteRestModel is the JSON:API resource for routes
type RouteRestModel struct {
	Id                     string   `json:"-"`
	Name                   string   `json:"name"`
	StartMapId             uint32   `json:"startMapId"`
	StagingMapId           uint32   `json:"stagingMapId"`
	EnRouteMapIds          []uint32 `json:"enRouteMapIds"`
	DestinationMapId       uint32   `json:"destinationMapId"`
	ObservationMapId       uint32   `json:"observationMapId"`
	BoardingWindowDuration uint32   `json:"boardingWindowDuration"`
	PreDepartureDuration   uint32   `json:"preDepartureDuration"`
	TravelDuration         uint32   `json:"travelDuration"`
	CycleInterval          uint32   `json:"cycleInterval"`
}

// GetID returns the resource ID
func (r RouteRestModel) GetID() string {
	return r.Id
}

// SetID sets the resource ID
func (r *RouteRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// GetName returns the resource name
func (r RouteRestModel) GetName() string {
	return "routes"
}

// TransformRoute converts a map[string]interface{} to a RouteRestModel
func TransformRoute(data map[string]interface{}) (RouteRestModel, error) {
	id, _ := data["id"].(string)

	attributes, ok := data["attributes"].(map[string]interface{})
	if !ok {
		attributes = make(map[string]interface{})
	}

	name, _ := attributes["name"].(string)

	startMapId := uint32(0)
	if val, ok := attributes["startMapId"].(float64); ok {
		startMapId = uint32(val)
	}

	stagingMapId := uint32(0)
	if val, ok := attributes["stagingMapId"].(float64); ok {
		stagingMapId = uint32(val)
	}

	enRouteMapIds := make([]uint32, 0)
	if vals, ok := attributes["enRouteMapIds"].([]interface{}); ok {
		for _, v := range vals {
			if val, ok := v.(float64); ok {
				enRouteMapIds = append(enRouteMapIds, uint32(val))
			}
		}
	}

	destinationMapId := uint32(0)
	if val, ok := attributes["destinationMapId"].(float64); ok {
		destinationMapId = uint32(val)
	}

	observationMapId := uint32(0)
	if val, ok := attributes["observationMapId"].(float64); ok {
		observationMapId = uint32(val)
	}

	boardingWindowDuration := uint32(0)
	if val, ok := attributes["boardingWindowDuration"].(float64); ok {
		boardingWindowDuration = uint32(val)
	}

	preDepartureDuration := uint32(0)
	if val, ok := attributes["preDepartureDuration"].(float64); ok {
		preDepartureDuration = uint32(val)
	}

	travelDuration := uint32(0)
	if val, ok := attributes["travelDuration"].(float64); ok {
		travelDuration = uint32(val)
	}

	cycleInterval := uint32(0)
	if val, ok := attributes["cycleInterval"].(float64); ok {
		cycleInterval = uint32(val)
	}

	return RouteRestModel{
		Id:                     id,
		Name:                   name,
		StartMapId:             startMapId,
		StagingMapId:           stagingMapId,
		EnRouteMapIds:          enRouteMapIds,
		DestinationMapId:       destinationMapId,
		ObservationMapId:       observationMapId,
		BoardingWindowDuration: boardingWindowDuration,
		PreDepartureDuration:   preDepartureDuration,
		TravelDuration:         travelDuration,
		CycleInterval:          cycleInterval,
	}, nil
}

// ExtractRoute converts a RouteRestModel to a map[string]interface{}
func ExtractRoute(r RouteRestModel) (map[string]interface{}, error) {
	return map[string]interface{}{
		"type": "routes",
		"id":   r.Id,
		"attributes": map[string]interface{}{
			"name":                   r.Name,
			"startMapId":             r.StartMapId,
			"stagingMapId":           r.StagingMapId,
			"enRouteMapIds":          r.EnRouteMapIds,
			"destinationMapId":       r.DestinationMapId,
			"observationMapId":       r.ObservationMapId,
			"boardingWindowDuration": r.BoardingWindowDuration,
			"preDepartureDuration":   r.PreDepartureDuration,
			"travelDuration":         r.TravelDuration,
			"cycleInterval":          r.CycleInterval,
		},
	}, nil
}

// CreateRouteJsonData creates a JSON:API compliant data structure for routes
func CreateRouteJsonData(routes []map[string]interface{}) (json.RawMessage, error) {
	data := map[string]interface{}{
		"data": routes,
	}
	return json.Marshal(data)
}

// CreateSingleRouteJsonData creates a JSON:API compliant data structure for a single route
func CreateSingleRouteJsonData(route map[string]interface{}) (json.RawMessage, error) {
	data := map[string]interface{}{
		"data": route,
	}
	return json.Marshal(data)
}

// VesselRestModel is the JSON:API resource for vessels
type VesselRestModel struct {
	Id              string `json:"-"`
	Name            string `json:"name"`
	RouteAID        string `json:"routeAID"`
	RouteBID        string `json:"routeBID"`
	TurnaroundDelay uint32 `json:"turnaroundDelay"`
}

// GetID returns the resource ID
func (v VesselRestModel) GetID() string {
	return v.Id
}

// SetID sets the resource ID
func (v *VesselRestModel) SetID(id string) error {
	v.Id = id
	return nil
}

// GetName returns the resource name
func (v VesselRestModel) GetName() string {
	return "vessels"
}

// TransformVessel converts a map[string]interface{} to a VesselRestModel
func TransformVessel(data map[string]interface{}) (VesselRestModel, error) {
	id, _ := data["id"].(string)

	attributes, ok := data["attributes"].(map[string]interface{})
	if !ok {
		attributes = make(map[string]interface{})
	}

	name, _ := attributes["name"].(string)

	routeAID, _ := attributes["routeAID"].(string)

	routeBID, _ := attributes["routeBID"].(string)

	turnaroundDelay := uint32(0)
	if val, ok := attributes["turnaroundDelay"].(float64); ok {
		turnaroundDelay = uint32(val)
	}

	return VesselRestModel{
		Id:              id,
		Name:            name,
		RouteAID:        routeAID,
		RouteBID:        routeBID,
		TurnaroundDelay: turnaroundDelay,
	}, nil
}

// ExtractVessel converts a VesselRestModel to a map[string]interface{}
func ExtractVessel(v VesselRestModel) (map[string]interface{}, error) {
	return map[string]interface{}{
		"type": "vessels",
		"id":   v.Id,
		"attributes": map[string]interface{}{
			"name":            v.Name,
			"routeAID":        v.RouteAID,
			"routeBID":        v.RouteBID,
			"turnaroundDelay": v.TurnaroundDelay,
		},
	}, nil
}

// CreateVesselJsonData creates a JSON:API compliant data structure for vessels
func CreateVesselJsonData(vessels []map[string]interface{}) (json.RawMessage, error) {
	data := map[string]interface{}{
		"data": vessels,
	}
	return json.Marshal(data)
}

// CreateSingleVesselJsonData creates a JSON:API compliant data structure for a single vessel
func CreateSingleVesselJsonData(vessel map[string]interface{}) (json.RawMessage, error) {
	data := map[string]interface{}{
		"data": vessel,
	}
	return json.Marshal(data)
}

// InstanceRouteRestModel is the JSON:API resource for instance routes
type InstanceRouteRestModel struct {
	Id                    string `json:"-"`
	Name                  string `json:"name"`
	StartMapId            uint32 `json:"startMapId"`
	TransitMapId          uint32 `json:"transitMapId"`
	DestinationMapId      uint32 `json:"destinationMapId"`
	Capacity              uint32 `json:"capacity"`
	BoardingWindowSeconds uint32 `json:"boardingWindowSeconds"`
	TravelDurationSeconds uint32 `json:"travelDurationSeconds"`
	TransitMessage        string `json:"transitMessage,omitempty"`
}

// GetID returns the resource ID
func (r InstanceRouteRestModel) GetID() string {
	return r.Id
}

// SetID sets the resource ID
func (r *InstanceRouteRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// GetName returns the resource name
func (r InstanceRouteRestModel) GetName() string {
	return "instance-routes"
}

// TransformInstanceRoute converts a map[string]interface{} to an InstanceRouteRestModel
func TransformInstanceRoute(data map[string]interface{}) (InstanceRouteRestModel, error) {
	id, _ := data["id"].(string)

	attributes, ok := data["attributes"].(map[string]interface{})
	if !ok {
		attributes = make(map[string]interface{})
	}

	name, _ := attributes["name"].(string)

	startMapId := uint32(0)
	if val, ok := attributes["startMapId"].(float64); ok {
		startMapId = uint32(val)
	}

	transitMapId := uint32(0)
	if val, ok := attributes["transitMapId"].(float64); ok {
		transitMapId = uint32(val)
	}

	destinationMapId := uint32(0)
	if val, ok := attributes["destinationMapId"].(float64); ok {
		destinationMapId = uint32(val)
	}

	capacity := uint32(0)
	if val, ok := attributes["capacity"].(float64); ok {
		capacity = uint32(val)
	}

	boardingWindowSeconds := uint32(0)
	if val, ok := attributes["boardingWindowSeconds"].(float64); ok {
		boardingWindowSeconds = uint32(val)
	}

	travelDurationSeconds := uint32(0)
	if val, ok := attributes["travelDurationSeconds"].(float64); ok {
		travelDurationSeconds = uint32(val)
	}

	transitMessage, _ := attributes["transitMessage"].(string)

	return InstanceRouteRestModel{
		Id:                    id,
		Name:                  name,
		StartMapId:            startMapId,
		TransitMapId:          transitMapId,
		DestinationMapId:      destinationMapId,
		Capacity:              capacity,
		BoardingWindowSeconds: boardingWindowSeconds,
		TravelDurationSeconds: travelDurationSeconds,
		TransitMessage:        transitMessage,
	}, nil
}

// ExtractInstanceRoute converts an InstanceRouteRestModel to a map[string]interface{}
func ExtractInstanceRoute(r InstanceRouteRestModel) (map[string]interface{}, error) {
	return map[string]interface{}{
		"type": "instance-routes",
		"id":   r.Id,
		"attributes": map[string]interface{}{
			"name":                  r.Name,
			"startMapId":            r.StartMapId,
			"transitMapId":          r.TransitMapId,
			"destinationMapId":      r.DestinationMapId,
			"capacity":              r.Capacity,
			"boardingWindowSeconds": r.BoardingWindowSeconds,
			"travelDurationSeconds": r.TravelDurationSeconds,
			"transitMessage":        r.TransitMessage,
		},
	}, nil
}

// CreateInstanceRouteJsonData creates a JSON:API compliant data structure for instance routes
func CreateInstanceRouteJsonData(routes []map[string]interface{}) (json.RawMessage, error) {
	data := map[string]interface{}{
		"data": routes,
	}
	return json.Marshal(data)
}

// CreateSingleInstanceRouteJsonData creates a JSON:API compliant data structure for a single instance route
func CreateSingleInstanceRouteJsonData(route map[string]interface{}) (json.RawMessage, error) {
	data := map[string]interface{}{
		"data": route,
	}
	return json.Marshal(data)
}
