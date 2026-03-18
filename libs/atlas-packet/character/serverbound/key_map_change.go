package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterKeyMapChangeHandle = "CharacterKeyMapChangeHandle"

type KeyMapEntry struct {
	KeyId   int32
	TheType int8
	Action  int32
}

// KeyMapChange - CUser::SendFuncKeyMappedModified
type KeyMapChange struct {
	mode    uint32
	entries []KeyMapEntry
	itemId  uint32
}

func (m KeyMapChange) Mode() uint32            { return m.mode }
func (m KeyMapChange) Entries() []KeyMapEntry   { return m.entries }
func (m KeyMapChange) ItemId() uint32           { return m.itemId }

func (m KeyMapChange) Operation() string {
	return CharacterKeyMapChangeHandle
}

func (m KeyMapChange) String() string {
	if m.mode == 0 {
		return fmt.Sprintf("mode [%d], entries [%d]", m.mode, len(m.entries))
	}
	return fmt.Sprintf("mode [%d], itemId [%d]", m.mode, m.itemId)
}

func (m KeyMapChange) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.mode)
		if m.mode == 0 {
			w.WriteInt(uint32(len(m.entries)))
			for _, e := range m.entries {
				w.WriteInt32(e.KeyId)
				w.WriteInt8(e.TheType)
				w.WriteInt32(e.Action)
			}
		} else {
			w.WriteInt(m.itemId)
		}
		return w.Bytes()
	}
}

func (m *KeyMapChange) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadUint32()
		if m.mode == 0 {
			changes := r.ReadUint32()
			m.entries = make([]KeyMapEntry, 0, changes)
			for range changes {
				keyId := r.ReadInt32()
				theType := r.ReadInt8()
				action := r.ReadInt32()
				m.entries = append(m.entries, KeyMapEntry{KeyId: keyId, TheType: theType, Action: action})
			}
		} else {
			m.itemId = r.ReadUint32()
		}
	}
}
