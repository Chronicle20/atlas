package character

import (
	"testing"

	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
)

func TestComputeRebalance(t *testing.T) {
	tests := []struct {
		name                        string
		str, dex, in_, luk, unalloc uint16
		targets                     []sharedsaga.RebalanceTarget
		wantStr                     uint16
		wantDex                     uint16
		wantInt                     uint16
		wantLuk                     uint16
		wantUnalloc                 uint16
		wantErr                     bool
	}{
		{
			name: "pirate reference video — DEX 20",
			str:  53, dex: 9, in_: 4, luk: 4, unalloc: 0,
			targets: []sharedsaga.RebalanceTarget{{Stat: sharedsaga.RebalanceStatDexterity, Floor: 20}},
			wantStr: 4, wantDex: 20, wantInt: 4, wantLuk: 4, wantUnalloc: 38,
		},
		{
			name: "bowman/thief/wind-archer — DEX 25",
			str:  53, dex: 9, in_: 4, luk: 4, unalloc: 0,
			targets: []sharedsaga.RebalanceTarget{{Stat: sharedsaga.RebalanceStatDexterity, Floor: 25}},
			wantStr: 4, wantDex: 25, wantInt: 4, wantLuk: 4, wantUnalloc: 33,
		},
		{
			name: "warrior/dawn-warrior — STR 35 (surplus boundary)",
			str:  53, dex: 9, in_: 4, luk: 4, unalloc: 0,
			targets: []sharedsaga.RebalanceTarget{{Stat: sharedsaga.RebalanceStatStrength, Floor: 35}},
			wantStr: 35, wantDex: 4, wantInt: 4, wantLuk: 4, wantUnalloc: 23,
		},
		{
			name: "magician/blaze-wizard — INT 20",
			str:  53, dex: 9, in_: 4, luk: 4, unalloc: 0,
			targets: []sharedsaga.RebalanceTarget{{Stat: sharedsaga.RebalanceStatIntelligence, Floor: 20}},
			wantStr: 4, wantDex: 4, wantInt: 20, wantLuk: 4, wantUnalloc: 38,
		},
		{
			name: "night-walker — LUK 25",
			str:  53, dex: 9, in_: 4, luk: 4, unalloc: 0,
			targets: []sharedsaga.RebalanceTarget{{Stat: sharedsaga.RebalanceStatLuck, Floor: 25}},
			wantStr: 4, wantDex: 4, wantInt: 4, wantLuk: 25, wantUnalloc: 33,
		},
		{
			name: "thunder-breaker — STR 20 + DEX 20 (multi-target)",
			str:  53, dex: 9, in_: 4, luk: 4, unalloc: 0,
			targets: []sharedsaga.RebalanceTarget{
				{Stat: sharedsaga.RebalanceStatStrength, Floor: 20},
				{Stat: sharedsaga.RebalanceStatDexterity, Floor: 20},
			},
			wantStr: 20, wantDex: 20, wantInt: 4, wantLuk: 4, wantUnalloc: 22,
		},
		{
			name: "existing unallocated AP carries through",
			str:  53, dex: 9, in_: 4, luk: 4, unalloc: 5,
			targets: []sharedsaga.RebalanceTarget{{Stat: sharedsaga.RebalanceStatDexterity, Floor: 20}},
			wantStr: 4, wantDex: 20, wantInt: 4, wantLuk: 4, wantUnalloc: 43,
		},
		{
			name: "insufficient AP returns error",
			str:  4, dex: 4, in_: 4, luk: 4, unalloc: 0,
			targets: []sharedsaga.RebalanceTarget{{Stat: sharedsaga.RebalanceStatDexterity, Floor: 20}},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := computeRebalance(tc.str, tc.dex, tc.in_, tc.luk, tc.unalloc, tc.targets)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (result=%+v)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Str != tc.wantStr || got.Dex != tc.wantDex || got.Int != tc.wantInt || got.Luk != tc.wantLuk || got.Unallocated != tc.wantUnalloc {
				t.Errorf("mismatch: got=%+v want STR=%d DEX=%d INT=%d LUK=%d AP=%d",
					got, tc.wantStr, tc.wantDex, tc.wantInt, tc.wantLuk, tc.wantUnalloc)
			}
		})
	}
}
