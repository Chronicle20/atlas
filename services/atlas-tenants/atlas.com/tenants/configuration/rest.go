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
	return CreateRouteJsonData([]map[string]interface{}{route})
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
	return CreateVesselJsonData([]map[string]interface{}{vessel})
}

// MtsConfigRestModel is the JSON:API resource for the per-tenant MTS economic
// configuration. The attribute JSON keys must match what atlas-mts decodes in
// services/atlas-mts/atlas.com/mts/configuration/model.go (RestModel).
type MtsConfigRestModel struct {
	Id                string  `json:"-"`
	ListingFee        uint32  `json:"listingFee"`
	CommissionRate    float64 `json:"commissionRate"`
	MaxActiveListings int     `json:"maxActiveListings"`
	MinLevel          int     `json:"minLevel"`
	AuctionMinHours   int     `json:"auctionMinHours"`
	AuctionMaxHours   int     `json:"auctionMaxHours"`
	PriceFloor        uint32  `json:"priceFloor"`
	PageSize          int     `json:"pageSize"`
	MinBidIncrement   uint32  `json:"minBidIncrement"`
}

// GetID returns the resource ID
func (m MtsConfigRestModel) GetID() string {
	return m.Id
}

// SetID sets the resource ID
func (m *MtsConfigRestModel) SetID(id string) error {
	m.Id = id
	return nil
}

// GetName returns the resource name
func (m MtsConfigRestModel) GetName() string {
	return "mts-configs"
}

// TransformMtsConfig converts a map[string]interface{} to an MtsConfigRestModel
func TransformMtsConfig(data map[string]interface{}) (MtsConfigRestModel, error) {
	id, _ := data["id"].(string)

	attributes, ok := data["attributes"].(map[string]interface{})
	if !ok {
		attributes = make(map[string]interface{})
	}

	listingFee := uint32(0)
	if val, ok := attributes["listingFee"].(float64); ok {
		listingFee = uint32(val)
	}

	commissionRate := float64(0)
	if val, ok := attributes["commissionRate"].(float64); ok {
		commissionRate = val
	}

	maxActiveListings := 0
	if val, ok := attributes["maxActiveListings"].(float64); ok {
		maxActiveListings = int(val)
	}

	minLevel := 0
	if val, ok := attributes["minLevel"].(float64); ok {
		minLevel = int(val)
	}

	auctionMinHours := 0
	if val, ok := attributes["auctionMinHours"].(float64); ok {
		auctionMinHours = int(val)
	}

	auctionMaxHours := 0
	if val, ok := attributes["auctionMaxHours"].(float64); ok {
		auctionMaxHours = int(val)
	}

	priceFloor := uint32(0)
	if val, ok := attributes["priceFloor"].(float64); ok {
		priceFloor = uint32(val)
	}

	pageSize := 0
	if val, ok := attributes["pageSize"].(float64); ok {
		pageSize = int(val)
	}

	minBidIncrement := uint32(0)
	if val, ok := attributes["minBidIncrement"].(float64); ok {
		minBidIncrement = uint32(val)
	}

	return MtsConfigRestModel{
		Id:                id,
		ListingFee:        listingFee,
		CommissionRate:    commissionRate,
		MaxActiveListings: maxActiveListings,
		MinLevel:          minLevel,
		AuctionMinHours:   auctionMinHours,
		AuctionMaxHours:   auctionMaxHours,
		PriceFloor:        priceFloor,
		PageSize:          pageSize,
		MinBidIncrement:   minBidIncrement,
	}, nil
}

// ExtractMtsConfig converts an MtsConfigRestModel to a map[string]interface{}
func ExtractMtsConfig(m MtsConfigRestModel) (map[string]interface{}, error) {
	return map[string]interface{}{
		"type": "mts-configs",
		"id":   m.Id,
		"attributes": map[string]interface{}{
			"listingFee":        m.ListingFee,
			"commissionRate":    m.CommissionRate,
			"maxActiveListings": m.MaxActiveListings,
			"minLevel":          m.MinLevel,
			"auctionMinHours":   m.AuctionMinHours,
			"auctionMaxHours":   m.AuctionMaxHours,
			"priceFloor":        m.PriceFloor,
			"pageSize":          m.PageSize,
			"minBidIncrement":   m.MinBidIncrement,
		},
	}, nil
}

// CreateMtsConfigJsonData creates a JSON:API compliant data structure for mts configs
func CreateMtsConfigJsonData(configs []map[string]interface{}) (json.RawMessage, error) {
	data := map[string]interface{}{
		"data": configs,
	}
	return json.Marshal(data)
}

