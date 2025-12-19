package templates

import (
	"atlas-data/xml"
	"github.com/sirupsen/logrus/hooks/test"
	"testing"
)

const testXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="MakeCharInfo.img">
    <imgdir name="Info">
        <imgdir name="CharMale">
            <imgdir name="0">
                <int name="0" value="20000"/>
                <int name="1" value="20001"/>
                <int name="2" value="20002"/>
            </imgdir>
            <imgdir name="1">
                <int name="0" value="30030"/>
                <int name="1" value="30020"/>
                <int name="2" value="30000"/>
            </imgdir>
            <imgdir name="2">
                <int name="0" value="0"/>
                <int name="1" value="7"/>
                <int name="2" value="3"/>
                <int name="3" value="2"/>
            </imgdir>
            <imgdir name="3">
                <int name="0" value="0"/>
                <int name="1" value="1"/>
                <int name="2" value="2"/>
                <int name="3" value="3"/>
            </imgdir>
            <imgdir name="4">
                <int name="0" value="1040002"/>
                <int name="1" value="1040006"/>
                <int name="2" value="1040010"/>
            </imgdir>
            <imgdir name="5">
                <int name="0" value="1060002"/>
                <int name="1" value="1060006"/>
            </imgdir>
            <imgdir name="6">
                <int name="0" value="1072001"/>
                <int name="1" value="1072005"/>
                <int name="2" value="1072037"/>
                <int name="3" value="1072038"/>
            </imgdir>
            <imgdir name="7">
                <int name="0" value="1302000"/>
                <int name="1" value="1322005"/>
                <int name="2" value="1312004"/>
            </imgdir>
        </imgdir>
        <imgdir name="CharFemale">
            <imgdir name="0">
                <int name="0" value="21000"/>
                <int name="1" value="21001"/>
                <int name="2" value="21002"/>
            </imgdir>
            <imgdir name="1">
                <int name="0" value="31000"/>
                <int name="1" value="31040"/>
                <int name="2" value="31050"/>
            </imgdir>
            <imgdir name="2">
                <int name="0" value="0"/>
                <int name="1" value="7"/>
                <int name="2" value="3"/>
                <int name="3" value="2"/>
            </imgdir>
            <imgdir name="3">
                <int name="0" value="0"/>
                <int name="1" value="1"/>
                <int name="2" value="2"/>
                <int name="3" value="3"/>
            </imgdir>
            <imgdir name="4">
                <int name="0" value="1041002"/>
                <int name="1" value="1041006"/>
                <int name="2" value="1041010"/>
                <int name="3" value="1041011"/>
            </imgdir>
            <imgdir name="5">
                <int name="0" value="1061002"/>
                <int name="1" value="1061008"/>
            </imgdir>
            <imgdir name="6">
                <int name="0" value="1072001"/>
                <int name="1" value="1072005"/>
                <int name="2" value="1072037"/>
                <int name="3" value="1072038"/>
            </imgdir>
            <imgdir name="7">
                <int name="0" value="1302000"/>
                <int name="1" value="1322005"/>
                <int name="2" value="1312004"/>
            </imgdir>
        </imgdir>
    </imgdir>
    <imgdir name="PremiumCharMale">
        <imgdir name="0">
            <int name="0" value="20000"/>
            <int name="1" value="20001"/>
            <int name="2" value="20002"/>
        </imgdir>
        <imgdir name="1">
            <int name="0" value="30030"/>
            <int name="1" value="30020"/>
            <int name="2" value="30000"/>
        </imgdir>
        <imgdir name="2">
            <int name="0" value="0"/>
            <int name="1" value="7"/>
            <int name="2" value="3"/>
            <int name="3" value="2"/>
        </imgdir>
        <imgdir name="3">
            <int name="0" value="0"/>
            <int name="1" value="1"/>
            <int name="2" value="2"/>
            <int name="3" value="3"/>
        </imgdir>
        <imgdir name="4">
            <int name="0" value="1040002"/>
            <int name="1" value="1040006"/>
            <int name="2" value="1040010"/>
        </imgdir>
        <imgdir name="5">
            <int name="0" value="1060002"/>
            <int name="1" value="1060006"/>
        </imgdir>
        <imgdir name="6">
            <int name="0" value="1072001"/>
            <int name="1" value="1072005"/>
            <int name="2" value="1072037"/>
            <int name="3" value="1072038"/>
        </imgdir>
        <imgdir name="7">
            <int name="0" value="1302000"/>
            <int name="1" value="1322005"/>
            <int name="2" value="1312004"/>
        </imgdir>
    </imgdir>
