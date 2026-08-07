package merchant

// SelectOpenShop picks the one live shop from a character's shops for owl-warp
// resolution. A character accumulates StateClosed leftovers over a session, so
// callers must not assume shops[0] is current. Preference order:
//   - a StateOpen shop (what the owl search surfaces) wins outright; there is at
//     most one per character.
//   - otherwise the first non-Closed shop (draft/maintenance) is returned so the
//     warp ladder can still report its true state (e.g. MAINTENANCE).
//   - if every shop is Closed (or the slice is empty), returns (_, false) so the
//     caller leaves ShopFound false and the ladder answers CLOSED honestly.
func SelectOpenShop(shops []Model) (Model, bool) {
	var alt Model
	haveAlt := false
	for _, s := range shops {
		if s.state == StateOpen {
			return s, true
		}
		if s.state != StateClosed && !haveAlt {
			alt, haveAlt = s, true
		}
	}
	return alt, haveAlt
}
