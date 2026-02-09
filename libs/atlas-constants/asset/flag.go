package asset

type Flag uint16

const (
	FlagLock             Flag = 0x01
	FlagSpikes           Flag = 0x02
	FlagKarmaUse         Flag = 0x02
	FlagCold             Flag = 0x04
	FlagUntradeable      Flag = 0x08
	FlagKarmaEquip       Flag = 0x10
	FlagSandbox          Flag = 0x40
	FlagPetCome          Flag = 0x80
	FlagAccountSharing   Flag = 0x100
	FlagMergeUntradeable Flag = 0x200
)

func HasFlag(flags uint16, f Flag) bool {
	return flags&uint16(f) != 0
}

func SetFlag(flags uint16, f Flag) uint16 {
	return flags | uint16(f)
}

func ClearFlag(flags uint16, f Flag) uint16 {
	return flags &^ uint16(f)
}