</imgdir>`

func Identity[M any](m M) M {
	return m
}

func TestRead(t *testing.T) {
	l, _ := test.NewNullLogger()

	provider := xml.FromByteArrayProvider([]byte(testXML))
	rms := Read(l)(provider)
	res, err := rms()
	if err != nil {
		t.Fatal(err)
	}

	// We should have 3 character types: CharMale, CharFemale, and PremiumCharMale
	if len(res) != 3 {
		t.Fatalf("len(res) = %d, want 3", len(res))
	}

	// Check CharMale
	var charMale *RestModel
	for i := range res {
		if res[i].CharacterType == "CharMale" {
			charMale = &res[i]
			break
		}
	}
	if charMale == nil {
		t.Fatalf("CharMale not found")
	}

	// Check faces
	if len(charMale.Faces) != 3 {
		t.Fatalf("len(charMale.Faces) = %d, want 3", len(charMale.Faces))
	}
	if charMale.Faces[0] != 20000 {
		t.Fatalf("charMale.Faces[0] = %d, want 20000", charMale.Faces[0])
	}
	if charMale.Faces[1] != 20001 {
		t.Fatalf("charMale.Faces[1] = %d, want 20001", charMale.Faces[1])
	}
	if charMale.Faces[2] != 20002 {
		t.Fatalf("charMale.Faces[2] = %d, want 20002", charMale.Faces[2])
	}

	// Check hair styles
	if len(charMale.HairStyles) != 3 {
		t.Fatalf("len(charMale.HairStyles) = %d, want 3", len(charMale.HairStyles))
	}
	if charMale.HairStyles[0] != 30030 {
		t.Fatalf("charMale.HairStyles[0] = %d, want 30030", charMale.HairStyles[0])
	}
	if charMale.HairStyles[1] != 30020 {
		t.Fatalf("charMale.HairStyles[1] = %d, want 30020", charMale.HairStyles[1])
	}
	if charMale.HairStyles[2] != 30000 {
		t.Fatalf("charMale.HairStyles[2] = %d, want 30000", charMale.HairStyles[2])
	}

	// Check hair colors
	if len(charMale.HairColors) != 4 {
		t.Fatalf("len(charMale.HairColors) = %d, want 4", len(charMale.HairColors))
	}
	if charMale.HairColors[0] != 0 {
		t.Fatalf("charMale.HairColors[0] = %d, want 0", charMale.HairColors[0])
	}
	if charMale.HairColors[1] != 7 {
		t.Fatalf("charMale.HairColors[1] = %d, want 7", charMale.HairColors[1])
	}
	if charMale.HairColors[2] != 3 {
		t.Fatalf("charMale.HairColors[2] = %d, want 3", charMale.HairColors[2])
	}
	if charMale.HairColors[3] != 2 {
		t.Fatalf("charMale.HairColors[3] = %d, want 2", charMale.HairColors[3])
	}

	// Check skin colors
	if len(charMale.SkinColors) != 4 {
		t.Fatalf("len(charMale.SkinColors) = %d, want 4", len(charMale.SkinColors))
	}
	if charMale.SkinColors[0] != 0 {
		t.Fatalf("charMale.SkinColors[0] = %d, want 0", charMale.SkinColors[0])
	}
	if charMale.SkinColors[1] != 1 {
		t.Fatalf("charMale.SkinColors[1] = %d, want 1", charMale.SkinColors[1])
	}
	if charMale.SkinColors[2] != 2 {
		t.Fatalf("charMale.SkinColors[2] = %d, want 2", charMale.SkinColors[2])
	}
	if charMale.SkinColors[3] != 3 {
		t.Fatalf("charMale.SkinColors[3] = %d, want 3", charMale.SkinColors[3])
	}

	// Check tops
	if len(charMale.Tops) != 3 {
		t.Fatalf("len(charMale.Tops) = %d, want 3", len(charMale.Tops))
	}
	if charMale.Tops[0] != 1040002 {
		t.Fatalf("charMale.Tops[0] = %d, want 1040002", charMale.Tops[0])
	}
	if charMale.Tops[1] != 1040006 {
		t.Fatalf("charMale.Tops[1] = %d, want 1040006", charMale.Tops[1])
	}
	if charMale.Tops[2] != 1040010 {
		t.Fatalf("charMale.Tops[2] = %d, want 1040010", charMale.Tops[2])
	}

	// Check bottoms
	if len(charMale.Bottoms) != 2 {
		t.Fatalf("len(charMale.Bottoms) = %d, want 2", len(charMale.Bottoms))
	}
	if charMale.Bottoms[0] != 1060002 {
		t.Fatalf("charMale.Bottoms[0] = %d, want 1060002", charMale.Bottoms[0])
	}
	if charMale.Bottoms[1] != 1060006 {
		t.Fatalf("charMale.Bottoms[1] = %d, want 1060006", charMale.Bottoms[1])
	}

	// Check shoes
	if len(charMale.Shoes) != 4 {
		t.Fatalf("len(charMale.Shoes) = %d, want 4", len(charMale.Shoes))
	}
	if charMale.Shoes[0] != 1072001 {
		t.Fatalf("charMale.Shoes[0] = %d, want 1072001", charMale.Shoes[0])
	}
	if charMale.Shoes[1] != 1072005 {
		t.Fatalf("charMale.Shoes[1] = %d, want 1072005", charMale.Shoes[1])
	}
	if charMale.Shoes[2] != 1072037 {
		t.Fatalf("charMale.Shoes[2] = %d, want 1072037", charMale.Shoes[2])
	}
	if charMale.Shoes[3] != 1072038 {
		t.Fatalf("charMale.Shoes[3] = %d, want 1072038", charMale.Shoes[3])
	}

	// Check weapons
	if len(charMale.Weapons) != 3 {
		t.Fatalf("len(charMale.Weapons) = %d, want 3", len(charMale.Weapons))
	}
	if charMale.Weapons[0] != 1302000 {
		t.Fatalf("charMale.Weapons[0] = %d, want 1302000", charMale.Weapons[0])
	}
	if charMale.Weapons[1] != 1322005 {
		t.Fatalf("charMale.Weapons[1] = %d, want 1322005", charMale.Weapons[1])
	}
	if charMale.Weapons[2] != 1312004 {
		t.Fatalf("charMale.Weapons[2] = %d, want 1312004", charMale.Weapons[2])
	}
}

func TestProcessCharacterNode(t *testing.T) {
	// Create a test node
	node := &xml.Node{
		Name: "TestCharacter",
		ChildNodes: []xml.Node{
			{
				Name: "0",
				IntegerNodes: []xml.IntegerNode{
					{Name: "0", Value: "20000"},
					{Name: "1", Value: "20001"},
				},
			},
			{
				Name: "1",
				IntegerNodes: []xml.IntegerNode{
					{Name: "0", Value: "30030"},
					{Name: "1", Value: "30020"},
				},
			},
			{
				Name: "2",
				IntegerNodes: []xml.IntegerNode{
					{Name: "0", Value: "0"},
					{Name: "1", Value: "7"},
				},
			},
			{
				Name: "3",
				IntegerNodes: []xml.IntegerNode{
					{Name: "0", Value: "0"},
					{Name: "1", Value: "1"},
				},
			},
			{
				Name: "4",
				IntegerNodes: []xml.IntegerNode{
					{Name: "0", Value: "1040002"},
					{Name: "1", Value: "1040006"},
				},
			},
			{
				Name: "5",
				IntegerNodes: []xml.IntegerNode{
					{Name: "0", Value: "1060002"},
					{Name: "1", Value: "1060006"},
				},
			},
			{
				Name: "6",
				IntegerNodes: []xml.IntegerNode{
					{Name: "0", Value: "1072001"},
					{Name: "1", Value: "1072005"},
				},
			},
			{
				Name: "7",
				IntegerNodes: []xml.IntegerNode{
					{Name: "0", Value: "1302000"},
					{Name: "1", Value: "1322005"},
				},
			},
		},
	}

	// Process the node
	model := processCharacterNode(0, node)

	// Check the model
	if model.CharacterType != "TestCharacter" {
		t.Fatalf("model.CharacterType = %s, want TestCharacter", model.CharacterType)
	}

	// Check faces
	if len(model.Faces) != 2 {
		t.Fatalf("len(model.Faces) = %d, want 2", len(model.Faces))
	}
	if model.Faces[0] != 20000 {
		t.Fatalf("model.Faces[0] = %d, want 20000", model.Faces[0])
	}
	if model.Faces[1] != 20001 {
		t.Fatalf("model.Faces[1] = %d, want 20001", model.Faces[1])
	}

	// Check hair styles
	if len(model.HairStyles) != 2 {
		t.Fatalf("len(model.HairStyles) = %d, want 2", len(model.HairStyles))
	}
	if model.HairStyles[0] != 30030 {
		t.Fatalf("model.HairStyles[0] = %d, want 30030", model.HairStyles[0])
	}
	if model.HairStyles[1] != 30020 {
		t.Fatalf("model.HairStyles[1] = %d, want 30020", model.HairStyles[1])
	}

	// Check hair colors
	if len(model.HairColors) != 2 {
		t.Fatalf("len(model.HairColors) = %d, want 2", len(model.HairColors))
	}
	if model.HairColors[0] != 0 {
		t.Fatalf("model.HairColors[0] = %d, want 0", model.HairColors[0])
	}
	if model.HairColors[1] != 7 {
		t.Fatalf("model.HairColors[1] = %d, want 7", model.HairColors[1])
	}

	// Check skin colors
	if len(model.SkinColors) != 2 {
		t.Fatalf("len(model.SkinColors) = %d, want 2", len(model.SkinColors))
	}
	if model.SkinColors[0] != 0 {
		t.Fatalf("model.SkinColors[0] = %d, want 0", model.SkinColors[0])
	}
	if model.SkinColors[1] != 1 {
		t.Fatalf("model.SkinColors[1] = %d, want 1", model.SkinColors[1])
	}

	// Check tops
	if len(model.Tops) != 2 {
		t.Fatalf("len(model.Tops) = %d, want 2", len(model.Tops))
	}
	if model.Tops[0] != 1040002 {
		t.Fatalf("model.Tops[0] = %d, want 1040002", model.Tops[0])
	}
	if model.Tops[1] != 1040006 {
		t.Fatalf("model.Tops[1] = %d, want 1040006", model.Tops[1])
	}

	// Check bottoms
	if len(model.Bottoms) != 2 {
		t.Fatalf("len(model.Bottoms) = %d, want 2", len(model.Bottoms))
	}
	if model.Bottoms[0] != 1060002 {
		t.Fatalf("model.Bottoms[0] = %d, want 1060002", model.Bottoms[0])
	}
	if model.Bottoms[1] != 1060006 {
		t.Fatalf("model.Bottoms[1] = %d, want 1060006", model.Bottoms[1])
	}

	// Check shoes
	if len(model.Shoes) != 2 {
		t.Fatalf("len(model.Shoes) = %d, want 2", len(model.Shoes))
	}
	if model.Shoes[0] != 1072001 {
		t.Fatalf("model.Shoes[0] = %d, want 1072001", model.Shoes[0])
	}
	if model.Shoes[1] != 1072005 {
		t.Fatalf("model.Shoes[1] = %d, want 1072005", model.Shoes[1])
	}

	// Check weapons
	if len(model.Weapons) != 2 {
		t.Fatalf("len(model.Weapons) = %d, want 2", len(model.Weapons))
	}
	if model.Weapons[0] != 1302000 {
		t.Fatalf("model.Weapons[0] = %d, want 1302000", model.Weapons[0])
	}
	if model.Weapons[1] != 1322005 {
		t.Fatalf("model.Weapons[1] = %d, want 1322005", model.Weapons[1])
	}
}
