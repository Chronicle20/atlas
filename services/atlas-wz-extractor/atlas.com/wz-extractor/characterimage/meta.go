package characterimage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Vec is a 2D coordinate pair used by both origin and joint maps.
type Vec struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// PartMeta is the JSON sidecar shape the extractor emits next to each PNG.
type PartMeta struct {
	Origin Vec            `json:"origin"`
	Map    map[string]Vec `json:"map"`
	Z      string         `json:"z"`
	Group  string         `json:"group"`
	Delay  int            `json:"delay"`
	Face   int            `json:"face"`
}

// TemplateInfo mirrors {templateId}/info.json on disk.
type TemplateInfo struct {
	Islot string `json:"islot"`
	Vslot string `json:"vslot"`
	Cash  int    `json:"cash"`
}

// LoadZmap reads character-meta/zmap.json. The slice is the render order.
func LoadZmap(assetsRoot string) ([]string, error) {
	b, err := os.ReadFile(filepath.Join(assetsRoot, "character-meta", "zmap.json"))
	if err != nil {
		return nil, fmt.Errorf("%w: zmap: %v", ErrAssetsMissing, err)
	}
	var out []string
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("parse zmap: %w", err)
	}
	return out, nil
}

// LoadSmap reads character-meta/smap.json: layer-string -> slot codes.
func LoadSmap(assetsRoot string) (map[string]string, error) {
	b, err := os.ReadFile(filepath.Join(assetsRoot, "character-meta", "smap.json"))
	if err != nil {
		return nil, fmt.Errorf("%w: smap: %v", ErrAssetsMissing, err)
	}
	out := map[string]string{}
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("parse smap: %w", err)
	}
	return out, nil
}

// LoadInfo reads character-parts/{templateId}/info.json.
func LoadInfo(assetsRoot, templateId string) (TemplateInfo, error) {
	b, err := os.ReadFile(filepath.Join(assetsRoot, "character-parts", templateId, "info.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return TemplateInfo{}, fmt.Errorf("%w: %s", ErrUnknownTemplateId, templateId)
		}
		return TemplateInfo{}, fmt.Errorf("read info %s: %w", templateId, err)
	}
	var ti TemplateInfo
	if err := json.Unmarshal(b, &ti); err != nil {
		return TemplateInfo{}, fmt.Errorf("parse info %s: %w", templateId, err)
	}
	return ti, nil
}

// LoadPartMeta reads {templateId}/{stance}/{frame}/{part}.json.
func LoadPartMeta(assetsRoot, templateId, stance string, frame int, part string) (PartMeta, error) {
	path := filepath.Join(
		assetsRoot, "character-parts", templateId,
		stance, fmt.Sprintf("%d", frame), part+".json",
	)
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return PartMeta{}, fmt.Errorf("%w: %s", ErrAssetsMissing, path)
		}
		return PartMeta{}, fmt.Errorf("read part meta %s: %w", path, err)
	}
	var pm PartMeta
	if err := json.Unmarshal(b, &pm); err != nil {
		return PartMeta{}, fmt.Errorf("parse part meta %s: %w", path, err)
	}
	return pm, nil
}
