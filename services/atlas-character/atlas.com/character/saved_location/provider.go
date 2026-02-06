package saved_location

func modelFromEntity(e entity) (Model, error) {
	return NewBuilder().
		SetId(e.ID).
		SetCharacterId(e.CharacterId).
		SetLocationType(e.LocationType).
		SetMapId(e.MapId).
		SetPortalId(e.PortalId).
		Build(), nil
}
