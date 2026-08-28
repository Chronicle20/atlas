package configuration

import (
	"testing"

	"github.com/google/uuid"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestRpsRewardRoundTrip verifies that TransformRpsReward(ExtractRpsReward(m))
// round-trips the entryCostMeso/consolationMeso attributes and a 2-rung
// certificate ladder.
func TestRpsRewardRoundTrip(t *testing.T) {
	original := RpsRewardRestModel{
		Id:              "rps-rewards",
		EntryCostMeso:   1000,
		ConsolationMeso: 500,
		Ladder: []RpsRewardRungRestModel{
			{Rung: 1, ItemId: 4031332, Quantity: 1, Meso: 0},
			{Rung: 2, ItemId: 4031333, Quantity: 1, Meso: 0},
		},
	}

	extracted, err := ExtractRpsReward(original)
	if err != nil {
		t.Fatalf("ExtractRpsReward returned error: %v", err)
	}

	if extracted["type"] != "rps-rewards" {
		t.Fatalf("expected type %q, got %v", "rps-rewards", extracted["type"])
	}

	attributes, ok := extracted["attributes"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected attributes map, got %T", extracted["attributes"])
	}

	// Simulate the JSON:API wire format by wrapping the extracted map exactly
	// as it would be persisted/served (numeric attrs surface as float64 once
	// round-tripped through JSON, so mirror that here rather than relying on
	// the native Go int/uint32 types produced by Extract).
	wireData := map[string]interface{}{
		"id":         extracted["id"],
		"attributes": toFloat64Attributes(attributes),
	}

	transformed, err := TransformRpsReward(wireData)
	if err != nil {
		t.Fatalf("TransformRpsReward returned error: %v", err)
	}

	if transformed.EntryCostMeso != original.EntryCostMeso {
		t.Errorf("EntryCostMeso mismatch: got %d, want %d", transformed.EntryCostMeso, original.EntryCostMeso)
	}

	if transformed.ConsolationMeso != original.ConsolationMeso {
		t.Errorf("ConsolationMeso mismatch: got %d, want %d", transformed.ConsolationMeso, original.ConsolationMeso)
	}

	if len(transformed.Ladder) != len(original.Ladder) {
		t.Fatalf("Ladder length mismatch: got %d, want %d", len(transformed.Ladder), len(original.Ladder))
	}

	for i, rung := range original.Ladder {
		got := transformed.Ladder[i]
		if got.Rung != rung.Rung || got.ItemId != rung.ItemId || got.Quantity != rung.Quantity || got.Meso != rung.Meso {
			t.Errorf("Ladder[%d] mismatch: got %+v, want %+v", i, got, rung)
		}
	}
}

func TestTransformRoute_PopulatesDerivedUuid(t *testing.T) {
	tid := uuid.MustParse("ec876921-c363-4cc6-9c51-5bb8d57f9553")
	data := map[string]interface{}{
		"id":         "boat-ellinia-orbis",
		"type":       "routes",
		"attributes": map[string]interface{}{"name": "boat-ellinia-orbis"},
	}
	rm, err := TransformRoute(tid, data)
	if err != nil {
		t.Fatalf("TransformRoute: %v", err)
	}
	want := tenant.DerivedId(tid, "routes", "boat-ellinia-orbis").String()
	if rm.Uuid != want {
		t.Fatalf("Uuid = %q, want %q", rm.Uuid, want)
	}
	if rm.Id != "boat-ellinia-orbis" {
		t.Fatalf("Id = %q, want the slug (uuid is additive)", rm.Id)
	}
}

func TestTransformInstanceRoute_PopulatesDerivedUuid(t *testing.T) {
	tid := uuid.MustParse("ec876921-c363-4cc6-9c51-5bb8d57f9553")
	data := map[string]interface{}{
		"id":         "temple-of-time-return-flight",
		"type":       "instance-routes",
		"attributes": map[string]interface{}{"name": "temple-of-time-return-flight"},
	}
	rm, err := TransformInstanceRoute(tid, data)
	if err != nil {
		t.Fatalf("TransformInstanceRoute: %v", err)
	}
	want := tenant.DerivedId(tid, "instance-routes", "temple-of-time-return-flight").String()
	if rm.Uuid != want {
		t.Fatalf("Uuid = %q, want %q", rm.Uuid, want)
	}
}

func TestTransformVessel_PopulatesDerivedUuid(t *testing.T) {
	tid := uuid.MustParse("ec876921-c363-4cc6-9c51-5bb8d57f9553")
	data := map[string]interface{}{
		"id":         "boat-ellinia-orbis",
		"type":       "vessels",
		"attributes": map[string]interface{}{"name": "boat-ellinia-orbis"},
	}
	rm, err := TransformVessel(tid, data)
	if err != nil {
		t.Fatalf("TransformVessel: %v", err)
	}
	want := tenant.DerivedId(tid, "vessels", "boat-ellinia-orbis").String()
	if rm.Uuid != want {
		t.Fatalf("Uuid = %q, want %q", rm.Uuid, want)
	}
}

// A route and an instance-route sharing one slug must not share a uuid.
func TestTransform_ResourcesDoNotCollideOnSharedSlug(t *testing.T) {
	tid := uuid.MustParse("ec876921-c363-4cc6-9c51-5bb8d57f9553")
	shared := map[string]interface{}{"id": "shared-slug", "attributes": map[string]interface{}{"name": "shared-slug"}}
	r, _ := TransformRoute(tid, shared)
	i, _ := TransformInstanceRoute(tid, shared)
	if r.Uuid == i.Uuid {
		t.Fatalf("route and instance-route collided on uuid %q", r.Uuid)
	}
}

// TransformInstanceRoute explicitly projects each known attribute out of the
// untyped JSONB. An attribute it does not name is silently dropped before it
// ever reaches atlas-transports — no error, no log. This is that guard.
func TestTransformInstanceRoute_ProjectsEffectAttributes(t *testing.T) {
	tid := uuid.MustParse("ec876921-c363-4cc6-9c51-5bb8d57f9553")
	data := map[string]interface{}{
		"id":   "temple-of-time-flight",
		"type": "instance-routes",
		"attributes": map[string]interface{}{
			"name":              "temple-of-time-flight",
			"effectItemIds":     []interface{}{float64(2210016)},
			"forcedReturnMapId": float64(240000110),
		},
	}
	rm, err := TransformInstanceRoute(tid, data)
	if err != nil {
		t.Fatalf("TransformInstanceRoute: %v", err)
	}
	if len(rm.EffectItemIds) != 1 || rm.EffectItemIds[0] != 2210016 {
		t.Fatalf("EffectItemIds = %v, want [2210016]", rm.EffectItemIds)
	}
	if rm.ForcedReturnMapId != 240000110 {
		t.Fatalf("ForcedReturnMapId = %d, want 240000110", rm.ForcedReturnMapId)
	}
}

// The ten routes that declare neither attribute must project to zero values,
// not to an error.
func TestTransformInstanceRoute_EffectAttributesAreOptional(t *testing.T) {
	tid := uuid.MustParse("ec876921-c363-4cc6-9c51-5bb8d57f9553")
	data := map[string]interface{}{
		"id":         "ellinia-ereve-ferry",
		"type":       "instance-routes",
		"attributes": map[string]interface{}{"name": "ellinia-ereve-ferry"},
	}
	rm, err := TransformInstanceRoute(tid, data)
	if err != nil {
		t.Fatalf("TransformInstanceRoute: %v", err)
	}
	if len(rm.EffectItemIds) != 0 {
		t.Fatalf("EffectItemIds = %v, want empty", rm.EffectItemIds)
	}
	if rm.ForcedReturnMapId != 0 {
		t.Fatalf("ForcedReturnMapId = %d, want 0", rm.ForcedReturnMapId)
	}
}

// ExtractInstanceRoute is the write-back half: a POST/PATCH through the CRUD
// handlers (resource.go:509,558) turns the REST model back into the JSONB
// attribute map. An attribute missing here is erased from storage on the next
// operator edit.
func TestExtractInstanceRoute_RoundTripsEffectAttributes(t *testing.T) {
	tid := uuid.MustParse("ec876921-c363-4cc6-9c51-5bb8d57f9553")
	in := InstanceRouteRestModel{
		Id:                "temple-of-time-flight",
		Name:              "temple-of-time-flight",
		EffectItemIds:     []uint32{2210016},
		ForcedReturnMapId: 240000110,
	}
	extracted, err := ExtractInstanceRoute(in)
	if err != nil {
		t.Fatalf("ExtractInstanceRoute: %v", err)
	}
	out, err := TransformInstanceRoute(tid, toFloat64Attributes(extracted))
	if err != nil {
		t.Fatalf("TransformInstanceRoute: %v", err)
	}
	if len(out.EffectItemIds) != 1 || out.EffectItemIds[0] != 2210016 {
		t.Fatalf("EffectItemIds = %v, want [2210016]", out.EffectItemIds)
	}
	if out.ForcedReturnMapId != 240000110 {
		t.Fatalf("ForcedReturnMapId = %d, want 240000110", out.ForcedReturnMapId)
	}
}

// toFloat64Attributes converts numeric attribute values (and nested ladder
// rung values) to float64, mirroring how they arrive after JSON
// marshal/unmarshal over the wire (encoding/json decodes all JSON numbers
// into map[string]interface{} as float64). It recurses into nested
// map[string]interface{} values (e.g. the "attributes" sub-map inside a full
// Extract* result) and converts []uint32 slices to []interface{} of float64,
// mirroring how a JSON array of numbers decodes.
func toFloat64Attributes(attributes map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(attributes))
	for k, v := range attributes {
		switch val := v.(type) {
		case uint32:
			out[k] = float64(val)
		case []uint32:
			ids := make([]interface{}, 0, len(val))
			for _, id := range val {
				ids = append(ids, float64(id))
			}
			out[k] = ids
		case []RpsRewardRungRestModel:
			rungs := make([]interface{}, 0, len(val))
			for _, r := range val {
				rungs = append(rungs, map[string]interface{}{
					"rung":     float64(r.Rung),
					"itemId":   float64(r.ItemId),
					"quantity": float64(r.Quantity),
					"meso":     float64(r.Meso),
				})
			}
			out[k] = rungs
		case map[string]interface{}:
			out[k] = toFloat64Attributes(val)
		default:
			out[k] = v
		}
	}
	return out
}

