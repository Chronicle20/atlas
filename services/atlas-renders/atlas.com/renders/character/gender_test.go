package character

import "testing"

func TestResolveGender(t *testing.T) {
	cases := []struct {
		name        string
		genderParam int
		face        int
		want        int
	}{
		{"explicit-male-wins-over-female-face", GenderMale, 21000, GenderMale},
		{"explicit-female-wins-over-male-face", GenderFemale, 20000, GenderFemale},
		{"infer-female-from-21xxx", GenderUnspecified, 21000, GenderFemale},
		{"infer-male-from-20xxx", GenderUnspecified, 20000, GenderMale},
		{"infer-male-from-zero-face", GenderUnspecified, 0, GenderMale},
		{"infer-male-from-negative-face", GenderUnspecified, -5, GenderMale},
		{"infer-male-from-30xxx", GenderUnspecified, 30030, GenderMale},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ResolveGender(tc.genderParam, tc.face); got != tc.want {
				t.Fatalf("ResolveGender(%d, %d) = %d; want %d", tc.genderParam, tc.face, got, tc.want)
			}
		})
	}
}

func TestDefaultCoatPants(t *testing.T) {
	if defaultCoat(GenderMale) != DefaultCoatMale || defaultPants(GenderMale) != DefaultPantsMale {
		t.Fatalf("male defaults wrong: coat=%d pants=%d", defaultCoat(GenderMale), defaultPants(GenderMale))
	}
	if defaultCoat(GenderFemale) != DefaultCoatFemale || defaultPants(GenderFemale) != DefaultPantsFemale {
		t.Fatalf("female defaults wrong: coat=%d pants=%d", defaultCoat(GenderFemale), defaultPants(GenderFemale))
	}
}
