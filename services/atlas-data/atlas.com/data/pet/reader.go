package pet

import (
	"atlas-data/xml"
	"context"
	"fmt"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"path/filepath"
	"strconv"
	"strings"
)

func parsePetId(filePath string) (uint32, error) {
	baseName := filepath.Base(filePath)
	if !strings.HasSuffix(baseName, ".img.xml") {
		return 0, fmt.Errorf("file does not match expected format: %s", filePath)
	}
	idStr := strings.TrimSuffix(baseName, ".img.xml")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, err
	}
	return uint32(id), nil
}

func ReadFromFile(l logrus.FieldLogger) func(ctx context.Context) func(path string) model.Provider[Model] {
	return func(ctx context.Context) func(path string) model.Provider[Model] {
		return func(path string) model.Provider[Model] {
			petId, err := parsePetId(path)
			if err != nil {
				return model.ErrorProvider[Model](err)
			}
			l.Debugf("Processing pet [%d].", petId)

			exml, err := xml.Read(path)
			if err != nil {
				return model.ErrorProvider[Model](err)
			}

			i, err := exml.ChildByName("info")
			if err != nil {
				return model.ErrorProvider[Model](err)
			}

			m := Model{id: petId}
			m.hungry = uint32(i.GetIntegerWithDefault("hungry", 0))
			m.cash = i.GetBool("cash", true)
			m.life = uint32(i.GetIntegerWithDefault("life", 0))

			it, err := exml.ChildByName("interact")
			if err != nil {
				return model.ErrorProvider[Model](err)
			}

			for _, s := range it.ChildNodes {
				var sid int
				sid, err = strconv.Atoi(s.Name)
				if err != nil {
					return model.ErrorProvider[Model](err)
				}
				sm := SkillModel{
					id:          fmt.Sprintf("%d-%d", petId, sid),
					increase:    uint16(s.GetIntegerWithDefault("inc", 0)),
					probability: uint16(s.GetIntegerWithDefault("prob", 0)),
				}
				m.skills = append(m.skills, sm)
			}
			return model.FixedProvider(m)
		}
	}
}