func TestKiteConfigTransformExtractRoundTrip(t *testing.T) {
	data := map[string]interface{}{
		"id":                 "kite-configs",
		"maxPerMap":          float64(10),
		"maxMessageLength":   float64(182),
		"blockedMapPrefixes": []interface{}{float64(91)},
	}
	rm, err := TransformKiteConfig(data)
	if err != nil {
		t.Fatalf("TransformKiteConfig: %v", err)
	}
	if rm.MaxPerMap != 10 {
		t.Errorf("MaxPerMap = %d, want 10", rm.MaxPerMap)
	}
	if rm.MaxMessageLength != 182 {
		t.Errorf("MaxMessageLength = %d, want 182", rm.MaxMessageLength)
	}
	if len(rm.BlockedMapPrefixes) != 1 || rm.BlockedMapPrefixes[0] != 91 {
		t.Errorf("BlockedMapPrefixes = %v, want [91]", rm.BlockedMapPrefixes)
	}
	if rm.GetName() != "kite-configs" {
		t.Errorf("GetName() = %s, want kite-configs", rm.GetName())
	}

	out, err := ExtractKiteConfig(rm)
	if err != nil {
		t.Fatalf("ExtractKiteConfig: %v", err)
	}
	if out["maxPerMap"] != 10 {
		t.Errorf("round-trip maxPerMap = %v, want 10", out["maxPerMap"])
	}
}

