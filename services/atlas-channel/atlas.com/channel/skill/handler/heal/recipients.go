package heal

import (
	"atlas-channel/skill/handler"
)

// selectRecipients prepends the caster to the in-range party members
// returned by the shared resolver. Caller is responsible for the
// shared resolver call so test stubs don't need to fake the whole
// processor stack.
func selectRecipients(caster recipient, party []handler.PartyRecipient) []recipient {
	out := make([]recipient, 0, 1+len(party))
	out = append(out, caster)
	for _, p := range party {
		out = append(out, recipient{
			Id:    p.Id(),
			X:     p.X(),
			Y:     p.Y(),
			Hp:    p.Hp(),
			MaxHp: p.MaxHp(),
			Level: p.Level(),
		})
	}
	return out
}
