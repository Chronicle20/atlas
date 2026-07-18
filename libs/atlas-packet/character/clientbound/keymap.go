package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const CharacterKeyMapWriter = "CharacterKeyMap"

// keyMapEntryCount returns the number of FUNCKEY_MAPPED entries the client reads
// in the non-reset KEYMAP path. Legacy GMS (< v83) reads 89 entries
// (CFuncKeyMappedMan::OnInit v79 @0x569e69 `v5 = 89`, memcpy 0x1BD = 445 = 89*5);
// v83+ keeps the historical 90 the codec has always emitted (documented as a
// benign over-send / truncation in the gms_v83 CharacterKeyMap evidence notes).
func keyMapEntryCount(ctx context.Context) int32 {
	t := tenant.MustFromContext(ctx)
	if t.Region() == "GMS" && t.MajorVersion() < 83 {
		return 89
	}
	return 90
}

type KeyBinding struct {
	KeyType   int8
	KeyAction int32
}

type CharacterKeyMap struct {
	resetToDefault bool
	keys           map[int32]KeyBinding
}

func NewCharacterKeyMap(keys map[int32]KeyBinding) CharacterKeyMap {
	return CharacterKeyMap{keys: keys}
}

func NewCharacterKeyMapResetToDefault() CharacterKeyMap {
	return CharacterKeyMap{resetToDefault: true}
}

func (m CharacterKeyMap) ResetToDefault() bool       { return m.resetToDefault }
func (m CharacterKeyMap) Keys() map[int32]KeyBinding { return m.keys }
func (m CharacterKeyMap) Operation() string          { return CharacterKeyMapWriter }
func (m CharacterKeyMap) String() string {
	if m.resetToDefault {
		return "resetToDefault"
	}
	return fmt.Sprintf("keys [%d]", len(m.keys))
}

func (m CharacterKeyMap) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	count := keyMapEntryCount(ctx)
	return func(options map[string]interface{}) []byte {
		if m.resetToDefault {
			w.WriteByte(1)
			return w.Bytes()
		}
		w.WriteByte(0)
		for i := int32(0); i < count; i++ {
			if k, ok := m.keys[i]; ok {
				w.WriteInt8(k.KeyType)
				w.WriteInt32(k.KeyAction)
			} else {
				w.WriteInt8(0)
				w.WriteInt32(0)
			}
		}
		return w.Bytes()
	}
}

func (m *CharacterKeyMap) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	count := keyMapEntryCount(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		flag := r.ReadByte()
		if flag == 1 {
			m.resetToDefault = true
			return
		}
		m.keys = make(map[int32]KeyBinding)
		for i := int32(0); i < count; i++ {
			keyType := r.ReadInt8()
			keyAction := r.ReadInt32()
			if keyType != 0 || keyAction != 0 {
				m.keys[i] = KeyBinding{KeyType: keyType, KeyAction: keyAction}
			}
		}
	}
}
