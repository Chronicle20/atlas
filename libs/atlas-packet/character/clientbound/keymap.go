package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterKeyMapWriter = "CharacterKeyMap"

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

func (m CharacterKeyMap) ResetToDefault() bool         { return m.resetToDefault }
func (m CharacterKeyMap) Keys() map[int32]KeyBinding   { return m.keys }
func (m CharacterKeyMap) Operation() string             { return CharacterKeyMapWriter }
func (m CharacterKeyMap) String() string {
	if m.resetToDefault {
		return "resetToDefault"
	}
	return fmt.Sprintf("keys [%d]", len(m.keys))
}

func (m CharacterKeyMap) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		if m.resetToDefault {
			w.WriteByte(1)
			return w.Bytes()
		}
		w.WriteByte(0)
		for i := int32(0); i < 90; i++ {
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

func (m *CharacterKeyMap) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		flag := r.ReadByte()
		if flag == 1 {
			m.resetToDefault = true
			return
		}
		m.keys = make(map[int32]KeyBinding)
		for i := int32(0); i < 90; i++ {
			keyType := r.ReadInt8()
			keyAction := r.ReadInt32()
			if keyType != 0 || keyAction != 0 {
				m.keys[i] = KeyBinding{KeyType: keyType, KeyAction: keyAction}
			}
		}
	}
}
