package item

// Extra-expression (emote) cash items — Item.wz/Cash/0516.img. These are
// permanent unlocks with no `spec` node: owning one extends the character's
// emote palette beyond the seven free base emotes. They are never consumed.

// MaxEmoteId is the highest emote id a stock client can put on the wire in
// either direction. CWvsContext::SendEmotionChange@0x9f9386 (GMS v95) refuses
// to send above 0x17, and CAvatar::SetEmotion@0x466b00 re-checks the same
// bound before applying a received emotion.
const MaxEmoteId = uint32(23)

// MaxBaseEmoteId is the highest emote every character has without owning a
// cash item. Emotes above it are ClassificationExpression (516) unlocks.
const MaxBaseEmoteId = uint32(7)

// IsExtraExpressionEmote reports whether an emote requires an owned
// ClassificationExpression cash item.
func IsExtraExpressionEmote(emote uint32) bool {
	return emote > MaxBaseEmoteId && emote <= MaxEmoteId
}

// ExtraExpressionItemId maps an extra-expression emote to the cash item that
// unlocks it, reporting false for emotes outside the gated range.
//
// CWvsContext::SendEtcCashItemUseRequest@0xa02c86 and
// CUserLocal::UseFuncKeyMapped case 3u@0x933874 (GMS v95) both compute the
// emote as nItemID % 100 + 8, so the item's index within classification 516
// is emote - 8: emote 8 -> 5160000, emote 22 -> 5160014. Emote 23 yields
// 5160015, which has no entry in v83.1 data — no special case is needed
// because the caller's ownership check fails on it naturally.
func ExtraExpressionItemId(emote uint32) (Id, bool) {
	if !IsExtraExpressionEmote(emote) {
		return Id(0), false
	}
	return Id(uint32(ClassificationExpression)*10000 + emote - MaxBaseEmoteId - 1), true
}
