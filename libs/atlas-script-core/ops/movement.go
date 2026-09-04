package ops

import (
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	saga "github.com/Chronicle20/atlas/libs/atlas-saga"
)

const (
	opWarpToPortal           = "warp_to_portal"
	opWarpToSavedLocation    = "warp_to_saved_location"
	opSaveLocation           = "save_location"
	opStartInstanceTransport = "start_instance_transport"
)

// WarpToPortal builds a WarpToPortal step. It backs both the `warp`
// (portal-actions) and `warp_to_map` (npc-conversations) script operations —
// FR-18 keeps both dispatch names valid, and `warp` stays opClassMoving in
// portal's opTable.
//
// Parameters:
//   - mapId      (required) the destination map. npc-conversations previously
//     defaulted this to 0; it is now required (design §5.3). The FR-20 sweep
//     confirmed all 3,377 seeded warp_to_map and 594 warp operations carry it.
//   - portalId   (optional) defaults to 0.
//   - portalName (optional) defaults to "".
//
// Instance is deliberately left uuid.Nil: neither caller populates it today
// and this task does not change destination-field addressing.
func WarpToPortal(p map[string]string, r Resolver, t Target, characterId uint32) (Step, error) {
	mapIdInt, err := requiredInt(p, r, characterId, opWarpToPortal, "mapId")
	if err != nil {
		return Step{}, err
	}
	mapId, err := rangedUint32(opWarpToPortal, "mapId", mapIdInt)
	if err != nil {
		return Step{}, err
	}

	portalIdInt, err := optionalInt(p, r, characterId, opWarpToPortal, "portalId", 0)
	if err != nil {
		return Step{}, err
	}
	portalId, err := rangedUint32(opWarpToPortal, "portalId", portalIdInt)
	if err != nil {
		return Step{}, err
	}

	portalName, err := optionalString(p, r, characterId, opWarpToPortal, "portalName", "")
	if err != nil {
		return Step{}, err
	}

	return newStep(saga.WarpToPortal, saga.WarpToPortalPayload{
		CharacterId: characterId,
		WorldId:     t.Field().WorldId(),
		ChannelId:   t.Field().ChannelId(),
		MapId:       _map.Id(mapId),
		PortalId:    portalId,
		PortalName:  portalName,
	}), nil
}

// WarpToSavedLocation builds a WarpToSavedLocation step, warping the character
// back to a previously saved location (pop semantics: get + warp + delete are
// handled by the orchestrator).
//
// Parameters:
//   - locationType (required) the saved-location key (e.g. "FREE_MARKET").
func WarpToSavedLocation(p map[string]string, r Resolver, t Target, characterId uint32) (Step, error) {
	locationType, err := requiredString(p, r, characterId, opWarpToSavedLocation, "locationType")
	if err != nil {
		return Step{}, err
	}

	return newStep(saga.WarpToSavedLocation, saga.WarpToSavedLocationPayload{
		CharacterId:  characterId,
		WorldId:      t.Field().WorldId(),
		ChannelId:    t.Field().ChannelId(),
		LocationType: locationType,
	}), nil
}

// SaveLocation builds a SaveLocation step, saving the character's current
// location for later retrieval by WarpToSavedLocation.
//
// Parameters:
//   - locationType (required) the saved-location key (e.g. "FREE_MARKET").
//   - mapId        (optional) defaults to t.Field().MapId(), the character's
//     current map.
//   - portalId     (optional) defaults to t.PortalId(), portal-actions'
//     "current portal" default; npc-conversations' unset-portal target
//     resolves the same way to 0.
func SaveLocation(p map[string]string, r Resolver, t Target, characterId uint32) (Step, error) {
	locationType, err := requiredString(p, r, characterId, opSaveLocation, "locationType")
	if err != nil {
		return Step{}, err
	}

	mapIdInt, err := optionalInt(p, r, characterId, opSaveLocation, "mapId", int(t.Field().MapId()))
	if err != nil {
		return Step{}, err
	}
	mapId, err := rangedUint32(opSaveLocation, "mapId", mapIdInt)
	if err != nil {
		return Step{}, err
	}

	portalIdInt, err := optionalInt(p, r, characterId, opSaveLocation, "portalId", int(t.PortalId()))
	if err != nil {
		return Step{}, err
	}
	portalId, err := rangedUint32(opSaveLocation, "portalId", portalIdInt)
	if err != nil {
		return Step{}, err
	}

	return newStep(saga.SaveLocation, saga.SaveLocationPayload{
		CharacterId:  characterId,
		WorldId:      t.Field().WorldId(),
		ChannelId:    t.Field().ChannelId(),
		LocationType: locationType,
		MapId:        _map.Id(mapId),
		PortalId:     portalId,
	}), nil
}

// StartInstanceTransport builds a StartInstanceTransport step, starting an
// instance-based transport for the character (e.g. a subway route).
//
// Parameters:
//   - routeName (required) the transport route name (e.g.
//     "kerning-square-subway-in").
//
// failureMessage is deliberately not read here: it feeds portal-actions'
// pending-action registry (executor.go:459-466) for surfacing a
// client-visible failure message, not the saga payload.
func StartInstanceTransport(p map[string]string, r Resolver, t Target, characterId uint32) (Step, error) {
	routeName, err := requiredString(p, r, characterId, opStartInstanceTransport, "routeName")
	if err != nil {
		return Step{}, err
	}

	return newStep(saga.StartInstanceTransport, saga.StartInstanceTransportPayload{
		CharacterId: characterId,
		WorldId:     t.Field().WorldId(),
		ChannelId:   t.Field().ChannelId(),
		RouteName:   routeName,
	}), nil
}
