package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// legacyAddEntry reports whether the tenant is a legacy GMS version (v29..v82)
// whose create-character result decodes only the stat + avatar blocks into an
// empty character slot, with NO list-entry trailer (family/rank bytes are zeroed
// locally, not read). Verified against the v79 client add handler @0x5ceb55
// (Decode1(code) → GW_CharacterStat::Decode → AvatarLook::Decode, then locally
// zeroes the family byte and 16-byte rank buffer). GMS<=28 keeps its own full
// entry format; GMS>=83 and JMS carry the full CharacterListEntry (incl. trailer).
func legacyAddEntry(t tenant.Model) bool {
	return t.Region() == "GMS" && t.MajorVersion() > 28 && t.MajorVersion() < 83
}

const AddCharacterEntryWriter = "AddCharacterEntry"

type AddCharacterEntry struct {
	code      byte
	character model.CharacterListEntry
}

func NewAddCharacterEntry(code byte, character model.CharacterListEntry) AddCharacterEntry {
	return AddCharacterEntry{code: code, character: character}
}

func (m AddCharacterEntry) Code() byte                          { return m.code }
func (m AddCharacterEntry) Character() model.CharacterListEntry { return m.character }
func (m AddCharacterEntry) Operation() string                   { return AddCharacterEntryWriter }
func (m AddCharacterEntry) String() string                      { return fmt.Sprintf("code [%d]", m.code) }

func (m AddCharacterEntry) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.code)
		if legacyAddEntry(t) {
			// Legacy GMS: [code][GW_CharacterStat][AvatarLook] with no entry
			// trailer (family/rank). v79 handler @0x5ceb55 zeroes those locally.
			w.WriteByteArray(m.character.Statistics().Encode(l, ctx)(options))
			w.WriteByteArray(m.character.Avatar().Encode(l, ctx)(options))
			return w.Bytes()
		}
		w.WriteByteArray(m.character.Encode(l, ctx)(options))
		return w.Bytes()
	}
}

func (m *AddCharacterEntry) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.code = r.ReadByte()
		if legacyAddEntry(t) {
			var stats model.CharacterStatistics
			var av model.Avatar
			stats.Decode(l, ctx)(r, options)
			av.Decode(l, ctx)(r, options)
			m.character = model.NewCharacterListEntry(stats, av, false, false, 0, 0, 0, 0)
			return
		}
		m.character.Decode(l, ctx)(r, options)
	}
}