// TestRouteTransformExtractRoundTrip verifies that
// ExtractRoute(TransformRoute(tenantId, data)) preserves every attribute of
// data, and that the tenant-scoped Uuid derived by Transform (absent from the
// input map) is correct.
func TestRouteTransformExtractRoundTrip(t *testing.T) {
	tid := uuid.MustParse("ec876921-c363-4cc6-9c51-5bb8d57f9553")
	data := map[string]interface{}{
		"id":   "boat-ellinia-orbis",
		"type": "routes",
		"attributes": map[string]interface{}{
			"name":                   "boat-ellinia-orbis",
			"startMapId":             float64(101000000),
			"stagingMapId":           float64(101000001),
			"enRouteMapIds":          []interface{}{float64(101000002), float64(101000003)},
			"destinationMapId":       float64(101000004),
			"observationMapId":       float64(101000005),
			"boardingWindowDuration": float64(30),
			"preDepartureDuration":   float64(15),
			"travelDuration":         float64(600),
			"cycleInterval":          float64(1200),
		},
	}

	rm, err := TransformRoute(tid, data)
	if err != nil {
		t.Fatalf("TransformRoute: %v", err)
	}

	wantUuid := tenant.DerivedId(tid, "routes", "boat-ellinia-orbis").String()
	if rm.Uuid != wantUuid {
		t.Fatalf("Uuid = %q, want %q", rm.Uuid, wantUuid)
	}

	extracted, err := ExtractRoute(rm)
	if err != nil {
		t.Fatalf("ExtractRoute: %v", err)
	}
	if extracted["type"] != "routes" {
		t.Errorf("type = %v, want routes", extracted["type"])
	}
	if extracted["id"] != "boat-ellinia-orbis" {
		t.Errorf("id = %v, want boat-ellinia-orbis", extracted["id"])
	}

	attrs, ok := extracted["attributes"].(map[string]interface{})
	if !ok {
		t.Fatalf("attributes: got %T, want map[string]interface{}", extracted["attributes"])
	}
	if attrs["name"] != "boat-ellinia-orbis" {
		t.Errorf("name = %v, want boat-ellinia-orbis", attrs["name"])
	}
	if attrs["startMapId"] != uint32(101000000) {
		t.Errorf("startMapId = %v, want 101000000", attrs["startMapId"])
	}
	if attrs["stagingMapId"] != uint32(101000001) {
		t.Errorf("stagingMapId = %v, want 101000001", attrs["stagingMapId"])
	}
	enRoute, ok := attrs["enRouteMapIds"].([]uint32)
	if !ok || len(enRoute) != 2 || enRoute[0] != 101000002 || enRoute[1] != 101000003 {
		t.Errorf("enRouteMapIds = %v, want [101000002 101000003]", attrs["enRouteMapIds"])
	}
	if attrs["destinationMapId"] != uint32(101000004) {
		t.Errorf("destinationMapId = %v, want 101000004", attrs["destinationMapId"])
	}
	if attrs["observationMapId"] != uint32(101000005) {
		t.Errorf("observationMapId = %v, want 101000005", attrs["observationMapId"])
	}
	if attrs["boardingWindowDuration"] != uint32(30) {
		t.Errorf("boardingWindowDuration = %v, want 30", attrs["boardingWindowDuration"])
	}
	if attrs["preDepartureDuration"] != uint32(15) {
		t.Errorf("preDepartureDuration = %v, want 15", attrs["preDepartureDuration"])
	}
	if attrs["travelDuration"] != uint32(600) {
		t.Errorf("travelDuration = %v, want 600", attrs["travelDuration"])
	}
	if attrs["cycleInterval"] != uint32(1200) {
		t.Errorf("cycleInterval = %v, want 1200", attrs["cycleInterval"])
	}
}

