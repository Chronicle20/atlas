package hair

import (
	"atlas-data/xml"
	"fmt"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"path/filepath"
	"strconv"
	"strings"
)

func parseHairId(filePath string) (uint32, error) {
	baseName := filepath.Base(filePath)
	if !strings.HasSuffix(baseName, ".img") {
		return 0, fmt.Errorf("file does not match expected format: %s", filePath)
	}
	idStr := strings.TrimSuffix(baseName, ".img")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, err
	}
	return uint32(id), nil
}

func Read(l logrus.FieldLogger) func(np model.Provider[xml.Node]) model.Provider[RestModel] {
	return func(np model.Provider[xml.Node]) model.Provider[RestModel] {
		exml, err := np()
		if err != nil {
			return model.ErrorProvider[RestModel](err)
		}

		hairId, err := parseHairId(exml.Name)
		if err != nil {
			return model.ErrorProvider[RestModel](err)
		}
		l.Debugf("Processing hair [%d].", hairId)

		m := RestModel{Id: hairId}

		i, err := exml.ChildByName("info")
		if err == nil && i != nil {
			m.Cash = i.GetBool("cash", false)
		}

		return model.FixedProvider(m)
	}
}
