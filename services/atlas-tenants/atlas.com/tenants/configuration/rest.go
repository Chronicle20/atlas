package configuration

import (
	"encoding/json"

	"github.com/google/uuid"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// RouteRestModel is the JSON:API resource for routes
type RouteRestModel struct {
	Id string `json:"-"`
	// Uuid is a stable, tenant-scoped UUIDv5 derived from
	// (tenantId, resourceName, slug). It exists so atlas-transports can
	// key its Redis registry on an id that survives restarts and is
	// identical across replicas. The JSON:API resource id stays the
	// slug, because configuration-status events and the CRUD routes
	// reference resources by slug.
	Uuid                   string   `json:"uuid"`
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
func TransformRoute(tenantId uuid.UUID, data map[string]interface{}) (RouteRestModel, error) {
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
		Uuid:                   tenant.DerivedId(tenantId, "routes", id).String(),
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
	Id string `json:"-"`
	// Uuid is a stable, tenant-scoped UUIDv5 derived from
	// (tenantId, resourceName, slug). It exists so atlas-transports can
	// key its Redis registry on an id that survives restarts and is
	// identical across replicas. The JSON:API resource id stays the
	// slug, because configuration-status events and the CRUD routes
	// reference resources by slug.
	Uuid            string `json:"uuid"`
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
func TransformVessel(tenantId uuid.UUID, data map[string]interface{}) (VesselRestModel, error) {
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
		Uuid:            tenant.DerivedId(tenantId, "vessels", id).String(),
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

// TradeTaxTierRestModel is one meso-tax band inside a TradeConfigRestModel's
// taxTiers array. The JSON keys must match what atlas-trades decodes in
// services/atlas-trades/atlas.com/trades/configuration/rest.go (TierRestModel).
type TradeTaxTierRestModel struct {
	Threshold uint32  `json:"threshold"`
	Rate      float64 `json:"rate"`
}

// TradeConfigRestModel is the JSON:API resource for the per-tenant
// player-to-player trade configuration. The attribute JSON keys must match what
// atlas-trades decodes in
// services/atlas-trades/atlas.com/trades/configuration/rest.go (RestModel).
// EVERY knob is optional, because PATCH decodes into this whole struct and
// ExtractTradeConfig writes the whole attributes object back. A non-optional
// field would arrive at its Go zero value on a request that never mentioned it
// and silently overwrite the stored setting — that is how a PATCH of one knob
// wipes the rest.
//
// nil (or, for TaxTiers, empty) means "the request did not mention this knob":
// ExtractTradeConfig omits the key entirely, and UpdateTradeConfig's attribute
// merge carries the stored value forward. An explicit zero value is a non-nil
// pointer and IS transmitted and stored — api2go marshals via encoding/json
// (api2go@v1.0.4/jsonapi/marshal.go), whose omitempty omits only a nil pointer,
// never a pointer to false or 0. So `minTradeLevel: 0` and `taxEnabled: false`
// both survive the round trip and win over the stored value.
//
// TaxTiers is a slice, where omitempty cannot distinguish nil from empty — but
// an empty tax table is meaningless (atlas-trades substitutes the shipped table
// for one anyway, see WithTaxTiers), so collapsing both to "not mentioned" is
// the correct reading.
type TradeConfigRestModel struct {
	Id                        string                  `json:"-"`
	TaxEnabled                *bool                   `json:"taxEnabled,omitempty"`
	TaxTiers                  []TradeTaxTierRestModel `json:"taxTiers,omitempty"`
	MaxStagedItems            *int                    `json:"maxStagedItems,omitempty"`
	MinTradeLevel             *int                    `json:"minTradeLevel,omitempty"`
	ReservationTtlSeconds     *int                    `json:"reservationTtlSeconds,omitempty"`
	AttestationTimeoutSeconds *int                    `json:"attestationTimeoutSeconds,omitempty"`
}

// GetID returns the resource ID
func (m TradeConfigRestModel) GetID() string {
	return m.Id
}

// SetID sets the resource ID
func (m *TradeConfigRestModel) SetID(id string) error {
	m.Id = id
	return nil
}

// GetName returns the resource name
func (m TradeConfigRestModel) GetName() string {
	return "trade-configs"
}

// TransformTradeConfig converts a map[string]interface{} to a
// TradeConfigRestModel.
//
// taxTiers is an array of objects rather than a scalar, so it decodes as
// []interface{} of map[string]interface{} with float64 numbers — its arm below
// differs from the `attributes["x"].(float64)` pattern the scalar knobs use.
func TransformTradeConfig(data map[string]interface{}) (TradeConfigRestModel, error) {
	id, _ := data["id"].(string)

	attributes, ok := data["attributes"].(map[string]interface{})
	if !ok {
		attributes = make(map[string]interface{})
	}

	// Absent stays nil so a PATCH that never mentioned the knob cannot flip it.
	var taxEnabled *bool
	if val, ok := attributes["taxEnabled"].(bool); ok {
		taxEnabled = &val
	}

	var taxTiers []TradeTaxTierRestModel
	if raw, ok := attributes["taxTiers"].([]interface{}); ok {
		taxTiers = make([]TradeTaxTierRestModel, 0, len(raw))
		for _, entry := range raw {
			tier, ok := entry.(map[string]interface{})
			if !ok {
				continue
			}
			threshold := uint32(0)
			if val, ok := tier["threshold"].(float64); ok {
				threshold = uint32(val)
			}
			rate := float64(0)
			if val, ok := tier["rate"].(float64); ok {
				rate = val
			}
			taxTiers = append(taxTiers, TradeTaxTierRestModel{Threshold: threshold, Rate: rate})
		}
	}

	return TradeConfigRestModel{
		Id:                        id,
		TaxEnabled:                taxEnabled,
		TaxTiers:                  taxTiers,
		MaxStagedItems:            optionalInt(attributes, "maxStagedItems"),
		MinTradeLevel:             optionalInt(attributes, "minTradeLevel"),
		ReservationTtlSeconds:     optionalInt(attributes, "reservationTtlSeconds"),
		AttestationTimeoutSeconds: optionalInt(attributes, "attestationTimeoutSeconds"),
	}, nil
}

// optionalInt reads a JSON number attribute into a *int, leaving it nil when the
// attribute is absent. Absent must stay distinguishable from an explicit 0:
// minTradeLevel's default IS zero, so a wiped value would otherwise be
// indistinguishable from an operator intentionally clearing the level gate.
func optionalInt(attributes map[string]interface{}, key string) *int {
	val, ok := attributes[key].(float64)
	if !ok {
		return nil
	}
	out := int(val)
	return &out
}

// ExtractTradeConfig converts a TradeConfigRestModel to a map[string]interface{}.
//
// An attribute the caller left unset is OMITTED from the returned map rather
// than written at its Go zero value, so UpdateTradeConfig's merge carries the
// stored value forward. Writing every key unconditionally is what let a PATCH
// naming one knob reset all the others.
func ExtractTradeConfig(m TradeConfigRestModel) (map[string]interface{}, error) {
	attributes := make(map[string]interface{}, 6)

	if m.TaxEnabled != nil {
		attributes["taxEnabled"] = *m.TaxEnabled
	}
	// An empty table is never a meaningful instruction, so treat it as "not
	// mentioned" and keep whatever tiers are stored.
	if len(m.TaxTiers) > 0 {
		tiers := make([]map[string]interface{}, 0, len(m.TaxTiers))
		for _, t := range m.TaxTiers {
			tiers = append(tiers, map[string]interface{}{
				"threshold": t.Threshold,
				"rate":      t.Rate,
			})
		}
		attributes["taxTiers"] = tiers
	}
	if m.MaxStagedItems != nil {
		attributes["maxStagedItems"] = *m.MaxStagedItems
	}
	if m.MinTradeLevel != nil {
		attributes["minTradeLevel"] = *m.MinTradeLevel
	}
	if m.ReservationTtlSeconds != nil {
		attributes["reservationTtlSeconds"] = *m.ReservationTtlSeconds
	}
	if m.AttestationTimeoutSeconds != nil {
		attributes["attestationTimeoutSeconds"] = *m.AttestationTimeoutSeconds
	}

	return map[string]interface{}{
		"type":       "trade-configs",
		"id":         m.Id,
		"attributes": attributes,
	}, nil
}

// mergeTradeConfigAttributes returns a copy of incoming whose attributes carry
// every attribute the stored config had but that incoming omitted. Attributes
// present in incoming always win, including an explicit false or zero.
//
// This only protects a knob if ExtractTradeConfig actually omits its key when
// the caller did not set it — the merge cannot rescue a key that arrives
// written at its zero value. The two halves are a pair: every field of
// TradeConfigRestModel is optional, and ExtractTradeConfig omits each unset one.
func mergeTradeConfigAttributes(existing map[string]interface{}, incoming map[string]interface{}) map[string]interface{} {
	existingAttributes, ok := existing["attributes"].(map[string]interface{})
	if !ok {
		return incoming
	}

	incomingAttributes, ok := incoming["attributes"].(map[string]interface{})
	if !ok {
		incomingAttributes = make(map[string]interface{})
	}

	merged := make(map[string]interface{}, len(existingAttributes)+len(incomingAttributes))
	for k, v := range existingAttributes {
		merged[k] = v
	}
	for k, v := range incomingAttributes {
		merged[k] = v
	}

	out := make(map[string]interface{}, len(incoming))
	for k, v := range incoming {
		out[k] = v
	}
	out["attributes"] = merged
	return out
}

// CreateTradeConfigJsonData creates a JSON:API compliant data structure for trade configs
func CreateTradeConfigJsonData(configs []map[string]interface{}) (json.RawMessage, error) {
	data := map[string]interface{}{
		"data": configs,
	}
	return json.Marshal(data)
}

// CreateSingleTradeConfigJsonData creates a JSON:API compliant data structure for a single trade config
func CreateSingleTradeConfigJsonData(config map[string]interface{}) (json.RawMessage, error) {
	return CreateTradeConfigJsonData([]map[string]interface{}{config})
}

// InstanceRouteRestModel is the JSON:API resource for instance routes
type InstanceRouteRestModel struct {
	Id string `json:"-"`
	// Uuid is a stable, tenant-scoped UUIDv5 derived from
	// (tenantId, resourceName, slug). It exists so atlas-transports can
	// key its Redis registry on an id that survives restarts and is
	// identical across replicas. The JSON:API resource id stays the
	// slug, because configuration-status events and the CRUD routes
	// reference resources by slug.
	Uuid                  string   `json:"uuid"`
	Name                  string   `json:"name"`
	StartMapId            uint32   `json:"startMapId"`
	TransitMapIds         []uint32 `json:"transitMapIds"`
	DestinationMapId      uint32   `json:"destinationMapId"`
	Capacity              uint32   `json:"capacity"`
	BoardingWindowSeconds uint32   `json:"boardingWindowSeconds"`
	TravelDurationSeconds uint32   `json:"travelDurationSeconds"`
	TransitMessage        string   `json:"transitMessage,omitempty"`
	// EffectItemIds are the consumable item ids atlas-transports applies on
	// boarding and cancels on every terminal path of this route. Optional.
	EffectItemIds []uint32 `json:"effectItemIds,omitempty"`
	// ForcedReturnMapId is where atlas-transports warps a character whose
	// travel timer expired, instead of destinationMapId. Zero means "not
	// set". It mirrors the client's Map.wz info/forcedReturn node. Optional.
	ForcedReturnMapId uint32 `json:"forcedReturnMapId,omitempty"`
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
func TransformInstanceRoute(tenantId uuid.UUID, data map[string]interface{}) (InstanceRouteRestModel, error) {
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

	var effectItemIds []uint32
	if val, ok := attributes["effectItemIds"].([]interface{}); ok {
		for _, v := range val {
			if f, ok := v.(float64); ok {
				effectItemIds = append(effectItemIds, uint32(f))
			}
		}
	}

	forcedReturnMapId := uint32(0)
	if val, ok := attributes["forcedReturnMapId"].(float64); ok {
		forcedReturnMapId = uint32(val)
	}

	return InstanceRouteRestModel{
		Id:                    id,
		Uuid:                  tenant.DerivedId(tenantId, "instance-routes", id).String(),
		Name:                  name,
		StartMapId:            startMapId,
		TransitMapIds:         transitMapIds,
		DestinationMapId:      destinationMapId,
		Capacity:              capacity,
		BoardingWindowSeconds: boardingWindowSeconds,
		TravelDurationSeconds: travelDurationSeconds,
		TransitMessage:        transitMessage,
		EffectItemIds:         effectItemIds,
		ForcedReturnMapId:     forcedReturnMapId,
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
			"effectItemIds":         r.EffectItemIds,
			"forcedReturnMapId":     r.ForcedReturnMapId,
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

// RankingsRestModel is the JSON:API resource for the per-tenant rankings
// configuration. It is a single-object resource (no id-scoped sub-routes);
// the attribute name/type must match what atlas-rankings decodes in
// services/atlas-rankings/atlas.com/rankings/configuration/rest.go (RestModel).
type RankingsRestModel struct {
	Id                       string `json:"-"`
	RecomputeIntervalMinutes uint32 `json:"recomputeIntervalMinutes"`
}

// GetID returns the resource ID
func (r RankingsRestModel) GetID() string {
	return r.Id
}

// SetID sets the resource ID
func (r *RankingsRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// GetName returns the resource name
func (r RankingsRestModel) GetName() string {
	return "rankings"
}

// TransformRankings converts a map[string]interface{} to a RankingsRestModel
func TransformRankings(data map[string]interface{}) (RankingsRestModel, error) {
	id, _ := data["id"].(string)

	attributes, ok := data["attributes"].(map[string]interface{})
	if !ok {
		attributes = make(map[string]interface{})
	}

	interval := uint32(0)
	if val, ok := attributes["recomputeIntervalMinutes"].(float64); ok {
		interval = uint32(val)
	}

	return RankingsRestModel{Id: id, RecomputeIntervalMinutes: interval}, nil
}

// ExtractRankings converts a RankingsRestModel to a map[string]interface{}
func ExtractRankings(r RankingsRestModel) (map[string]interface{}, error) {
	id := r.Id
	if id == "" {
		id = uuid.New().String()
	}
	return map[string]interface{}{
		"type": "rankings",
		"id":   id,
		"attributes": map[string]interface{}{
			"recomputeIntervalMinutes": r.RecomputeIntervalMinutes,
		},
	}, nil
}

// CreateSingleRankingsJsonData creates a JSON:API compliant data structure
// for the single-object rankings configuration
func CreateSingleRankingsJsonData(rankings map[string]interface{}) (json.RawMessage, error) {
	return json.Marshal(map[string]interface{}{"data": rankings})
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
	Id string `json:"-"`
	// EntryCostMeso is the participation fee charged to enter (and to Retry).
	EntryCostMeso uint32 `json:"entryCostMeso"`
	// ConsolationMeso is the meso awarded on a loss (the client's "consolation
	// prize" message). 0 disables the consolation award.
	ConsolationMeso uint32                   `json:"consolationMeso"`
	Ladder          []RpsRewardRungRestModel `json:"ladder"`
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

	consolationMeso := uint32(0)
	if val, ok := attributes["consolationMeso"].(float64); ok {
		consolationMeso = uint32(val)
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
		Id:              id,
		EntryCostMeso:   entryCostMeso,
		ConsolationMeso: consolationMeso,
		Ladder:          ladder,
	}, nil
}

// ExtractRpsReward converts a RpsRewardRestModel to a map[string]interface{}
func ExtractRpsReward(r RpsRewardRestModel) (map[string]interface{}, error) {
	return map[string]interface{}{
		"type": "rps-rewards",
		"id":   r.Id,
		"attributes": map[string]interface{}{
			"entryCostMeso":   r.EntryCostMeso,
			"consolationMeso": r.ConsolationMeso,
			"ladder":          r.Ladder,
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
