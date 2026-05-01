package characterimage

import "fmt"

// internalSkinToWZ maps the atlas-ui internal 0..10 to the Character.wz id
// 2000..2013 (non-contiguous range). Source of truth lives in this file.
var internalSkinToWZ = map[int]int{
	0:  2000,
	1:  2001,
	2:  2002,
	3:  2003,
	4:  2004,
	5:  2005,
	6:  2009,
	7:  2010,
	8:  2011,
	9:  2012,
	10: 2013,
}

// MapInternalSkin returns the WZ skin id for an internal 0..10 value.
func MapInternalSkin(internal int) (int, error) {
	if wz, ok := internalSkinToWZ[internal]; ok {
		return wz, nil
	}
	return 0, fmt.Errorf("internal skin %d out of range 0..10", internal)
}
