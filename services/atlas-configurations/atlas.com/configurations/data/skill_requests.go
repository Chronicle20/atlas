package data

import (
	"fmt"
	"strings"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	skillsPath = "data/skills"
)

type SkillRestModel struct {
	Id       uint32 `json:"-"`
	Name     string `json:"name"`
	MaxLevel uint8  `json:"maxLevel"`
}

func (s SkillRestModel) GetName() string {
	return "skills"
}

func (s SkillRestModel) GetID() string {
	return fmt.Sprint(s.Id)
}

func (s *SkillRestModel) SetID(id string) error {
	var x uint32
	_, err := fmt.Sscan(id, &x)
	if err != nil {
		return err
	}
	s.Id = x
	return nil
}

func getDataBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestSkillsByIds(ids []uint32) requests.Request[[]SkillRestModel] {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = fmt.Sprint(id)
	}
	url := fmt.Sprintf("%s%s?ids=%s", getDataBaseRequest(), skillsPath, strings.Join(parts, ","))
	return requests.GetRequest[[]SkillRestModel](url)
}
