package templates

import (
	"atlas-data/xml"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"strconv"
)

func Read(l logrus.FieldLogger) func(np model.Provider[xml.Node]) model.Provider[[]RestModel] {
	return func(np model.Provider[xml.Node]) model.Provider[[]RestModel] {
		exml, err := np()
		if err != nil {
			return model.ErrorProvider[[]RestModel](err)
		}

		res := make([]RestModel, 0)

		index := uint32(0)
		// Process Info/CharMale and Info/CharFemale
		infoNode, err := exml.ChildByName("Info")
		if err == nil {
			for _, charNode := range infoNode.ChildNodes {
				model := processCharacterNode(index, &charNode)
				index++
				res = append(res, model)
			}
		}

		// Process all character nodes at the root level except "Info" and "Name"
		for _, charNode := range exml.ChildNodes {
			if charNode.Name != "Info" && charNode.Name != "Name" {
				model := processCharacterNode(index, &charNode)
				index++
				res = append(res, model)
			}
		}

		return model.FixedProvider(res)
	}
}

func processCharacterNode(index uint32, charNode *xml.Node) RestModel {
	model := RestModel{
		Id:            index,
		CharacterType: charNode.Name,
		Faces:         make([]uint32, 0),
		HairStyles:    make([]uint32, 0),
		HairColors:    make([]uint32, 0),
		SkinColors:    make([]uint32, 0),
		Tops:          make([]uint32, 0),
		Bottoms:       make([]uint32, 0),
		Shoes:         make([]uint32, 0),
		Weapons:       make([]uint32, 0),
	}

	// Process each attribute category
	for _, categoryNode := range charNode.ChildNodes {
		switch categoryNode.Name {
		case "0": // Faces
			for _, valueNode := range categoryNode.IntegerNodes {
				if val, err := strconv.ParseUint(valueNode.Value, 10, 32); err == nil {
					model.Faces = append(model.Faces, uint32(val))
				}
			}
		case "1": // Hair styles
			for _, valueNode := range categoryNode.IntegerNodes {
				if val, err := strconv.ParseUint(valueNode.Value, 10, 32); err == nil {
					model.HairStyles = append(model.HairStyles, uint32(val))
				}
			}
		case "2": // Hair colors
			for _, valueNode := range categoryNode.IntegerNodes {
				if val, err := strconv.ParseUint(valueNode.Value, 10, 32); err == nil {
					model.HairColors = append(model.HairColors, uint32(val))
				}
			}
		case "3": // Skin colors
			for _, valueNode := range categoryNode.IntegerNodes {
				if val, err := strconv.ParseUint(valueNode.Value, 10, 32); err == nil {
					model.SkinColors = append(model.SkinColors, uint32(val))
				}
			}
		case "4": // Tops
			for _, valueNode := range categoryNode.IntegerNodes {
				if val, err := strconv.ParseUint(valueNode.Value, 10, 32); err == nil {
					model.Tops = append(model.Tops, uint32(val))
				}
			}
		case "5": // Bottoms
			for _, valueNode := range categoryNode.IntegerNodes {
				if val, err := strconv.ParseUint(valueNode.Value, 10, 32); err == nil {
					model.Bottoms = append(model.Bottoms, uint32(val))
				}
			}
		case "6": // Shoes
			for _, valueNode := range categoryNode.IntegerNodes {
				if val, err := strconv.ParseUint(valueNode.Value, 10, 32); err == nil {
					model.Shoes = append(model.Shoes, uint32(val))
				}
			}
		case "7": // Weapons
			for _, valueNode := range categoryNode.IntegerNodes {
				if val, err := strconv.ParseUint(valueNode.Value, 10, 32); err == nil {
					model.Weapons = append(model.Weapons, uint32(val))
				}
			}
		}
	}

	return model
}
