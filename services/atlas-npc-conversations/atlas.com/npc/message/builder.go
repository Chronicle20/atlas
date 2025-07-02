package message

import (
	"fmt"
	"strings"
)

type builder struct {
	b strings.Builder
}

func NewBuilder() *builder {
	return &builder{}
}

func (b *builder) AddText(message string) *builder {
	b.b.WriteString(message)
	return b
}

func (b *builder) String() string {
	return b.b.String()
}

func (b *builder) NewLine() *builder {
	b.b.WriteString("\r\n")
	return b
}

func (b *builder) OpenItem(i int) *builder {
	b.b.WriteString(fmt.Sprintf("#L%d#", i))
	return b
}

func (b *builder) BlueText() *builder {
	b.b.WriteString("#b")
	return b
}

func (b *builder) CloseItem() *builder {
	b.b.WriteString("#l")
	return b
}

func (b *builder) BlackText() *builder {
	b.b.WriteString("#k")
	return b
}

func (b *builder) PurpleText() *builder {
	b.b.WriteString("#d")
	return b
}

func (b *builder) BoldText() *builder {
	b.b.WriteString("#e")
	return b
}

func (b *builder) GreenText() *builder {
	b.b.WriteString("#g")
	return b
}

func (b *builder) RedText() *builder {
	b.b.WriteString("#r")
	return b
}

func (b *builder) NormalText() *builder {
	b.b.WriteString("#n")
	return b
}

func (b *builder) ShowMap(mapId uint32) *builder {
	b.b.WriteString(fmt.Sprintf("#m%d#", mapId))
	return b
}

func (b *builder) ShowNPC(npcId uint32) *builder {
	b.b.WriteString(fmt.Sprintf("#p%d#", npcId))
	return b
}

func (b *builder) ShowItemName1(itemId uint32) *builder {
	b.b.WriteString(fmt.Sprintf("#t%d#", itemId))
	return b
}

func (b *builder) ShowItemName2(itemId uint32) *builder {
	b.b.WriteString(fmt.Sprintf("#z%d#", itemId))
	return b
}

func (b *builder) ShowCharacterName() *builder {
	b.b.WriteString("#h #")
	return b
}

func (b *builder) ShowItemImage1(itemId uint32) *builder {
	b.b.WriteString(fmt.Sprintf("#v%d#", itemId))
	return b
}

func (b *builder) ShowItemImage2(itemId uint32) *builder {
	b.b.WriteString(fmt.Sprintf("#i%d#", itemId))
	return b
}

func (b *builder) ShowItemCount(itemId uint32) *builder {
	b.b.WriteString(fmt.Sprintf("#c%d#", itemId))
	return b
}

func (b *builder) ShowMonsterName(monsterId uint32) *builder {
	b.b.WriteString(fmt.Sprintf("#o%d#", monsterId))
	return b
}

func (b *builder) ShowSkillImage(skillId uint32) *builder {
	b.b.WriteString(fmt.Sprintf("#s%d#", skillId))
	return b
}

func (b *builder) ShowProgressBar(amount uint32) *builder {
	b.b.WriteString(fmt.Sprintf("#B%d#", amount))
	return b
}

func (b *builder) DimensionalMirrorOption(index int, selection string) *builder {
	b.b.WriteString(fmt.Sprintf("#%d# %s", index, selection))
	return b
}
