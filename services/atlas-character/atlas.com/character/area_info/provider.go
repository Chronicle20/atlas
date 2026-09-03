package area_info

func modelFromEntity(e entity) (Model, error) {
	return NewBuilder().
		SetId(e.ID).
		SetCharacterId(e.CharacterId).
		SetArea(e.Area).
		SetInfo(e.Info).
		Build(), nil
}