// TestVesselTransformExtractRoundTrip verifies that
// ExtractVessel(TransformVessel(tenantId, data)) preserves every attribute of
// data, and that the tenant-scoped Uuid derived by Transform is correct.
func TestVesselTransformExtractRoundTrip(t *testing.T) {
	tid := uuid.MustParse("ec876921-c363-4cc6-9c51-5bb8d57f9553")
	data := map[string]interface{}{
		"id":   "vessel-ellinia-orbis",
		"type": "vessels",
		"attributes": map[string]interface{}{
			"name":            "vessel-ellinia-orbis",
			"routeAID":        "boat-ellinia-orbis-outbound",
			"routeBID":        "boat-ellinia-orbis-inbound",
			"turnaroundDelay": float64(45),
		},
	}

	rm, err := TransformVessel(tid, data)
	if err != nil {
		t.Fatalf("TransformVessel: %v", err)
	}

	wantUuid := tenant.DerivedId(tid, "vessels", "vessel-ellinia-orbis").String()
	if rm.Uuid != wantUuid {
		t.Fatalf("Uuid = %q, want %q", rm.Uuid, wantUuid)
	}

	extracted, err := ExtractVessel(rm)
	if err != nil {
		t.Fatalf("ExtractVessel: %v", err)
	}
	if extracted["type"] != "vessels" {
		t.Errorf("type = %v, want vessels", extracted["type"])
	}
	if extracted["id"] != "vessel-ellinia-orbis" {
		t.Errorf("id = %v, want vessel-ellinia-orbis", extracted["id"])
	}

	attrs, ok := extracted["attributes"].(map[string]interface{})
	if !ok {
		t.Fatalf("attributes: got %T, want map[string]interface{}", extracted["attributes"])
	}
	if attrs["name"] != "vessel-ellinia-orbis" {
		t.Errorf("name = %v, want vessel-ellinia-orbis", attrs["name"])
	}
	if attrs["routeAID"] != "boat-ellinia-orbis-outbound" {
		t.Errorf("routeAID = %v, want boat-ellinia-orbis-outbound", attrs["routeAID"])
	}
	if attrs["routeBID"] != "boat-ellinia-orbis-inbound" {
		t.Errorf("routeBID = %v, want boat-ellinia-orbis-inbound", attrs["routeBID"])
	}
	if attrs["turnaroundDelay"] != uint32(45) {
		t.Errorf("turnaroundDelay = %v, want 45", attrs["turnaroundDelay"])
	}
}

