package ops

import saga "github.com/Chronicle20/atlas/libs/atlas-saga"

const (
	opShowIntro             = "show_intro"
	opShowHint              = "show_hint"
	opPlayPortalSound       = "play_portal_sound"
	opApplyConsumableEffect = "apply_consumable_effect"
)

// ShowIntro builds a ShowIntro step, displaying an intro/direction effect to
// the acting character.
//
// Parameters:
//   - path (required) the intro effect path (e.g.
//     "Effect/Direction1.img/aranTutorial/ClickPoleArm").
func ShowIntro(p map[string]string, r Resolver, t Target, characterId uint32) (Step, error) {
	path, err := requiredString(p, r, characterId, opShowIntro, "path")
	if err != nil {
		return Step{}, err
	}

	return newStep(saga.ShowIntro, saga.ShowIntroPayload{
		CharacterId: characterId,
		WorldId:     t.Field().WorldId(),
		ChannelId:   t.Field().ChannelId(),
		Path:        path,
	}), nil
}

// ShowHint builds a ShowHint step, displaying a hint box to the acting
// character.
//
// Parameters:
//   - hint   (required) the hint text.
//   - width  (optional) hint box width in pixels, uint16; 0 (default) is auto.
//   - height (optional) hint box height in pixels, uint16; 0 (default) is auto.
func ShowHint(p map[string]string, r Resolver, t Target, characterId uint32) (Step, error) {
	hint, err := requiredString(p, r, characterId, opShowHint, "hint")
	if err != nil {
		return Step{}, err
	}

	widthInt, err := optionalInt(p, r, characterId, opShowHint, "width", 0)
	if err != nil {
		return Step{}, err
	}
	width, err := rangedUint16(opShowHint, "width", widthInt)
	if err != nil {
		return Step{}, err
	}

	heightInt, err := optionalInt(p, r, characterId, opShowHint, "height", 0)
	if err != nil {
		return Step{}, err
	}
	height, err := rangedUint16(opShowHint, "height", heightInt)
	if err != nil {
		return Step{}, err
	}

	return newStep(saga.ShowHint, saga.ShowHintPayload{
		CharacterId: characterId,
		WorldId:     t.Field().WorldId(),
		ChannelId:   t.Field().ChannelId(),
		Hint:        hint,
		Width:       width,
		Height:      height,
	}), nil
}

// PlayPortalSound builds a PlayPortalSound step, playing the portal sound
// effect for the acting character. Takes no parameters.
func PlayPortalSound(p map[string]string, r Resolver, t Target, characterId uint32) (Step, error) {
	return newStep(saga.PlayPortalSound, saga.PlayPortalSoundPayload{
		CharacterId: characterId,
		WorldId:     t.Field().WorldId(),
		ChannelId:   t.Field().ChannelId(),
	}), nil
}

// ApplyConsumableEffect builds an ApplyConsumableEffect step, applying a
// consumable item's effects to the acting character.
//
// Parameters:
//   - itemId (required) the consumable item id, uint32.
func ApplyConsumableEffect(p map[string]string, r Resolver, t Target, characterId uint32) (Step, error) {
	itemIdInt, err := requiredInt(p, r, characterId, opApplyConsumableEffect, "itemId")
	if err != nil {
		return Step{}, err
	}
	itemId, err := rangedUint32(opApplyConsumableEffect, "itemId", itemIdInt)
	if err != nil {
		return Step{}, err
	}

	return newStep(saga.ApplyConsumableEffect, saga.ApplyConsumableEffectPayload{
		CharacterId: characterId,
		WorldId:     t.Field().WorldId(),
		ChannelId:   t.Field().ChannelId(),
		ItemId:      itemId,
	}), nil
}
