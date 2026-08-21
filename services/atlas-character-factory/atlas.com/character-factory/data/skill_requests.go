package data

import (
	"context"
	"fmt"
	"strings"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const skillsPath = "data/skills"

// SkillEffectRestModel carries only the field Maple Life's HP/MP arithmetic
// needs from atlas-data's per-level skill effect resource
// (services/atlas-data/atlas.com/data/skill/effect/rest.go): the level-up
// HP/MP gain bonus atlas-character's own resolveHPMPGainParams reads as
// se.X() (services/atlas-character/atlas.com/character/character/processor.go:1810,1817).
type SkillEffectRestModel struct {
	X int16 `json:"x"`
}

type SkillRestModel struct {
	Id       uint32 `json:"-"`
	Name     string `json:"name"`
	MaxLevel uint8  `json:"maxLevel"`
	// Effects is one entry per skill level, index 0 == level 1 (matching
	// atlas-data's own GetEffect(level) convention: Effects()[level-1]).
	Effects []SkillEffectRestModel `json:"effects"`
}

func (s SkillRestModel) GetName() string { return "skills" }
func (s SkillRestModel) GetID() string   { return fmt.Sprint(s.Id) }
func (s *SkillRestModel) SetID(id string) error {
	var x uint64
	if _, err := fmt.Sscan(id, &x); err != nil {
		return err
	}
	s.Id = uint32(x)
	return nil
}

func getDataBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "DATA")
}

func requestSkillsByIds(ctx context.Context, ids []uint32) requests.Request[[]SkillRestModel] {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = fmt.Sprint(id)
	}
	root, err := getDataBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[[]SkillRestModel](err)
	}
	url := fmt.Sprintf("%s%s?ids=%s", root, skillsPath, strings.Join(parts, ","))
	return requests.GetRequest[[]SkillRestModel](url)
}