// TestMtsConfigTransformExtractRoundTrip verifies that
// ExtractMtsConfig(TransformMtsConfig(data)) preserves every attribute of
// data, matching the concrete Go types TransformMtsConfig actually produces
// (uint32 for the two money-scaled knobs, int for everything else).
func TestMtsConfigTransformExtractRoundTrip(t *testing.T) {
	data := map[string]interface{}{
		"id": "mts-configs",
		"attributes": map[string]interface{}{
			"listingFee":        float64(1000),
			"commissionRate":    float64(0.05),
			"maxActiveListings": float64(20),
			"minLevel":          float64(10),
			"auctionMinHours":   float64(6),
			"auctionMaxHours":   float64(48),
			"priceFloor":        float64(100),
			"pageSize":          float64(25),
			"minBidIncrement":   float64(500),
		},
	}

	rm, err := TransformMtsConfig(data)
	if err != nil {
		t.Fatalf("TransformMtsConfig: %v", err)
	}

	extracted, err := ExtractMtsConfig(rm)
	if err != nil {
		t.Fatalf("ExtractMtsConfig: %v", err)
	}
	if extracted["type"] != "mts-configs" {
		t.Errorf("type = %v, want mts-configs", extracted["type"])
	}
	if extracted["id"] != "mts-configs" {
		t.Errorf("id = %v, want mts-configs", extracted["id"])
	}

	attrs, ok := extracted["attributes"].(map[string]interface{})
	if !ok {
		t.Fatalf("attributes: got %T, want map[string]interface{}", extracted["attributes"])
	}
	if attrs["listingFee"] != uint32(1000) {
		t.Errorf("listingFee = %v, want 1000", attrs["listingFee"])
	}
	if attrs["commissionRate"] != float64(0.05) {
		t.Errorf("commissionRate = %v, want 0.05", attrs["commissionRate"])
	}
	if attrs["maxActiveListings"] != 20 {
		t.Errorf("maxActiveListings = %v, want 20", attrs["maxActiveListings"])
	}
	if attrs["minLevel"] != 10 {
		t.Errorf("minLevel = %v, want 10", attrs["minLevel"])
	}
	if attrs["auctionMinHours"] != 6 {
		t.Errorf("auctionMinHours = %v, want 6", attrs["auctionMinHours"])
	}
	if attrs["auctionMaxHours"] != 48 {
		t.Errorf("auctionMaxHours = %v, want 48", attrs["auctionMaxHours"])
	}
	if attrs["priceFloor"] != uint32(100) {
		t.Errorf("priceFloor = %v, want 100", attrs["priceFloor"])
	}
	if attrs["pageSize"] != 25 {
		t.Errorf("pageSize = %v, want 25", attrs["pageSize"])
	}
	if attrs["minBidIncrement"] != uint32(500) {
		t.Errorf("minBidIncrement = %v, want 500", attrs["minBidIncrement"])
	}
}

// TestImprintConfigTransformExtractRoundTrip verifies that
// ExtractImprintConfig(TransformImprintConfig(data)) preserves the flat
// (non-nested-attributes) config entry.
func TestImprintConfigTransformExtractRoundTrip(t *testing.T) {
	data := map[string]interface{}{
		"id":                 "imprint-configs",
		"pendingExpiryHours": float64(72),
	}

	rm, err := TransformImprintConfig(data)
	if err != nil {
		t.Fatalf("TransformImprintConfig: %v", err)
	}
	if rm.PendingExpiryHours != 72 {
		t.Fatalf("PendingExpiryHours = %d, want 72", rm.PendingExpiryHours)
	}

	extracted, err := ExtractImprintConfig(rm)
	if err != nil {
		t.Fatalf("ExtractImprintConfig: %v", err)
	}
	if extracted["id"] != "imprint-configs" {
		t.Errorf("id = %v, want imprint-configs", extracted["id"])
	}
	if extracted["pendingExpiryHours"] != 72 {
		t.Errorf("pendingExpiryHours = %v, want 72", extracted["pendingExpiryHours"])
	}
}

