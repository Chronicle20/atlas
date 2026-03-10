package writer

import (
	"atlas-channel/character/key"

	charpkt "github.com/Chronicle20/atlas-packet/character"
	"github.com/Chronicle20/atlas-socket/packet"
)

const CharacterKeyMap = "CharacterKeyMap"

func CharacterKeyMapBody(keys map[int32]key.Model) packet.Encode {
	bindings := make(map[int32]charpkt.KeyBinding)
	for k, v := range keys {
		bindings[k] = charpkt.KeyBinding{KeyType: v.Type(), KeyAction: v.Action()}
	}
	return charpkt.NewCharacterKeyMap(bindings).Encode
}

func CharacterKeyMapResetToDefaultBody() packet.Encode {
	return charpkt.NewCharacterKeyMapResetToDefault().Encode
}
