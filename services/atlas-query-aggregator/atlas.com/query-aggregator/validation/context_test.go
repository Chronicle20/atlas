package validation

import (
	"atlas-query-aggregator/character"
	"testing"
)

func TestValidationContext_MaxPetClosenessForTemplates(t *testing.T) {
	char := character.NewModelBuilder().SetId(123).Build()
	ctx := NewValidationContextBuilder(char).
		SetSpawnedPets([]SpawnedPet{
			{TemplateId: 5000029, Closeness: 1700},
			{TemplateId: 5000048, Closeness: 50},
		}).
		Build()

	if got := ctx.MaxPetClosenessForTemplates([]uint32{5000029}); got != 1700 {
		t.Fatalf("MaxPetClosenessForTemplates([5000029]) = %d, want 1700", got)
	}
	if got := ctx.MaxPetClosenessForTemplates([]uint32{5000048}); got != 50 {
		t.Fatalf("MaxPetClosenessForTemplates([5000048]) = %d, want 50", got)
	}
	if got := ctx.MaxPetClosenessForTemplates([]uint32{5000030}); got != 0 {
		t.Fatalf("MaxPetClosenessForTemplates([5000030]) = %d, want 0", got)
	}
	if got := ctx.MaxPetClosenessForTemplates([]uint32{5000029, 5000048}); got != 1700 {
		t.Fatalf("MaxPetClosenessForTemplates([...]) = %d, want 1700", got)
	}
}