// CreateSingleMtsConfigJsonData creates a JSON:API compliant data structure for a single mts config
func CreateSingleMtsConfigJsonData(config map[string]interface{}) (json.RawMessage, error) {
	return CreateMtsConfigJsonData([]map[string]interface{}{config})
}

// InstanceRouteRestModel is the JSON:API resource for instance routes
type InstanceRouteRestModel struct {
	Id                    string `json:"-"`
	Name                  string `json:"name"`
	StartMapId            uint32 `json:"startMapId"`
	TransitMapIds         []uint32 `json:"transitMapIds"`
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

	var transitMapIds []uint32
	if val, ok := attributes["transitMapIds"].([]interface{}); ok {
		for _, v := range val {
			if f, ok := v.(float64); ok {
				transitMapIds = append(transitMapIds, uint32(f))
			}
		}
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
		TransitMapIds:         transitMapIds,
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
			"transitMapIds":         r.TransitMapIds,
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
	return CreateInstanceRouteJsonData([]map[string]interface{}{route})
}

// RpsRewardRungRestModel is the nested JSON attribute shape of a single rung
// embedded in the rps-rewards `ladder` array.
type RpsRewardRungRestModel struct {
	Rung     int    `json:"rung"`
	ItemId   uint32 `json:"itemId"`
	Quantity uint32 `json:"quantity"`
	Meso     uint32 `json:"meso"`
}

// RpsRewardRestModel is the JSON:API resource for the rps-rewards configuration
type RpsRewardRestModel struct {
	Id            string                   `json:"-"`
	EntryCostMeso uint32                   `json:"entryCostMeso"`
	Ladder        []RpsRewardRungRestModel `json:"ladder"`
}

// GetID returns the resource ID
func (r RpsRewardRestModel) GetID() string {
	return r.Id
}

// SetID sets the resource ID
func (r *RpsRewardRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// GetName returns the resource name
func (r RpsRewardRestModel) GetName() string {
	return "rps-rewards"
}

// TransformRpsReward converts a map[string]interface{} to a RpsRewardRestModel
func TransformRpsReward(data map[string]interface{}) (RpsRewardRestModel, error) {
	id, _ := data["id"].(string)

	attributes, ok := data["attributes"].(map[string]interface{})
	if !ok {
		attributes = make(map[string]interface{})
	}

	entryCostMeso := uint32(0)
	if val, ok := attributes["entryCostMeso"].(float64); ok {
		entryCostMeso = uint32(val)
	}

	ladder := make([]RpsRewardRungRestModel, 0)
	if vals, ok := attributes["ladder"].([]interface{}); ok {
		for _, v := range vals {
			rungMap, ok := v.(map[string]interface{})
			if !ok {
				continue
			}

			rung := 0
			if val, ok := rungMap["rung"].(float64); ok {
				rung = int(val)
			}

			itemId := uint32(0)
			if val, ok := rungMap["itemId"].(float64); ok {
				itemId = uint32(val)
			}

			quantity := uint32(0)
			if val, ok := rungMap["quantity"].(float64); ok {
				quantity = uint32(val)
			}

			meso := uint32(0)
			if val, ok := rungMap["meso"].(float64); ok {
				meso = uint32(val)
			}

			ladder = append(ladder, RpsRewardRungRestModel{
				Rung:     rung,
				ItemId:   itemId,
				Quantity: quantity,
				Meso:     meso,
			})
		}
	}

	return RpsRewardRestModel{
		Id:            id,
		EntryCostMeso: entryCostMeso,
		Ladder:        ladder,
	}, nil
}

// ExtractRpsReward converts a RpsRewardRestModel to a map[string]interface{}
func ExtractRpsReward(r RpsRewardRestModel) (map[string]interface{}, error) {
	return map[string]interface{}{
		"type": "rps-rewards",
		"id":   r.Id,
		"attributes": map[string]interface{}{
			"entryCostMeso": r.EntryCostMeso,
			"ladder":        r.Ladder,
		},
	}, nil
}

// CreateRpsRewardJsonData creates a JSON:API compliant data structure for rps-rewards
func CreateRpsRewardJsonData(rewards []map[string]interface{}) (json.RawMessage, error) {
	data := map[string]interface{}{
		"data": rewards,
	}
	return json.Marshal(data)
}

// CreateSingleRpsRewardJsonData creates a JSON:API compliant data structure for a single rps-reward
func CreateSingleRpsRewardJsonData(reward map[string]interface{}) (json.RawMessage, error) {
	return CreateRpsRewardJsonData([]map[string]interface{}{reward})
}
