package cosmetic

type CosmeticType string

const (
	CosmeticTypeHair CosmeticType = "hair"
	CosmeticTypeFace CosmeticType = "face"
	CosmeticTypeSkin CosmeticType = "skin"
)

type GenerationMode string

const (
	ModePreserveColor GenerationMode = "preserveColor"
	ModeColorVariants GenerationMode = "colorVariants"
	ModeBaseOnly      GenerationMode = "baseOnly"
)

// CharacterAppearance represents a character's cosmetic appearance attributes
type CharacterAppearance struct {
	characterId uint32
	gender      byte
	hair        uint32
	face        uint32
	skinColor   byte
}

// NewCharacterAppearance creates a new CharacterAppearance instance
func NewCharacterAppearance(characterId uint32, gender byte, hair uint32, face uint32, skinColor byte) CharacterAppearance {
	return CharacterAppearance{
		characterId: characterId,
		gender:      gender,
		hair:        hair,
		face:        face,
		skinColor:   skinColor,
	}
}

// CharacterId returns the character ID
func (c CharacterAppearance) CharacterId() uint32 {
	return c.characterId
}

// Gender returns the character's gender (0 = male, 1 = female)
func (c CharacterAppearance) Gender() byte {
	return c.gender
}

// Hair returns the full hair ID (base * 10 + color)
func (c CharacterAppearance) Hair() uint32 {
	return c.hair
}

// Face returns the face ID
func (c CharacterAppearance) Face() uint32 {
	return c.face
}

// SkinColor returns the skin color ID
func (c CharacterAppearance) SkinColor() byte {
	return c.skinColor
}

// HairBase returns the base style from the hair ID (hairId / 10)
// Example: 30067 -> 3006
func (c CharacterAppearance) HairBase() uint32 {
	return c.hair / 10
}

// HairColor returns the color variant from the hair ID (hairId % 10)
// Example: 30067 -> 7
func (c CharacterAppearance) HairColor() byte {
	return byte(c.hair % 10)
}

// IsMale returns true if the character is male (gender == 0)
func (c CharacterAppearance) IsMale() bool {
	return c.gender == 0
}

// IsFemale returns true if the character is female (gender == 1)
func (c CharacterAppearance) IsFemale() bool {
	return c.gender == 1
}

// FaceBase returns the base style from the face ID
// Face structure: 2XYZZ where X=gender (0=male, 1=female), YZZ=style+color
// Base face = face % 100 (extracts the base face style number 0-99)
// Example: 20401 -> base 1 (face style 1, color 4)
// Example: 21307 -> base 7 (face style 7, color 3)
func (c CharacterAppearance) FaceBase() uint32 {
	return c.face % 100
}

// FaceColor returns the color variant from the face ID
// Color is encoded in the hundreds digit: (face / 100) % 10
// Example: 20401 -> color 4
// Example: 21307 -> color 3
// Example: 20001 -> color 0
func (c CharacterAppearance) FaceColor() byte {
	return byte((c.face / 100) % 10)
}
