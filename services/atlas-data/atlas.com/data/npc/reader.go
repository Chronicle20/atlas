package npc

import (
	"atlas-data/xml"
	"context"
	"fmt"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"path/filepath"
	"strconv"
	"strings"
)

func parseNpcId(filePath string) (uint32, error) {
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

func Read(l logrus.FieldLogger) func(ctx context.Context) func(np model.Provider[xml.Node]) model.Provider[RestModel] {
	return func(ctx context.Context) func(np model.Provider[xml.Node]) model.Provider[RestModel] {
		t := tenant.MustFromContext(ctx)
		return func(np model.Provider[xml.Node]) model.Provider[RestModel] {
			exml, err := np()
			if err != nil {
				return model.ErrorProvider[RestModel](err)
			}

			npcId, err := parseNpcId(exml.Name)
			if err != nil {
				return model.ErrorProvider[RestModel](err)
			}
			l.Debugf("Processing NPC [%d].", npcId)

			m := &RestModel{Id: npcId}

			// Get NPC name from string registry
			ns, err := GetNpcStringRegistry().Get(t, strconv.Itoa(int(npcId)))
			if err != nil {
				// NPC might not have a name in the string registry, use empty string
				l.Debugf("NPC [%d] not found in string registry, using empty name.", npcId)
				m.Name = ""
			} else {
				m.Name = ns.Name()
			}

			// Parse info section if it exists
			node, err := exml.ChildByName("info")
			if err != nil {
				// No info section, return with defaults
				return model.FixedProvider(*m)
			}

			// Parse storage-related fields
			m.TrunkPut = node.GetIntegerWithDefault("trunkPut", 0)
			m.TrunkGet = node.GetIntegerWithDefault("trunkGet", 0)
			m.Storebank = node.GetIntegerWithDefault("storebank", 0) == 1
			m.HideName = node.GetIntegerWithDefault("hideName", 0) == 1

			// Parse dialog collision box
			m.DcLeft = node.GetIntegerWithDefault("dcLeft", 0)
			m.DcRight = node.GetIntegerWithDefault("dcRight", 0)
			m.DcTop = node.GetIntegerWithDefault("dcTop", 0)
			m.DcBottom = node.GetIntegerWithDefault("dcBottom", 0)

			return model.FixedProvider(*m)
		}
	}
}
