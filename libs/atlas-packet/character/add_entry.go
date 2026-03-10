package character

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-packet/model"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

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
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.code)
		m.character.Write(l, ctx, w, options, false)
		return w.Bytes()
	}
}

func (m *AddCharacterEntry) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.code = r.ReadByte()
		m.character.Read(l, ctx, r, options, false)
	}
}