// TestInstanceRouteTransformExtractRoundTrip verifies that
// ExtractInstanceRoute(TransformInstanceRoute(tenantId, data)) preserves
// every attribute of data, including the optional transitMessage/
// effectItemIds/forcedReturnMapId knobs the existing partial tests do not
// exercise together with the core fields.
func TestInstanceRouteTransformExtractRoundTrip(t *testing.T) {
	tid := uuid.MustParse("ec876921-c363-4cc6-9c51-5bb8d57f9553")
	data := map[string]interface{}{
		"id":   "temple-of-time-return-flight",
		"type": "instance-routes",
		"attributes": map[string]interface{}{
			"name":                  "temple-of-time-return-flight",
			"startMapId":            float64(926010000),
			"transitMapIds":         []interface{}{float64(926010001), float64(926010002)},
			"destinationMapId":      float64(926010003),
			"capacity":              float64(6),
			"boardingWindowSeconds": float64(20),
			"travelDurationSeconds": float64(300),
			"transitMessage":        "The boat departs shortly.",
			"effectItemIds":         []interface{}{float64(2210016)},
			"forcedReturnMapId":     float64(240000110),
		},
	}

	rm, err := TransformInstanceRoute(tid, data)
	if err != nil {
		t.Fatalf("TransformInstanceRoute: %v", err)
	}

	wantUuid := tenant.DerivedId(tid, "instance-routes", "temple-of-time-return-flight").String()
	if rm.Uuid != wantUuid {
		t.Fatalf("Uuid = %q, want %q", rm.Uuid, wantUuid)
	}

	extracted, err := ExtractInstanceRoute(rm)
	if err != nil {
		t.Fatalf("ExtractInstanceRoute: %v", err)
	}
	if extracted["type"] != "instance-routes" {
		t.Errorf("type = %v, want instance-routes", extracted["type"])
	}
	if extracted["id"] != "temple-of-time-return-flight" {
		t.Errorf("id = %v, want temple-of-time-return-flight", extracted["id"])
	}

	attrs, ok := extracted["attributes"].(map[string]interface{})
	if !ok {
		t.Fatalf("attributes: got %T, want map[string]interface{}", extracted["attributes"])
	}
	if attrs["name"] != "temple-of-time-return-flight" {
		t.Errorf("name = %v, want temple-of-time-return-flight", attrs["name"])
	}
	if attrs["startMapId"] != uint32(926010000) {
		t.Errorf("startMapId = %v, want 926010000", attrs["startMapId"])
	}
	transit, ok := attrs["transitMapIds"].([]uint32)
	if !ok || len(transit) != 2 || transit[0] != 926010001 || transit[1] != 926010002 {
		t.Errorf("transitMapIds = %v, want [926010001 926010002]", attrs["transitMapIds"])
	}
	if attrs["destinationMapId"] != uint32(926010003) {
		t.Errorf("destinationMapId = %v, want 926010003", attrs["destinationMapId"])
	}
	if attrs["capacity"] != uint32(6) {
		t.Errorf("capacity = %v, want 6", attrs["capacity"])
	}
	if attrs["boardingWindowSeconds"] != uint32(20) {
		t.Errorf("boardingWindowSeconds = %v, want 20", attrs["boardingWindowSeconds"])
	}
	if attrs["travelDurationSeconds"] != uint32(300) {
		t.Errorf("travelDurationSeconds = %v, want 300", attrs["travelDurationSeconds"])
	}
	if attrs["transitMessage"] != "The boat departs shortly." {
		t.Errorf("transitMessage = %v, want %q", attrs["transitMessage"], "The boat departs shortly.")
	}
	effect, ok := attrs["effectItemIds"].([]uint32)
	if !ok || len(effect) != 1 || effect[0] != 2210016 {
		t.Errorf("effectItemIds = %v, want [2210016]", attrs["effectItemIds"])
	}
	if attrs["forcedReturnMapId"] != uint32(240000110) {
		t.Errorf("forcedReturnMapId = %v, want 240000110", attrs["forcedReturnMapId"])
	}
}
