package face

import (
	"atlas-data/xml"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

func parseFaceId(filePath string) (uint32, error) {
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

		faceId, err := parseFaceId(exml.Name)
		if err != nil {
			return model.ErrorProvider[RestModel](err)
		}
		l.Debugf("Processing face [%d].", faceId)

		m := RestModel{Id: faceId}

		i, err := exml.ChildByName("info")
		if err == nil && i != nil {
			m.Cash = i.GetBool("cash", false)
		}

		return model.FixedProvider(m)
	}
}
