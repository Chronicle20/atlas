package characterimage

import (
	"fmt"
	"os"
	"path/filepath"
)

var supportedStances = map[string]struct{}{
	"stand1": {}, "stand2": {}, "walk1": {}, "alert": {}, "jump": {},
}

// SupportedStances is the canonical list returned in 400 error meta.
func SupportedStances() []string {
	return []string{"stand1", "stand2", "walk1", "alert", "jump"}
}

// ValidateStance returns ErrInvalidStance if `s` is not in scope.
func ValidateStance(s string) error {
	if _, ok := supportedStances[s]; ok {
		return nil
	}
	return fmt.Errorf("%w: %s", ErrInvalidStance, s)
}

// ValidateFrame checks whether {templateId}/{stance}/{frame} exists in the
// extract. Used against the body skin img only (other parts inherit).
func ValidateFrame(assetsRoot, templateId, stance string, frame int) error {
	path := filepath.Join(assetsRoot, "character-parts", templateId, stance, fmt.Sprintf("%d", frame))
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %s/%s/%d", ErrFrameOutOfRange, templateId, stance, frame)
		}
		return fmt.Errorf("stat frame: %w", err)
	}
	return nil
}
